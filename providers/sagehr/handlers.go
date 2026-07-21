package sagehr

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/amp-labs/connectors/internal/simultaneously"
	"github.com/spyzhov/ajson"
)

const (
	// maxConcurrentChildFetch bounds parallel per-parent fan-out requests
	// (employees/compensations, employees/custom-fields,
	// employees/leave-management/balances, recruitment/positions/applicants).
	// Sage HR does not publish a rate limit (see README), so this is a
	// conservative default rather than a value derived from documented limits.
	maxConcurrentChildFetch = 4

	// maxChildPagesPerParent bounds how many pages of a single parent's child
	// collection we will walk before giving up, to keep one Read call finite.
	maxChildPagesPerParent = 50
)

// newGetRequest builds a GET request with an explicit Accept: application/json
// header. Sage HR's newer endpoints (e.g. recruitment/positions) return a 404
// "page not found" HTML response when the Accept header is absent, while the
// older core endpoints (employees, teams, ...) don't care either way — so this
// is set unconditionally on every request built here.
func newGetRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		return newGetRequest(ctx, params.NextPage.String())
	}

	spec, ok := objectSpecs[params.ObjectName]
	if !ok {
		return nil, common.ErrOperationNotSupportedForObject
	}

	if spec.fanOut != nil {
		return c.buildParentListRequest(ctx, params, spec.fanOut)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, spec.path)
	if err != nil {
		return nil, err
	}

	applyPaginationParams(url, spec.pagination, spec.perPageDefault, params)

	if spec.dateWindowed {
		applyDateWindow(url, params)
	}

	if params.ObjectName == objectEmployees {
		applyEmployeeHistoryParams(url)
	}

	return newGetRequest(ctx, url.String())
}

// buildParentListRequest builds the request for a fan-out object's PARENT list
// (e.g. /employees for employees/compensations). The actual child records are
// fetched in parseReadResponse, once the parent ids are known.
func (c *Connector) buildParentListRequest(
	ctx context.Context, params common.ReadParams, fo *fanOutSpec,
) (*http.Request, error) {
	parentSpec := objectSpecs[fo.parentObject]

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, parentSpec.path)
	if err != nil {
		return nil, err
	}

	applyPaginationParams(url, parentSpec.pagination, parentSpec.perPageDefault, params)

	if fo.parentObject == objectEmployees {
		applyEmployeeHistoryParams(url)
	}

	return newGetRequest(ctx, url.String())
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	resp *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	spec, ok := objectSpecs[params.ObjectName]
	if !ok {
		return nil, common.ErrOperationNotSupportedForObject
	}

	reqURL, err := urlbuilder.FromRawURL(request.URL)
	if err != nil {
		return nil, err
	}

	if spec.fanOut != nil {
		return c.parseFanOutResponse(ctx, params, spec.fanOut, reqURL, resp)
	}

	// leave-management/requests cannot use common.ParseResult here: that
	// helper treats an empty page as "done", but a <65-day window can
	// legitimately contain zero records while later windows still have data
	// to walk (see dateWindowNextPage / parseDateWindowedResponse).
	if spec.dateWindowed {
		return parseDateWindowedResponse(resp, params, reqURL)
	}

	return common.ParseResult(
		resp,
		recordsFunc,
		metaNextPage(reqURL, spec.pagination),
		marshalFunc(),
		params.Fields,
	)
}

// parseFanOutResponse extracts the parent ids from the current parent-list
// page, fans out one request per parent to the child endpoint (walking all of
// that child's own pages), and flattens the results. Pagination advances
// through the parent list, same as any other object.
func (c *Connector) parseFanOutResponse(
	ctx context.Context,
	params common.ReadParams,
	fo *fanOutSpec,
	reqURL *urlbuilder.URL,
	resp *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	body, ok := resp.Body()
	if !ok {
		return &common.ReadResult{Done: true, Data: []common.ReadResultRow{}}, nil
	}

	parentRecords, err := jsonquery.New(body).ArrayOptional("data")
	if err != nil {
		return nil, err
	}

	rows, err := c.fetchChildrenForParents(ctx, parentRecords, params, fo)
	if err != nil {
		return nil, err
	}

	parentSpec := objectSpecs[fo.parentObject]

	nextPage, err := metaNextPage(reqURL, parentSpec.pagination)(body)
	if err != nil {
		return nil, err
	}

	return &common.ReadResult{
		Rows:     int64(len(rows)),
		Data:     rows,
		NextPage: common.NextPageToken(nextPage),
		Done:     nextPage == "",
	}, nil
}

// fetchChildrenForParents fans out one request per parent id concurrently,
// preserving parent order in the flattened result.
func (c *Connector) fetchChildrenForParents(
	ctx context.Context,
	parents []*ajson.Node,
	params common.ReadParams,
	fo *fanOutSpec,
) ([]common.ReadResultRow, error) {
	ids := extractParentIDs(parents)
	if len(ids) == 0 {
		return nil, nil
	}

	parentSpec := objectSpecs[fo.parentObject]

	results := make([][]common.ReadResultRow, len(ids))
	jobs := make([]simultaneously.Job, len(ids))

	for i, parentID := range ids {
		idx, id := i, parentID

		jobs[idx] = func(ctx context.Context) error {
			rows, err := c.fetchChildPages(ctx, parentSpec.path, id, params, fo)
			if err != nil {
				return fmt.Errorf("fetching %s for %s %s: %w", fo.childSuffix, fo.parentObject, id, err)
			}

			results[idx] = rows

			return nil
		}
	}

	if err := simultaneously.DoCtx(ctx, maxConcurrentChildFetch, jobs...); err != nil {
		return nil, err
	}

	var data []common.ReadResultRow

	for _, rows := range results {
		data = append(data, rows...)
	}

	return data, nil
}

// fetchChildPages walks every page of one parent's child collection.
func (c *Connector) fetchChildPages(
	ctx context.Context,
	parentPath string,
	parentID string,
	params common.ReadParams,
	fo *fanOutSpec,
) ([]common.ReadResultRow, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, parentPath, parentID, fo.childSuffix)
	if err != nil {
		return nil, err
	}

	applyPaginationParams(url, fo.pagination, fo.perPageDefault, params)

	var allRows []common.ReadResultRow

	for range maxChildPagesPerParent {
		resp, err := c.JSONHTTPClient().Get(ctx, url.String())
		if err != nil {
			return nil, err
		}

		body, ok := resp.Body()
		if !ok {
			break
		}

		records, err := recordsFunc(body)
		if err != nil {
			return nil, err
		}

		rows, err := marshalFunc()(records, params.Fields.List())
		if err != nil {
			return nil, err
		}

		allRows = append(allRows, rows...)

		nextPage, err := metaNextPage(url, fo.pagination)(body)
		if err != nil {
			return nil, err
		}

		if nextPage == "" {
			break
		}

		url, err = urlbuilder.New(nextPage)
		if err != nil {
			return nil, err
		}
	}

	return allRows, nil
}

func (c *Connector) buildSingleObjectMetadataRequest(ctx context.Context, objectName string) (*http.Request, error) {
	spec, ok := objectSpecs[objectName]
	if !ok {
		return nil, common.ErrOperationNotSupportedForObject
	}

	if spec.fanOut != nil {
		return c.buildFanOutMetadataRequest(ctx, spec.fanOut)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, spec.path)
	if err != nil {
		return nil, err
	}

	applyPaginationParams(url, spec.pagination, spec.perPageDefault, common.ReadParams{PageSize: 1})

	if objectName == objectEmployees {
		applyEmployeeHistoryParams(url)
	}

	return newGetRequest(ctx, url.String())
}

// buildFanOutMetadataRequest samples one real parent id (there is no
// company-wide listing for these child objects) and returns the request for
// that parent's child endpoint. This costs one extra internal HTTP call,
// executed synchronously here, in addition to the one the framework makes
// with the returned request.
func (c *Connector) buildFanOutMetadataRequest(ctx context.Context, fo *fanOutSpec) (*http.Request, error) {
	parentSpec := objectSpecs[fo.parentObject]

	parentURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, parentSpec.path)
	if err != nil {
		return nil, err
	}

	applyPaginationParams(parentURL, parentSpec.pagination, parentSpec.perPageDefault, common.ReadParams{PageSize: 1})

	if fo.parentObject == objectEmployees {
		applyEmployeeHistoryParams(parentURL)
	}

	resp, err := c.JSONHTTPClient().Get(ctx, parentURL.String())
	if err != nil {
		return nil, err
	}

	body, ok := resp.Body()
	if !ok {
		return nil, fmt.Errorf("%w: empty response while sampling %s", common.ErrMissingExpectedValues, fo.parentObject)
	}

	parentRecords, err := jsonquery.New(body).ArrayOptional("data")
	if err != nil {
		return nil, err
	}

	ids := extractParentIDs(parentRecords)
	if len(ids) == 0 {
		return nil, fmt.Errorf(
			"%w: no %s found to sample %s from", common.ErrMissingExpectedValues, fo.parentObject, fo.childSuffix)
	}

	childURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, parentSpec.path, ids[0], fo.childSuffix)
	if err != nil {
		return nil, err
	}

	applyPaginationParams(childURL, fo.pagination, fo.perPageDefault, common.ReadParams{PageSize: 1})

	return newGetRequest(ctx, childURL.String())
}

func (c *Connector) parseSingleObjectMetadataResponse(
	ctx context.Context,
	objectName string,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ObjectMetadata, error) {
	body, ok := response.Body()
	if !ok {
		return nil, common.ErrFailedToUnmarshalBody
	}

	records, err := recordsFunc(body)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("%w: could not find a record to sample fields from", common.ErrMissingExpectedValues)
	}

	fields := make(common.FieldsMetadata)

	for field, value := range records[0] {
		fields[field] = common.FieldMetadata{
			DisplayName:  field,
			ValueType:    common.InferValueTypeFromData(value),
			ProviderType: "", // Sage HR does not expose field type metadata.
		}
	}

	return common.NewObjectMetadata(objectName, fields), nil
}
