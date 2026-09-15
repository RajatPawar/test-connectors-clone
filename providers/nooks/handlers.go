package nooks

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

// defaultPageSize matches the provider's own default (docs: "Default page size: 50",
// "Maximum page size: 100"). See openapi_spec.json components.parameters.PageSize.
const defaultPageSize = "50"

// filterUpdatedAtGte/Lt/Lte are copied verbatim from openapi_spec.json's query
// parameter names. Every object except prospects uses the shared
// FilterUpdatedAtGte/FilterUpdatedAtLt parameters ("...][gte]"/"...][lt]",
// lt = exclusive upper bound). prospects instead declares its own inline
// filter[updatedAt][gte]/[lte] pair (lte = inclusive upper bound) — a
// deliberate, documented deviation, not a typo, so it is preserved exactly.
const (
	filterUpdatedAtGte = "filter[updatedAt][gte]"
	filterUpdatedAtLt  = "filter[updatedAt][lt]"
	filterUpdatedAtLte = "filter[updatedAt][lte]"
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		// Nooks returns links.next as a ready-to-use URL carrying its own page[size]/
		// page[after] query params (see PaginationLinks examples in openapi_spec.json).
		// The schema's prose calls these "relative references" but every example in the
		// same spec shows full absolute URLs; per CLAUDE.md we follow the examples. If a
		// live capture ever shows a genuinely relative link, this will need base-URL
		// resolution — TODO revisit if that happens.
		url, err := urlbuilder.New(params.NextPage.String())
		if err != nil {
			return nil, err
		}

		return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("page[size]", readhelper.PageSizeWithDefaultStr(params, defaultPageSize))

	if !objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		gte, upper := filterUpdatedAtGte, filterUpdatedAtLt
		if params.ObjectName == objectProspects {
			upper = filterUpdatedAtLte
		}

		if !params.Since.IsZero() {
			url.WithQueryParam(gte, datautils.Time.FormatRFC3339inUTC(params.Since))
		}

		if !params.Until.IsZero() {
			url.WithQueryParam(upper, datautils.Time.FormatRFC3339inUTC(params.Until))
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
	idField := readhelper.NewIdField("id")

	if objectsWithoutUpdatedAtFilter.Has(params.ObjectName) {
		// tasks and callDispositions have no server-side updatedAt filter (see
		// supports.go), so Since/Until is enforced connector-side here: fetch every
		// page and discard records outside the window, per CLAUDE.md's guidance for
		// endpoints with no time-scoping parameters (see providers/sellsy/ for the
		// pattern this mirrors).
		return common.ParseResultFiltered(
			params,
			response,
			common.MakeRecordsFunc(responseDataField),
			readhelper.MakeTimeFilterFunc(
				readhelper.Unordered, // ordering of tasks/callDispositions is not documented
				readhelper.NewTimeBoundary(),
				"updatedAt", updatedAtTimeFormat,
				nextRecordsURL(),
			),
			readhelper.MakeMarshaledDataFuncWithId(nil, idField),
			params.Fields,
		)
	}

	return common.ParseResult(
		response,
		records(),
		nextRecordsURL(),
		readhelper.MakeGetMarshaledDataWithId(idField),
		params.Fields,
	)
}
