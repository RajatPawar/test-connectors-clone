package orb

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { //nolint:funlen,maintidx
	t.Parallel()

	responseUnauthorized := testutils.DataFromFile(t, "error-unauthorized.json")
	responseCustomersFirstPage := testutils.DataFromFile(t, "customers-first-page.json")
	responseCustomersLastPage := testutils.DataFromFile(t, "customers-last-page.json")
	responseCoupons := testutils.DataFromFile(t, "coupons.json")
	responseEmpty := testutils.DataFromFile(t, "subscriptions-empty.json")

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "customers"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "prices", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Error response is parsed",
			Input: common.ReadParams{ObjectName: "customers", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrAccessToken,
				testutils.StringError("Invalid API key."),
			},
		},
		{
			Name: "First page of customers returns records and the next cursor",
			Input: common.ReadParams{
				ObjectName: "customers",
				Fields:     connectors.Fields("name", "payment_provider", "billing_address"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/customers"),
					mockcond.QueryParam("limit", "100"),
					mockcond.QueryParamsMissing("cursor"),
				},
				Then: mockserver.Response(http.StatusOK, responseCustomersFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Id: "XuxCbt7x9L82yyeF",
					Fields: map[string]any{
						"name":             "Acme Corp",
						"payment_provider": nil,
						"billing_address": map[string]any{
							"line1":       "1 Market St",
							"line2":       nil,
							"city":        "San Francisco",
							"state":       "CA",
							"postal_code": "94105",
							"country":     "US",
						},
					},
					Raw: map[string]any{
						"external_customer_id": "acme-001",
					},
				}, {
					Id: "Ka8DvPcTgAgrZs4L",
					Fields: map[string]any{
						"name":             "Globex",
						"payment_provider": "stripe_invoice",
						"billing_address":  nil,
					},
					Raw: map[string]any{
						"balance": "-25.00",
					},
				}},
				NextPage: "Y3VzdG9tZXJzOjI=",
				Done:     false,
			},
		},
		{
			Name: "Next page sends the cursor and ends when has_more is false",
			Input: common.ReadParams{
				ObjectName: "customers",
				Fields:     connectors.Fields("name"),
				NextPage:   "Y3VzdG9tZXJzOjI=",
				PageSize:   2,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/customers"),
					mockcond.QueryParam("limit", "2"),
					mockcond.QueryParam("cursor", "Y3VzdG9tZXJzOjI="),
				},
				Then: mockserver.Response(http.StatusOK, responseCustomersLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "Pm4tYrQe9ZbN2wHc",
					Fields: map[string]any{"name": "Initech"},
					Raw:    map[string]any{"id": "Pm4tYrQe9ZbN2wHc"},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Page size is capped at the documented maximum of 100",
			Input: common.ReadParams{
				ObjectName: "credit_notes",
				Fields:     connectors.Fields("id"),
				PageSize:   500,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/credit_notes"),
					mockcond.QueryParam("limit", "100"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Expected: &common.ReadResult{Rows: 0, Data: []common.ReadResultRow{}, Done: true},
		},
		{
			Name: "Since and Until are not sent as creation filters",
			Input: common.ReadParams{
				ObjectName: "invoices",
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/invoices"),
					mockcond.QueryParamsMissing("invoice_date[gt]", "invoice_date[lte]", "created_at[gt]"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Expected: &common.ReadResult{Rows: 0, Data: []common.ReadResultRow{}, Done: true},
		},
		{
			Name:  "Coupons include archived ones",
			Input: common.ReadParams{ObjectName: "coupons", Fields: connectors.Fields("redemption_code", "archived_at")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/coupons"),
					mockcond.QueryParam("show_archived", "true"),
				},
				Then: mockserver.Response(http.StatusOK, responseCoupons),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "7iz2yanVjQoBZhyH",
					Fields: map[string]any{
						"redemption_code": "HALFOFF",
						"archived_at":     "2026-10-01T00:00:00+00:00",
					},
					Raw: map[string]any{"times_redeemed": float64(3)},
				}},
				Done: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (testconn.TestableReader, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}
