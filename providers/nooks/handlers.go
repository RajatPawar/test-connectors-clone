package nooks

import (
	"context"
	"net/http"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/providers/nooks/metadata"
)

const defaultPageSize = "50"

// objectsWithoutUpdatedAtFilter lists read objects whose list endpoint has no
// server-side filter[updatedAt][...] support at all.
//
// Source: docs/openapi_spec.json GET /tasks parameters -- tasks only expose
// filter[dueAt][gte|lte], filter[status], filter[action], filter[priority],
// filter[completed], and filter[sequenceState][state]; there is no
// filter[updatedAt] parameter, unlike every other list endpoint in this
// connector. For tasks, Since/Until are applied connector-side against the
// record's `updatedAt` field in parseReadResponse (see README.md).
var objectsWithoutUpdatedAtFilter = datautils.NewSet(objectNameTasks) //nolint:gochecknoglobals

// objectsWithInclusiveUpperBound lists objects whose updatedAt upper-bound
// filter is inclusive (filter[updatedAt][lte]) rather than the exclusive
// filter[updatedAt][lt] shared by every other list endpoint via the spec's
// FilterUpdatedAtLt component.
//
// Source: docs/openapi_spec.json GET /prospects parameters, which define
// filter[updatedAt][gte] and filter[updatedAt][lte] as one-off parameters
// instead of referencing the shared FilterUpdatedAtGte/FilterUpdatedAtLt
// components every other object uses.
var objectsWithInclusiveUpperBound = datautils.NewSet(objectNameProspects) //nolint:gochecknoglobals

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		// links.next was already resolved to an absolute URL in parseReadResponse.
		return http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
	}

	path, err := metadata.Schemas.LookupURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	reqURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	reqURL.WithQueryParam("page[size]", readhelper.PageSizeWithDefaultStr(params, defaultPageSize))

	applyTimeFilters(reqURL, params)

	return http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
}

// applyTimeFilters sets filter[updatedAt][gte]/[lt] (or [lte] for prospects)
// query params for objects that support server-side updatedAt filtering. For
// objectsWithoutUpdatedAtFilter, no filter is applied here -- the whole
// collection is fetched and filtered connector-side in parseReadResponse.
func applyTimeFilters(u *urlbuilder.URL, params common.ReadParams) {
	if objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		return
	}

	if !params.Since.IsZero() {
		u.WithQueryParam("filter[updatedAt][gte]", datautils.Time.FormatRFC3339inUTC(params.Since))
	}

	if !params.Until.IsZero() {
		key := "filter[updatedAt][lt]"
		if objectsWithInclusiveUpperBound.Has(params.ObjectName) {
			key = "filter[updatedAt][lte]"
		}

		u.WithQueryParam(key, datautils.Time.FormatRFC3339inUTC(params.Until))
	}
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	nextPageFunc := makeNextPageFunc(request)

	if objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		// No server-side updatedAt filter is available (tasks): fetch every
		// page and discard records outside the Since/Until window here. Task
		// list order is not documented as sorted by updatedAt, so we use
		// Unordered and always walk every page rather than stopping early.
		return common.ParseResultFiltered(
			params,
			response,
			nodeRecordsFunc,
			readhelper.MakeTimeFilterFunc(
				readhelper.Unordered,
				readhelper.NewTimeBoundary(),
				"updatedAt",
				time.RFC3339,
				nextPageFunc,
			),
			readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField("id")),
			params.Fields,
		)
	}

	return common.ParseResult(
		response,
		recordsFunc,
		nextPageFunc,
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}
