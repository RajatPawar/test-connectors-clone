package orb

import (
	"context"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers/orb/internal/metadata"
)

// List endpoints default to 20 records and accept at most 100.
// https://docs.withorb.com/api-reference/pagination#limits
const maxPageSize = 100

// Invoice statuses accepted by the list endpoint's status[] filter.
// https://docs.withorb.com/api-reference/invoice/list-invoices
var invoiceStatuses = []string{"draft", "issued", "paid", "synced", "void"} // nolint:gochecknoglobals

// buildReadRequest lists every record of the object. Since and Until are not sent:
// no requested list endpoint filters on an update time, and Orb records carry no
// updated_at to filter on locally. Customers, plans and credit notes accept created_at[gt|lte]
// and invoices invoice_date[gt|lte], but those would never return a record again after it changes
// (an invoice that is paid or voided, a customer whose email changes), so every read is a full read.
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	readURL, err := c.readURL(params)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, readURL.String(), nil)
}

func (c *Connector) readURL(params common.ReadParams) (*urlbuilder.URL, error) {
	if params.NextPage != "" {
		return urlbuilder.New(params.NextPage.String())
	}

	path, err := metadata.Catalog.ReadPath(params.ObjectName)
	if err != nil {
		return nil, err
	}

	readURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	readURL.WithQueryParam("limit", strconv.Itoa(min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)))

	switch params.ObjectName {
	case objectNameInvoices:
		// Without a status filter only issued, paid and synced invoices are listed,
		// so drafts and voided invoices would never be read.
		// https://docs.withorb.com/api-reference/invoice/list-invoices
		readURL.WithQueryParamList("status[]", invoiceStatuses)
	case objectNameCoupons:
		// By default only active coupons are listed; archived ones are still referenced by
		// subscriptions and invoices. https://docs.withorb.com/api-reference/coupon/list-coupons
		readURL.WithQueryParam("show_archived", "true")
	}

	return readURL, nil
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResult(
		response,
		common.MakeRecordsFunc("data"),
		makeNextPageFunc(request.URL),
		readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField("id")),
		params.Fields,
	)
}
