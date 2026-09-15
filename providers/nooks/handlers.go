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

const defaultPageSize = "50" // matches page[size]'s documented default; max is 100.

// objectsWithInclusiveUntilFilter is the one object (per the OpenAPI spec) whose
// updatedAt upper-bound filter is inclusive: /prospects defines its own
// filter[updatedAt][gte]/[lte] pair instead of $ref-ing the shared
// FilterUpdatedAtGte/FilterUpdatedAtLt components that every other
// updatedAt-filterable object uses (filter[updatedAt][gte]/[lt], lt exclusive).
//
//nolint:gochecknoglobals
var objectsWithInclusiveUntilFilter = datautils.NewSet(objectProspects)

// objectsWithoutUpdatedAtFilter have no filter[updatedAt] parameter at all in
// the spec (tasks only supports dueAt/status/action/priority/... filters;
// callDispositions only supports filter[name]). Per CLAUDE.md's guidance for
// endpoints with no time-scoping params, these are filtered connector-side in
// parseReadResponse instead of via a query parameter.
//
//nolint:gochecknoglobals
var objectsWithoutUpdatedAtFilter = datautils.NewSet(objectTasks, objectCallDispositions)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	// The provider's `links.next` already encodes the full query (filters,
	// page size, cursor) for the next page, so we replay it verbatim rather
	// than reconstructing filters. See parse.go's nextPageFunc.
	if params.NextPage != "" {
		return http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
	}

	path, err := metadata.Schemas.FindURLPath(common.ModuleRoot, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("page[size]", readhelper.PageSizeWithDefaultStr(params, defaultPageSize))

	applyUpdatedAtFilter(url, params)

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// applyUpdatedAtFilter adds filter[updatedAt][gte]/[lt|lte] query params for
// every object that supports server-side updatedAt filtering. Objects without
// such a parameter (see objectsWithoutUpdatedAtFilter) are left unfiltered
// here and filtered connector-side instead (parse.go).
func applyUpdatedAtFilter(url *urlbuilder.URL, params common.ReadParams) {
	if objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		return
	}

	if !params.Since.IsZero() {
		url.WithQueryParam("filter[updatedAt][gte]", datautils.Time.FormatRFC3339inUTC(params.Since))
	}

	if !params.Until.IsZero() {
		key := "filter[updatedAt][lt]"
		if objectsWithInclusiveUntilFilter.Has(params.ObjectName) {
			key = "filter[updatedAt][lte]"
		}

		url.WithQueryParam(key, datautils.Time.FormatRFC3339inUTC(params.Until))
	}
}

func (c *Connector) parseReadResponse(
	_ context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	nextPage := nextPageFunc(request.URL)

	if objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		// No provider-side updatedAt filter for this object: walk every page and
		// filter locally. The spec makes no ordering guarantee for these two
		// objects, so we can't early-stop pagination once a boundary is crossed.
		return common.ParseResultFiltered(
			params,
			response,
			recordNodes,
			readhelper.MakeTimeFilterFunc(
				readhelper.Unordered,
				readhelper.NewTimeBoundary(),
				"updatedAt", time.RFC3339,
				nextPage,
			),
			readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField("id")),
			params.Fields,
		)
	}

	return common.ParseResult(
		response,
		records,
		nextPage,
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}
