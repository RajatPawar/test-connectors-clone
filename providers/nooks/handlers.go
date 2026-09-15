package nooks

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/providers/nooks/metadata"
)

const (
	pageSizeParam = "page[size]"
	// Nooks' own default is 50, max is 100 (docs/openapi_spec.json:
	// components.parameters.PageSize). We request the max to reduce round trips.
	defaultPageSize = "100"

	filterUpdatedAtGte = "filter[updatedAt][gte]"
	filterUpdatedAtLt  = "filter[updatedAt][lt]"  // exclusive upper bound
	filterUpdatedAtLte = "filter[updatedAt][lte]" // inclusive upper bound

	updatedAtField = "updatedAt"
)

// objectsWithInclusiveUpperBound is the one object (per docs/openapi_spec.json)
// whose list endpoint defines its own filter[updatedAt][gte]/[lte] parameters
// instead of $ref-ing the shared FilterUpdatedAtGte/FilterUpdatedAtLt components
// that every other object uses. Every other object's upper bound is exclusive
// (filter[updatedAt][lt]); prospects' is inclusive (filter[updatedAt][lte]).
// nolint:gochecknoglobals
var objectsWithInclusiveUpperBound = datautils.NewSet("prospects")

// objectsWithoutNativeTimeFilter have no filter[updatedAt] parameter at all in
// the spec (tasks: only dueAt/status/action/priority/completed/sequenceState
// filters; callDispositions: only name/id filters). Both objects do return an
// `updatedAt` field on each record, so incremental sync is implemented via
// connector-side filtering instead (see makeFilterFunc in parse.go), per
// CLAUDE.md's guidance for endpoints lacking a time-scoping parameter.
// nolint:gochecknoglobals
var objectsWithoutNativeTimeFilter = datautils.NewSet("tasks", "callDispositions")

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		// links.next (resolved to an absolute URL in nextPageFunc) already
		// encodes every filter/page param the original request used.
		url, err := urlbuilder.New(params.NextPage.String())
		if err != nil {
			return nil, err
		}

		return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	}

	path, err := metadata.Schemas.LookupURLPath(common.ModuleRoot, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(pageSizeParam, readhelper.PageSizeWithDefaultStr(params, defaultPageSize))

	applyTimeFilter(url, params)

	// TODO: reference fields (owner, prospect, account, sequence, creator, ...)
	// come back as {id, _href} stubs. Nooks supports `?include=` (max 3 values)
	// to expand up to 3 of them inline per request -- not used here since the
	// set of useful relations differs per object/caller. A future round could
	// thread a per-object include list through here if builders need hydrated
	// relations without a follow-up request.

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// applyTimeFilter adds the provider's native filter[updatedAt] query params.
// Objects with no native time filter (objectsWithoutNativeTimeFilter) are left
// unfiltered here -- they're filtered connector-side in parseReadResponse.
func applyTimeFilter(url *urlbuilder.URL, params common.ReadParams) {
	if objectsWithoutNativeTimeFilter.Has(params.ObjectName) {
		return
	}

	if !params.Since.IsZero() {
		url.WithQueryParam(filterUpdatedAtGte, datautils.Time.FormatRFC3339inUTC(params.Since))
	}

	if !params.Until.IsZero() {
		if objectsWithInclusiveUpperBound.Has(params.ObjectName) {
			url.WithQueryParam(filterUpdatedAtLte, datautils.Time.FormatRFC3339inUTC(params.Until))
		} else {
			url.WithQueryParam(filterUpdatedAtLt, datautils.Time.FormatRFC3339inUTC(params.Until))
		}
	}
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResultFiltered(
		params,
		response,
		records(),
		makeFilterFunc(params, nextRecordsURL(request)),
		readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField("id")),
		params.Fields,
	)
}
