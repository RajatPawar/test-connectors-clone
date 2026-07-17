package sagehr

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { //nolint:funlen,gocognit,cyclop
	t.Parallel()

	responseTeamsPage1 := testutils.DataFromFile(t, "teams_page1.json")
	responseTeamsPage2 := testutils.DataFromFile(t, "teams_page2.json")
	responseDocumentsCategories := testutils.DataFromFile(t, "documents_categories.json")
	responseLeaveRequests := testutils.DataFromFile(t, "leave_requests.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "teams"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name: "Provider error response",
			Input: common.ReadParams{
				ObjectName: "teams",
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, []byte(`{"message": "Invalid API key"}`)),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrAccessToken,
			},
		},
		{
			Name: "Read teams first page, follow next_page",
			Input: common.ReadParams{
				ObjectName: "teams",
				Fields:     connectors.Fields("id", "name"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/teams"),
				},
				Then: mockserver.Response(http.StatusOK, responseTeamsPage1),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":   float64(19),
						"name": "Sales",
					},
					Raw: map[string]any{
						"id":           float64(19),
						"name":         "Sales",
						"manager_ids":  []any{float64(1), float64(2)},
						"employee_ids": []any{float64(5), float64(7), float64(90)},
					},
					Id: "19",
				}},
				NextPage: "2",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read teams second (last) page",
			Input: common.ReadParams{
				ObjectName: "teams",
				Fields:     connectors.Fields("id", "name"),
				NextPage:   "2",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/teams"),
					mockcond.QueryParam("page", "2"),
				},
				Then: mockserver.Response(http.StatusOK, responseTeamsPage2),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":   float64(20),
						"name": "Engineering",
					},
					Raw: map[string]any{
						"id":           float64(20),
						"name":         "Engineering",
						"manager_ids":  []any{float64(3)},
						"employee_ids": []any{float64(11), float64(12)},
					},
					Id: "20",
				}},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read documents/categories, an endpoint with no pagination",
			Input: common.ReadParams{
				ObjectName: "documents/categories",
				Fields:     connectors.Fields("id", "name"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/documents/categories"),
				Then:  mockserver.Response(http.StatusOK, responseDocumentsCategories),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{"id": float64(1), "name": "General"},
						Raw: map[string]any{
							"id": float64(1), "name": "General", "documents_count": float64(10),
						},
						Id: "1",
					},
					{
						Fields: map[string]any{"id": float64(2), "name": "Job contracts"},
						Raw: map[string]any{
							"id": float64(2), "name": "Job contracts", "documents_count": float64(1),
						},
						Id: "2",
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read leave-management/requests applies Since/Until as from/to",
			Input: common.ReadParams{
				ObjectName: "leave-management/requests",
				Fields:     connectors.Fields("id", "status"),
				Since:      time.Date(2018, time.May, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2018, time.May, 31, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/leave-management/requests"),
					mockcond.QueryParam("from", "2018-05-01"),
					mockcond.QueryParam("to", "2018-05-31"),
				},
				Then: mockserver.Response(http.StatusOK, responseLeaveRequests),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":     float64(2902504),
						"status": "Approved",
					},
					Raw: map[string]any{
						"id":     float64(2902504),
						"status": "Approved",
					},
					Id: "2902504",
				}},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ReadConnector, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}
