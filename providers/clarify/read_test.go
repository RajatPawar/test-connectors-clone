package clarify

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { //nolint:funlen,maintidx
	t.Parallel()

	responseRateLimit := testutils.DataFromFile(t, "error-rate-limit.json")
	responseInvalidInput := testutils.DataFromFile(t, "error-invalid-input.json")
	responsePersonFirstPage := testutils.DataFromFile(t, "person-first-page.json")
	responsePersonLastPage := testutils.DataFromFile(t, "person-last-page.json")
	responseUsers := testutils.DataFromFile(t, "users.json")

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{Fields: connectors.Fields("_id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "person"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "c_project", Fields: connectors.Fields("_id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Rate limit error is parsed",
			Input: common.ReadParams{ObjectName: "person", Fields: connectors.Fields("_id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusTooManyRequests, responseRateLimit),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrLimitExceeded,
				testutils.StringError("API rate limit exceeded"),
			},
		},
		{
			Name:  "Validation error combines title and detail",
			Input: common.ReadParams{ObjectName: "deal", Fields: connectors.Fields("_id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnprocessableEntity, responseInvalidInput),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrCaller,
				testutils.StringError("Invalid input: /data/attributes/email: invalid email"),
			},
		},
		{
			Name: "First page of people flattens attributes and links to the next page over the base URL",
			Input: common.ReadParams{
				ObjectName: "person",
				Fields:     connectors.Fields("job_title", "name", "email_addresses", "company_id", "_updated_at"),
				PageSize:   2,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/workspaces/acme/objects/person/resources"),
					mockcond.QueryParam("page[limit]", "2"),
					mockcond.QueryParam("page[offset]", "0"),
					mockcond.QueryParamsMissing("filter[_updated_at][Is after]"),
				},
				Then: mockserver.Response(http.StatusOK, responsePersonFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Id: "0009db0a-a4ba-48ed-a696-49b789131970",
					Fields: map[string]any{
						"job_title": "Driving Growth",
						"name": map[string]any{
							"full_name":  "Anuj Sharma",
							"last_name":  "Sharma",
							"first_name": "Anuj",
						},
						"email_addresses": map[string]any{"items": []any{"anuj@humanic.so"}},
						"company_id":      "0e69c577-f8c4-45c6-8570-9a122e0fd08a",
						"_updated_at":     "2025-08-27T18:36:13.720Z",
					},
					// Raw keeps the JSON:API envelope untouched.
					Raw: map[string]any{
						"type":          "person",
						"attributes":    mockutils.Any{},
						"relationships": map[string]any{},
					},
				}, {
					Id: "04157a9a-9295-4a32-8e91-b5056a7278e7",
					Fields: map[string]any{
						"job_title": "VP of Engineering / Integration Lead",
					},
				}},
				NextPage: testconn.URLTestServer +
					"/v1/workspaces/acme/objects/person/resources?page%5Boffset%5D=2&page%5Blimit%5D=2",
				Done: false,
			},
		},
		{
			Name: "Next page link is followed and a null next link ends the read",
			Input: common.ReadParams{
				ObjectName: "person",
				Fields:     connectors.Fields("job_title"),
				NextPage: testconn.URLTestServer +
					"/v1/workspaces/acme/objects/person/resources?page%5Boffset%5D=2&page%5Blimit%5D=2",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/objects/person/resources"),
					mockcond.QueryParam("page[limit]", "2"),
					mockcond.QueryParam("page[offset]", "2"),
				},
				Then: mockserver.Response(http.StatusOK, responsePersonLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "5f8b7d2e-9c4a-4e1b-8f3d-2a6c9e0b4d71",
					Fields: map[string]any{"job_title": "VP of Marketing"},
					Raw:    map[string]any{"type": "person"},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Incremental read of deals sends _updated_at filters with milliseconds and the maximum page size",
			Input: common.ReadParams{
				ObjectName: "deal",
				Fields:     connectors.Fields("job_title"),
				Since:      time.Date(2026, 9, 1, 0, 0, 0, 123_000_000, time.UTC),
				Until:      time.Date(2026, 9, 21, 0, 0, 0, 456_000_000, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/objects/deal/resources"),
					mockcond.QueryParam("page[limit]", "500"),
					mockcond.QueryParam("page[offset]", "0"),
					mockcond.QueryParam("filter[_updated_at][Is after]", "2026-09-01T00:00:00.123Z"),
					mockcond.QueryParam("filter[_updated_at][Is on or before]", "2026-09-21T00:00:00.456Z"),
				},
				Then: mockserver.Response(http.StatusOK, responsePersonLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "5f8b7d2e-9c4a-4e1b-8f3d-2a6c9e0b4d71",
					Fields: map[string]any{"job_title": "VP of Marketing"},
					Raw:    map[string]any{"id": "5f8b7d2e-9c4a-4e1b-8f3d-2a6c9e0b4d71"},
				}},
				Done: true,
			},
		},
		{
			Name: "Page size above the maximum is capped",
			Input: common.ReadParams{
				ObjectName: "task",
				Fields:     connectors.Fields("_id"),
				PageSize:   1000,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/objects/task/resources"),
					mockcond.QueryParam("page[limit]", "500"),
				},
				Then: mockserver.Response(http.StatusOK, responsePersonLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "5f8b7d2e-9c4a-4e1b-8f3d-2a6c9e0b4d71",
					Fields: map[string]any{"_id": "5f8b7d2e-9c4a-4e1b-8f3d-2a6c9e0b4d71"},
					Raw:    map[string]any{"type": "person"},
				}},
				Done: true,
			},
		},
		{
			Name: "Users are read in full",
			Input: common.ReadParams{
				ObjectName: "users",
				Fields:     connectors.Fields("email", "is_service_account"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/users"),
					mockcond.QueryParam("page[limit]", "500"),
					mockcond.QueryParam("page[offset]", "0"),
				},
				Then: mockserver.Response(http.StatusOK, responseUsers),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 3,
				Data: []common.ReadResultRow{{
					Id:     "user_01JYPXW4RMNYBGYQQ7HVAE6BZT",
					Fields: map[string]any{"email": "andy@whispered.com", "is_service_account": false},
					Raw:    map[string]any{"type": "user"},
				}, {
					Id:     "user_01JZKRMYJMFWPFW74MD3G31998",
					Fields: map[string]any{"email": "ayan@withampersand.com"},
				}, {
					Id:     "user_01K1VN53JRP761BANN7KDQ8AVK",
					Fields: map[string]any{"email": "jlim@withampersand.com"},
				}},
				Done: true,
			},
		},
		{
			Name: "Users are filtered by _updated_at in the connector, Since exclusive and Until inclusive",
			Input: common.ReadParams{
				ObjectName: "users",
				Fields:     connectors.Fields("email"),
				Since:      time.Date(2025, 9, 10, 19, 53, 51, 989_000_000, time.UTC),
				Until:      time.Date(2026, 8, 7, 3, 16, 10, 362_000_000, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/users"),
					mockcond.QueryParamsMissing("filter[_updated_at][Is after]", "filter[_updated_at][Is on or before]"),
				},
				Then: mockserver.Response(http.StatusOK, responseUsers),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "user_01JYPXW4RMNYBGYQQ7HVAE6BZT",
					Fields: map[string]any{"email": "andy@whispered.com"},
					Raw:    map[string]any{"type": "user"},
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
