package orb

import (
	"context"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

const (
	apiVersion = "v1"

	// Default page size is 20, the maximum 100.
	// https://docs.withorb.com/api-reference/pagination#limits
	maxPageSize = 100
)

// buildReadRequest lists an object with cursor pagination: `cursor` carries the previous
// page's `pagination_metadata.next_cursor`.
//
// Since and Until are not sent. Orb records carry no last-modified timestamp, and the list
// endpoints filter only on creation (created_at[gt|gte|lt|lte] on customers, plans and credit
// notes) or invoice_date (invoices); items, coupons and subscriptions take no time filter.
// A creation filter would silently skip records that changed after they were created
// (an invoice moving from draft to paid, a voided credit note, an archived plan), so every
// read is a full read.
// https://docs.withorb.com/api-reference/customer/list-customers
// https://docs.withorb.com/api-reference/invoice/list-invoices
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	readURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, params.ObjectName)
	if err != nil {
		return nil, err
	}

	readURL.WithQueryParam("limit", strconv.Itoa(min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)))

	if params.NextPage != "" {
		readURL.WithQueryParam("cursor", params.NextPage.String())
	}

	// The coupons list returns only active coupons unless asked for archived ones, which
	// subscriptions and invoices may still reference; archived coupons carry `archived_at`.
	// https://docs.withorb.com/api-reference/coupon/list-coupons
	if params.ObjectName == objectNameCoupons {
		readURL.WithQueryParam("show_archived", "true")
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, readURL.String(), nil)
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResult(
		response,
		common.ExtractRecordsFromPath("data"),
		nextPageCursor,
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}
