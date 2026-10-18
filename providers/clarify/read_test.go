package clarify

import (
	"net/http"
	"strings"
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
	responseValidation := testutils.DataFromFile(t, "error-validation.json")
	responseDealFirstPage := testutils.DataFromFile(t, "deal-first-page.json")
	responseDealLastPage := testutils.DataFromFile(t, "deal-last-page.json")
	responseUsers := testutils.DataFromFile(t, "users.json")

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "deal"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "c_project", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Unauthorized error is parsed",
			Input: common.ReadParams{ObjectName: "deal", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrAccessToken,
				testutils.StringError("Invalid API key"),
			},
		},
		{
			Name:  "Validation error combines title and detail",
			Input: common.ReadParams{ObjectName: "person", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnprocessableEntity, responseValidation),
			}.Server(),
			ExpectedErrs: []error{
				testutils.StringError("Invalid input: /page/limit: must be <= 500"),
			},
		},
		{
			Name: "First page of deals is flattened and links to the next page over the request's scheme",
			Input: common.ReadParams{
				ObjectName: "deal",
				Fields:     connectors.Fields("name", "stage", "amount", "company_id", "_updated_at", "type"),
				PageSize:   2,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/workspaces/acme/objects/deal/resources"),
					mockcond.QueryParam("page[limit]", "2"),
					mockcond.QueryParamsMissing("filter[_updated_at][Is after]", "filter[_updated_at][Is before]"),
				},
				Then: mockserver.Response(http.StatusOK, responseDealFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Id: "0032b25d-f7f4-47df-abba-69d9fad0df94",
					Fields: map[string]any{
						"name":        "Mysa",
						"stage":       "Nurture",
						"amount":      float64(20000),
						"company_id":  "e069e184-fb6a-4e2a-ac01-7c8bce8d89d9",
						"_updated_at": "2026-07-30T04:24:41.093Z",
						"type":        "deal",
					},
					Raw: map[string]any{
						"id":            "0032b25d-f7f4-47df-abba-69d9fad0df94",
						"relationships": map[string]any{},
					},
				}, {
					Id: "0651a533-029c-4bb9-a5ac-f1ee74265212",
					Fields: map[string]any{
						"name":       "Zebra BI",
						"company_id": nil,
					},
					Raw: map[string]any{
						"type": "deal",
					},
				}},
				NextPage: testconn.URLTestServer +
					"/v1/workspaces/acme/objects/deal/resources?page%5Boffset%5D=2&page%5Blimit%5D=2",
				Done: false,
			},
		},
		{
			Name: "Next page link is requested as given and ends paging",
			Input: common.ReadParams{
				ObjectName: "deal",
				Fields:     connectors.Fields("name"),
				NextPage: testconn.URLTestServer +
					"/v1/workspaces/acme/objects/deal/resources?page%5Boffset%5D=2&page%5Blimit%5D=2",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/objects/deal/resources"),
					mockcond.QueryParam("page[offset]", "2"),
					mockcond.QueryParam("page[limit]", "2"),
				},
				Then: mockserver.Response(http.StatusOK, responseDealLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "1a7e2c44-5b8d-4f0e-9c3a-2d6b8e0f4a17",
					Fields: map[string]any{"name": "Initech"},
					Raw:    map[string]any{"id": "1a7e2c44-5b8d-4f0e-9c3a-2d6b8e0f4a17"},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Incremental read widens the filter to whole seconds and trims the records to (Since, Until]",
			Input: common.ReadParams{
				ObjectName: "deal",
				Fields:     connectors.Fields("name"),
				Since:      time.Date(2026, 7, 30, 4, 24, 41, 93_000_000, time.UTC),
				Until:      time.Date(2026, 7, 30, 4, 28, 1, 75_000_000, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/objects/deal/resources"),
					mockcond.QueryParam("page[limit]", "500"),
					mockcond.QueryParam("filter[_updated_at][Is after]", "2026-07-30T04:24:40Z"),
					mockcond.QueryParam("filter[_updated_at][Is before]", "2026-07-30T04:28:02Z"),
					// Operator names are sent with %20, as in the docs.
					mockcond.CustomCondition(func(_ http.ResponseWriter, r *http.Request) bool {
						return strings.Contains(r.URL.RawQuery, "%5BIs%20after%5D") &&
							strings.Contains(r.URL.RawQuery, "%5BIs%20before%5D")
					}),
				},
				Then: mockserver.Response(http.StatusOK, responseDealFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				// Mysa was updated exactly at Since, which is exclusive; Zebra BI exactly at Until, which is inclusive.
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "0651a533-029c-4bb9-a5ac-f1ee74265212",
					Fields: map[string]any{"name": "Zebra BI"},
					Raw:    map[string]any{"id": "0651a533-029c-4bb9-a5ac-f1ee74265212"},
				}},
				NextPage: testconn.URLTestServer +
					"/v1/workspaces/acme/objects/deal/resources?page%5Boffset%5D=2&page%5Blimit%5D=2",
				Done: false,
			},
		},
		{
			Name: "Users are read in full and filtered by the connector",
			Input: common.ReadParams{
				ObjectName: "users",
				Fields:     connectors.Fields("email", "is_service_account", "name"),
				Since:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/workspaces/acme/users"),
					mockcond.QueryParam("page[limit]", "500"),
					mockcond.QueryParamsMissing("filter[_updated_at][Is after]"),
				},
				Then: mockserver.Response(http.StatusOK, responseUsers),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Id: "user_01JZKRMYJMFWPFW74MD3G31998",
					Fields: map[string]any{
						"email":              "ayan@acme.com",
						"is_service_account": false,
						"name": map[string]any{
							"full_name":  "Ayan Barua",
							"last_name":  "Barua",
							"first_name": "Ayan",
						},
					},
					Raw: map[string]any{"type": "user"},
				}, {
					Id:     "user_01JYPXW4RMNYBGYQQ7HVAE6BZT",
					Fields: map[string]any{"email": "andy@acme.com"},
					Raw:    map[string]any{"id": "user_01JYPXW4RMNYBGYQQ7HVAE6BZT"},
				}},
				NextPage: "",
				Done:     true,
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
