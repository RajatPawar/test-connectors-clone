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

func TestRead(t *testing.T) { //nolint:funlen
	t.Parallel()

	responseUnauthorized := testutils.DataFromFile(t, "error-unauthorized.json")
	responseCustomersFirstPage := testutils.DataFromFile(t, "customers-first-page.json")
	responseCustomersLastPage := testutils.DataFromFile(t, "customers-last-page.json")
	responseInvoices := testutils.DataFromFile(t, "invoices.json")
	responseEmpty := testutils.DataFromFile(t, "empty.json")

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
			Input:        common.ReadParams{ObjectName: "events", Fields: connectors.Fields("id")},
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
				testutils.StringError("Your request did not provide a valid or well-formatted API key."),
			},
		},
		{
			Name: "First page of customers links to the next cursor",
			Input: common.ReadParams{
				ObjectName: "customers",
				Fields:     connectors.Fields("name", "email", "metadata"),
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
					Id: "eN3gvMFjPt5FjLbc",
					Fields: map[string]any{
						"name":     "11x",
						"email":    "11x@11x.ai",
						"metadata": map[string]any{},
					},
					Raw: map[string]any{
						"timezone": "Etc/UTC",
					},
				}, {
					Id: "NLGwoEaecJENHwSF",
					Fields: map[string]any{
						"name":     "staging-probe",
						"email":    "noemail.53939f75@withampersand.com",
						"metadata": map[string]any{"org_id": "d3042918-77df-442e-b3a8-42d5a427741d"},
					},
					Raw: map[string]any{
						"external_customer_id": "53939f75-7988-4855-beab-22b5ecceba43",
					},
				}},
				NextPage: testconn.URLTestServer + "/v1/customers?cursor=Y3VzdG9tZXJzOjI%3D&limit=100",
				Done:     false,
			},
		},
		{
			Name: "Page without more results is the last one",
			Input: common.ReadParams{
				ObjectName: "customers",
				Fields:     connectors.Fields("name"),
				NextPage:   testconn.URLTestServer + "/v1/customers?cursor=Y3VzdG9tZXJzOjI%3D&limit=100",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/customers"),
					mockcond.QueryParam("cursor", "Y3VzdG9tZXJzOjI="),
					mockcond.QueryParam("limit", "100"),
				},
				Then: mockserver.Response(http.StatusOK, responseCustomersLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "cwYu8K4RUVLjNJLn",
					Fields: map[string]any{"name": "dev-probe"},
					Raw:    map[string]any{"id": "cwYu8K4RUVLjNJLn"},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Page size is capped at 100 and Since and Until are not sent",
			Input: common.ReadParams{
				ObjectName: "plans",
				Fields:     connectors.Fields("id"),
				PageSize:   500,
				Since:      time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/plans"),
					mockcond.QueryParam("limit", "100"),
					mockcond.QueryParamsMissing(
						"created_at[gt]", "created_at[gte]", "created_at[lt]", "created_at[lte]",
					),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
		},
		{
			Name: "Invoices of every status are requested",
			Input: common.ReadParams{
				ObjectName: "invoices",
				Fields:     connectors.Fields("status", "total", "subscription"),
				PageSize:   10,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/invoices"),
					mockcond.QueryParam("limit", "10"),
					mockcond.QueryParam("status[]", "draft", "issued", "paid", "synced", "void"),
				},
				Then: mockserver.Response(http.StatusOK, responseInvoices),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "UFxvPGn6Smon5VGt",
					Fields: map[string]any{
						"status":       "draft",
						"total":        "8333.00",
						"subscription": map[string]any{"id": "DQ8hBPBugWDjFCFk"},
					},
					Raw: map[string]any{"invoice_number": "draft"},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name:  "Archived coupons are requested",
			Input: common.ReadParams{ObjectName: "coupons", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/coupons"),
					mockcond.QueryParam("show_archived", "true"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
		},
		{
			Name:  "Credit notes are read from their own path",
			Input: common.ReadParams{ObjectName: "credit_notes", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/credit_notes"),
					mockcond.QueryParamsMissing("show_archived", "status[]"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
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
