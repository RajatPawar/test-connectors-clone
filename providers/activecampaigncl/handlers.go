package activecampaigncl

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

const (
	filterUpdatedAfter  = "filters[updated_after]"
	filterUpdatedBefore = "filters[updated_before]"
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		return http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(limitParam, readhelper.PageSizeWithDefaultStr(params, defaultPageSize))
	url.WithQueryParam(offsetParam, "0")

	if objectSpecs[params.ObjectName].incremental == incrementalNative {
		// TODO: the docs type filters[updated_after]/[updated_before] only as
		// "date" with no concrete example value. Every timestamp field on these
		// objects (cdate/udate/mdate) is RFC3339-with-offset, so we format in
		// UTC RFC3339 as the closest documented-compatible representation. If
		// live capture shows the provider rejecting or ignoring this format,
		// revisit against a captured 400 response body.
		if !params.Since.IsZero() {
			url.WithQueryParam(filterUpdatedAfter, datautils.Time.FormatRFC3339inUTC(params.Since))
		}

		if !params.Until.IsZero() {
			url.WithQueryParam(filterUpdatedBefore, datautils.Time.FormatRFC3339inUTC(params.Until))
		}
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	reqURL, err := urlbuilder.FromRawURL(request.URL)
	if err != nil {
		return nil, err
	}

	spec := objectSpecs[params.ObjectName]

	if spec.incremental == incrementalClientSide {
		return common.ParseResultFiltered(
			params,
			response,
			common.MakeRecordsFunc(params.ObjectName),
			readhelper.MakeTimeFilterFunc(
				readhelper.Unordered,
				readhelper.NewTimeBoundary(),
				spec.clientTimeField,
				time.RFC3339,
				offsetNextPage(reqURL),
			),
			readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField("id")),
			params.Fields,
		)
	}

	return common.ParseResult(
		response,
		common.ExtractOptionalRecordsFromPath(params.ObjectName),
		offsetNextPage(reqURL),
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}

// buildSingleObjectMetadataRequest samples one record to build field metadata
// for objects missing from the embedded schemas.json (contacts, accounts,
// dealGroups) — Priority 3 in CLAUDE.md's metadata sourcing order.
func (c *Connector) buildSingleObjectMetadataRequest(ctx context.Context, objectName string) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(limitParam, "1")
	url.WithQueryParam(offsetParam, "0")

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
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

	records, err := common.ExtractOptionalRecordsFromPath(objectName)(body)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("%w: could not find a %s record to sample fields from",
			common.ErrMissingExpectedValues, objectName)
	}

	fields := make(common.FieldsMetadata)

	for field, value := range records[0] {
		fields[field] = common.FieldMetadata{
			DisplayName: field,
			ValueType:   common.InferValueTypeFromData(value),
		}
	}

	return common.NewObjectMetadata(objectName, fields), nil
}
