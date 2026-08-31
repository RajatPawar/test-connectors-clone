package nooks

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/providers/nooks/metadata"
)

const (
	pageSizeQueryParam = "page[size]"
	// defaultPageSize matches Nooks' own documented default (docs: "Default page size: 50").
	defaultPageSize = "50"
	// maxPageSize is Nooks' documented cap (docs: "Maximum page size: 100").
	maxPageSize = 100

	// apiVersionPathPrefix is the spec's server path segment
	// (https://partner-api.nooks.in/v1) applied here rather than in
	// ProviderInfo.BaseURL, per repo convention that base URLs stay
	// version-free; the resulting request URL is unchanged.
	apiVersionPathPrefix = "/v1"

	filterUpdatedAtGte = "filter[updatedAt][gte]"
	filterUpdatedAtLt  = "filter[updatedAt][lt]"
	filterUpdatedAtLte = "filter[updatedAt][lte]"

	// updatedAtField is the response field used to advance connector-side
	// incremental reads for objects with no server-side updatedAt filter.
	updatedAtField = "updatedAt"
)

// objectsWithoutUpdatedAtFilter lists objects whose GET endpoint has no
// filter[updatedAt] query parameter at all (confirmed against
// openapi_spec.json's parameter lists for each path). For these, incremental
// sync is done connector-side in parseReadResponse.
//
//nolint:gochecknoglobals
var objectsWithoutUpdatedAtFilter = datautils.NewSet("tasks", "callDispositions")

// objectsWithInclusiveUntilFilter lists objects whose upper-bound updatedAt
// filter is the inclusive `lte` (vs. the exclusive `lt` FilterUpdatedAtLt
// component used by every other object). Per openapi_spec.json, /prospects is
// the only endpoint that defines its own inline filter[updatedAt][lte]
// instead of referencing the shared FilterUpdatedAtLt parameter.
//
//nolint:gochecknoglobals
var objectsWithInclusiveUntilFilter = datautils.NewSet("prospects")

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		// nextPageFunc already resolved links.next to an absolute URL that carries
		// forward every query parameter (including our filters), so we just GET it.
		return http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
	}

	path, err := metadata.Schemas.LookupURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersionPathPrefix, path)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(pageSizeQueryParam, pageSizeStr(params))

	if !objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		applyUpdatedAtFilter(url, params)
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// pageSizeStr resolves the requested page size, clamped to Nooks' documented
// maximum of 100 (docs: "Number of items per page (max 100)").
func pageSizeStr(params common.ReadParams) string {
	size := readhelper.PageSizeWithDefaultStr(params, defaultPageSize)

	if n, err := strconv.Atoi(size); err == nil && n > maxPageSize {
		return strconv.Itoa(maxPageSize)
	}

	return size
}

// applyUpdatedAtFilter adds the server-side incremental filter for objects
// that support it (all except tasks/callDispositions — see
// objectsWithoutUpdatedAtFilter).
func applyUpdatedAtFilter(u *urlbuilder.URL, params common.ReadParams) {
	if !params.Since.IsZero() {
		u.WithQueryParam(filterUpdatedAtGte, datautils.Time.FormatRFC3339inUTC(params.Since))
	}

	if !params.Until.IsZero() {
		key := filterUpdatedAtLt
		if objectsWithInclusiveUntilFilter.Has(params.ObjectName) {
			key = filterUpdatedAtLte
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
	if objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		// tasks / callDispositions have no provider-side updatedAt filter, so
		// filter connector-side using the updatedAt field present on every
		// record (CLAUDE.md: implement connector-side filtering when the
		// endpoint has no time-scoping parameters at all). Record ordering
		// within a page is not documented, so we use Unordered — this filters
		// correctly but never short-circuits pagination early.
		return common.ParseResultFiltered(
			params,
			response,
			nodeRecordsFunc,
			readhelper.MakeTimeFilterFunc(
				readhelper.Unordered,
				readhelper.NewTimeBoundary(),
				updatedAtField, time.RFC3339,
				nextPageFunc(request),
			),
			readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField("id")),
			params.Fields,
		)
	}

	return common.ParseResult(
		response,
		recordsFunc,
		nextPageFunc(request),
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}
