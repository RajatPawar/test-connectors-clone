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

	responseWorkspaceNotFound := testutils.DataFromFile(t, "error-workspace-not-found.json")
	responseInvalidInput := testutils.DataFromFile(t, "error-invalid-input.json")
	responsePersonFirstPage := testutils.DataFromFile(t, "person-first-page.json")
	responsePersonLastPage := testutils.DataFromFile(t, "person-last-page.json")
	responseUsers := testutils.DataFromFile(t, "users.json")

	personNextPage := testconn.URLTestServer +
		"/v1/workspaces/acme/objects/person/resources?page%5Boffset%5D=2&page%5Blimit%5D=2"

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
			Input:        common.ReadParams{ObjectName: "campaigns", Fields: connectors.Fields("_id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Error response is parsed",
			Input: common.ReadParams{ObjectName: "person", Fields: connectors.Fields("_id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusNotFound, responseWorkspaceNotFound),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrNotFound,
				testutils.StringError("Workspace not found"),
			},
		},
		{
			Name:  "Validation error keeps its title and detail",
			Input: common.ReadParams{ObjectName: "deal", Fields: connectors.Fields("_id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnprocessableEntity, responseInvalidInput),
			}.Server(),
			ExpectedErrs: []error{
				testutils.StringError("Invalid input: /data/attributes/email: invalid email"),
			},
		},
		{
			Name: "First page of people is flattened and links to the next page over https",
			Input: common.ReadParams{
				ObjectName: "person",
				Fields:     connectors.Fields("_id", "job_title", "name", "_updated_at"),
				PageSize:   2,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/workspaces/acme/objects/person/resources"),
					mockcond.QueryParam("page[limit]", "2"),
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
						"_id":       "0009db0a-a4ba-48ed-a696-49b789131970",
						"job_title": "Driving Growth",
						"name": map[string]any{
							"full_name":  "Anuj Sharma",
							"last_name":  "Sharma",
							"first_name": "Anuj",
						},
						"_updated_at": "2025-08-27T18:36:13.720Z",
					},
					Raw: map[string]any{
						"id":   "0009db0a-a4ba-48ed-a696-49b789131970",
						"type": "person",
					},
				}, {
					Id: "0011b2c4-5d6e-4f70-8a9b-0c1d2e3f4a5b",
					Fields: map[string]any{
						"_id":       "0011b2c4-5d6e-4f70-8a9b-0c1d2e3f4a5b",
						"job_title": "VP of Marketing",
						"name": map[string]any{
							"full_name":  "Jane Doe",
							"last_name":  "Doe",
							"first_name": "Jane",
						},
						"_updated_at": "2026-02-01T17:45:00.000Z",
					},
					Raw: map[string]any{
						"id":            "0011b2c4-5d6e-4f70-8a9b-0c1d2e3f4a5b",
						"relationships": map[string]any{},
					},
				}},
				NextPage: common.NextPageToken(personNextPage),
				Done:     false,
			},
		},
		{
			Name: "Last page of people ends the read",
			Input: common.ReadParams{
				ObjectName: "person",
				Fields:     connectors.Fields("_id"),
				NextPage:   common.NextPageToken(personNextPage),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/workspaces/acme/objects/person/resources"),
					mockcond.QueryParam("page[offset]", "2"),
					mockcond.QueryParam("page[limit]", "2"),
				},
				Then: mockserver.Response(http.StatusOK, responsePersonLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "0022c3d5-6e7f-4081-9bac-1d2e3f4a5b6c",
					Fields: map[string]any{"_id": "0022c3d5-6e7f-4081-9bac-1d2e3f4a5b6c"},
					Raw:    map[string]any{"id": "0022c3d5-6e7f-4081-9bac-1d2e3f4a5b6c"},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Incremental read sends a day-widened _updated_at window and trims it exactly",
			Input: common.ReadParams{
				ObjectName: "person",
				Fields:     connectors.Fields("_id"),
				Since:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				// Until is inclusive: the second record was updated exactly then.
				Until: time.Date(2026, 2, 1, 17, 45, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/workspaces/acme/objects/person/resources"),
					mockcond.QueryParam("page[limit]", "500"),
					// Operator names are sent with %20, as in the docs, not "+".
					rawQueryContains("filter%5B_updated_at%5D%5BIs%20after%5D=2025-12-30"),
					rawQueryContains("filter%5B_updated_at%5D%5BIs%20on%20or%20before%5D=2026-02-03"),
				},
				Then: mockserver.Response(http.StatusOK, responsePersonFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "0011b2c4-5d6e-4f70-8a9b-0c1d2e3f4a5b",
					Fields: map[string]any{"_id": "0011b2c4-5d6e-4f70-8a9b-0c1d2e3f4a5b"},
					Raw:    map[string]any{"id": "0011b2c4-5d6e-4f70-8a9b-0c1d2e3f4a5b"},
				}},
				NextPage: common.NextPageToken(personNextPage),
				Done:     false,
			},
		},
		{
			Name: "Users are filtered by _updated_at on the connector side",
			Input: common.ReadParams{
				ObjectName: "user",
				Fields:     connectors.Fields("email", "_updated_at"),
				Since:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
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
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "user_01JZKRMYJMFWPFW74MD3G31998",
					Fields: map[string]any{
						"email":       "ayan@withampersand.com",
						"_updated_at": "2026-10-18T08:36:50.988Z",
					},
					Raw: map[string]any{"id": "user_01JZKRMYJMFWPFW74MD3G31998", "type": "user"},
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

func rawQueryContains(fragment string) mockcond.Check {
	return func(w http.ResponseWriter, r *http.Request) bool {
		return strings.Contains(r.URL.RawQuery, fragment)
	}
}
