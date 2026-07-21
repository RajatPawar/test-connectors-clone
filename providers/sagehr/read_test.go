package sagehr

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	teamsPage1 := testutils.DataFromFile(t, "teams_page1.json")
	teamsPage2 := testutils.DataFromFile(t, "teams_page2.json")
	errNotFound := testutils.DataFromFile(t, "error_not_found.json")
	leavePolicies := testutils.DataFromFile(t, "leave_policies.json")
	employeesSingle := testutils.DataFromFile(t, "employees_single.json")
	employeeCompensations := testutils.DataFromFile(t, "employee_compensations.json")
	employeeLeaveBalances := testutils.DataFromFile(t, "employee_leave_balances.json")
	leaveRequestsWindow1 := testutils.DataFromFile(t, "leave_requests_window1.json")
	leaveRequestsEmptyWindow := testutils.DataFromFile(t, "leave_requests_empty_window.json")
	recruitmentPositionsSingle := testutils.DataFromFile(t, "recruitment_positions_single.json")
	recruitmentApplicants := testutils.DataFromFile(t, "recruitment_applicants.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectTeams},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Unknown object name is not supported",
			Input: common.ReadParams{ObjectName: "unknown", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/teams"),
				Then:  mockserver.Response(http.StatusOK, teamsPage1),
			}.Server(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider error response is surfaced",
			Input: common.ReadParams{ObjectName: objectTeams, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/teams"),
				Then:  mockserver.Response(http.StatusNotFound, errNotFound),
			}.Server(),
			ExpectedErrs: []error{common.ErrRetryable},
		},
		{
			Name:  "Read teams, first page leads to a second page",
			Input: common.ReadParams{ObjectName: objectTeams, Fields: connectors.Fields("id", "name")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/teams"),
				Then:  mockserver.Response(http.StatusOK, teamsPage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "19",
						Fields: map[string]any{"id": float64(19), "name": "Sales"},
						Raw: map[string]any{
							"id":           float64(19),
							"name":         "Sales",
							"manager_ids":  []any{float64(1), float64(2)},
							"employee_ids": []any{float64(5), float64(7), float64(90)},
						},
					},
				},
				NextPage: testroutines.URLTestServer + "/api/teams?page=2",
				Done:     false,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "Read teams, second (last) page",
			Input: common.ReadParams{ObjectName: objectTeams, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/teams"),
				Then:  mockserver.Response(http.StatusOK, teamsPage2),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "20",
						Fields: map[string]any{"id": float64(20)},
						Raw: map[string]any{
							"id":           float64(20),
							"name":         "Engineering",
							"manager_ids":  []any{float64(3)},
							"employee_ids": []any{float64(11), float64(12)},
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "Read a non-paginated object (no meta envelope)",
			Input: common.ReadParams{ObjectName: objectLeavePolicies, Fields: connectors.Fields("id", "name")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/leave-management/policies"),
				Then:  mockserver.Response(http.StatusOK, leavePolicies),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "1",
						Fields: map[string]any{"id": float64(1), "name": "Vacation"},
						Raw: map[string]any{
							"id": float64(1), "name": "Vacation", "unit": "days", "color": "#49B284",
							"accrue_type": "yearly", "do_not_accrue": false,
							"max_carryover": "100.0", "default_allowance": "26",
						},
					},
					{
						Id:     "2",
						Fields: map[string]any{"id": float64(2), "name": "Sickday"},
						Raw: map[string]any{
							"id": float64(2), "name": "Sickday", "unit": "days", "color": "#DB263F",
							"accrue_type": "no_tracking", "do_not_accrue": true,
							"max_carryover": "0.0", "default_allowance": "0.0",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "Read a fan-out object: employees/compensations",
			Input: common.ReadParams{ObjectName: objectEmployeeCompensations, Fields: connectors.Fields("amount")},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/api/employees/19/compensations"),
						Then: mockserver.Response(http.StatusOK, employeeCompensations),
					},
					{
						If:   mockcond.Path("/api/employees"),
						Then: mockserver.Response(http.StatusOK, employeesSingle),
					},
				},
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					// employees/compensations records have no natural id field
					// in the provider's response (see parse.go / README).
					{
						Id:     "",
						Fields: map[string]any{"amount": float64(1234)},
						Raw: map[string]any{
							"amount":     float64(1234),
							"period":     "monthly",
							"comment":    "Starting salary",
							"category":   "Salary",
							"currency":   "EUR",
							"end_date":   "2019-01-01",
							"start_date": "2017-01-01",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			// Regression test: prior review feedback mistook the connector for
			// calling the SAME URL as the parent recruitment/positions list.
			// It does not -- fetchChildPages always appends the parent id +
			// "applicants" to the position's own path. This asserts that
			// sub-collection URL directly against a mock server, independent of
			// whether the live test account happens to have any recruitment
			// positions.
			Name:  "Read a fan-out object: recruitment/positions/applicants hits the position's own sub-collection URL",
			Input: common.ReadParams{ObjectName: objectRecruitmentApplicants, Fields: connectors.Fields("id")},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/api/recruitment/positions/42/applicants"),
						Then: mockserver.Response(http.StatusOK, recruitmentApplicants),
					},
					{
						If:   mockcond.Path("/api/recruitment/positions"),
						Then: mockserver.Response(http.StatusOK, recruitmentPositionsSingle),
					},
				},
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "1",
						Fields: map[string]any{"id": float64(1)},
						Raw: map[string]any{
							"id": float64(1), "email": "jon.vondrak@example.com",
							"full_name": "Jon Vondrak", "first_name": "Jon", "last_name": "Vondrak",
							"source": "recruiters",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			// Regression test: prior review feedback mistook the connector for
			// parsing /employees fields (email, first_name, ...) into this
			// object. It does not -- fetchChildPages hits
			// /employees/{id}/leave-management/balances, whose response shape
			// (used/available/policy_id, per the OpenAPI spec example) is what
			// actually gets parsed. This pins that behavior with a mock server.
			Name: "Read a fan-out object: employees/leave-management/balances parses the CHILD endpoint, not /employees",
			Input: common.ReadParams{
				ObjectName: objectEmployeeLeaveBalances,
				Fields:     connectors.Fields("policy_id", "used", "available"),
			},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/api/employees/19/leave-management/balances"),
						Then: mockserver.Response(http.StatusOK, employeeLeaveBalances),
					},
					{
						If:   mockcond.Path("/api/employees"),
						Then: mockserver.Response(http.StatusOK, employeesSingle),
					},
				},
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "",
						Fields: map[string]any{"policy_id": float64(1), "used": float64(5.6), "available": float64(2)},
						Raw:    map[string]any{"used": float64(5.6), "available": float64(2), "policy_id": float64(1)},
					},
					{
						Id:     "",
						Fields: map[string]any{"policy_id": float64(2), "used": float64(75), "available": nil},
						Raw:    map[string]any{"used": float64(75), "available": nil, "policy_id": float64(2)},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Read leave-management/requests: window >65 days is chunked to the next 60-day window",
			Input: common.ReadParams{
				ObjectName: objectLeaveRequests,
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/leave-management/requests"),
				Then:  mockserver.Response(http.StatusOK, leaveRequestsWindow1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "2902504",
						Fields: map[string]any{"id": float64(2902504)},
						Raw: map[string]any{
							"id": float64(2902504), "hours": 3.5, "status": "Approved",
							"details": "Birthday lunch", "end_date": "2018-05-24", "policy_id": float64(1),
							"start_date": "2018-05-24", "employee_id": float64(1),
							"status_code": "approved", "request_date": "2018-05-22",
						},
					},
				},
				// First window: from=Since, to=Since+59d (2024-02-29). Since this
				// window's meta.next_page is null, NextPage advances to the next
				// 60-day window (2024-03-01..2024-04-01, capped by Until) rather
				// than being empty.
				NextPage: testroutines.URLTestServer +
					"/api/leave-management/requests?from=2024-03-01&page=1&to=2024-04-01",
				Done: false,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			// Regression test for the root-cause bug fixed this round: a <65-day
			// window with ZERO records must still advance to the next window
			// (Done=false, NextPage populated) rather than stopping. Before this
			// fix, parseReadResponse went through common.ParseResult, whose
			// done := nextPage == "" || len(marshaledData) == 0 discarded the
			// correctly-computed next-window URL whenever a window's page had no
			// records -- exactly what happened against the live sandbox account,
			// which has no leave requests in most historical windows.
			Name: "Read leave-management/requests: an empty window still advances to the next window",
			Input: common.ReadParams{
				ObjectName: objectLeaveRequests,
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/leave-management/requests"),
				Then:  mockserver.Response(http.StatusOK, leaveRequestsEmptyWindow),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 0,
				NextPage: testroutines.URLTestServer +
					"/api/leave-management/requests?from=2024-03-01&page=1&to=2024-04-01",
				Done: false,
			},
			Comparator: func(baseURL string, actual, expected *common.ReadResult) *testutils.CompareResult {
				result := testutils.NewCompareResult()

				if actual.Rows != expected.Rows {
					result.AddDifference("rows mismatch: expected the empty window to yield 0 records")
				}

				if actual.Done != expected.Done {
					result.AddDifference("done mismatch: an empty window must not stop chunking (Done=false)")
				}

				expectedNextPage := testroutines.ResolveTestServerURL(expected.NextPage.String(), baseURL)
				if actual.NextPage.String() != expectedNextPage {
					result.AddDifference("next page mismatch: expected to advance to the following 60-day window")
				}

				return result
			},
		},
		{
			// Regression test: the docs say this endpoint defaults its OWN
			// from/to window to only the current month when the params are
			// omitted. A full sync (no Since) must not rely on that default —
			// see applyDateWindow / defaultLeaveRequestLookbackYears — so the
			// connector must still send an explicit from/to and keep chunking
			// (NextPage non-empty) rather than stopping after one page.
			Name:  "Read leave-management/requests: full sync (no Since) sends an explicit historical window, not the bare endpoint",
			Input: common.ReadParams{ObjectName: objectLeaveRequests, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/leave-management/requests"),
					mockcond.Check(func(w http.ResponseWriter, r *http.Request) bool {
						from := r.URL.Query().Get("from")
						to := r.URL.Query().Get("to")

						return from != "" && to != ""
					}),
				},
				Then: mockserver.Response(http.StatusOK, leaveRequestsWindow1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Done: false,
			},
			Comparator: func(baseURL string, actual, expected *common.ReadResult) *testutils.CompareResult {
				result := testutils.NewCompareResult()

				if actual.Rows != expected.Rows {
					result.AddDifference("rows mismatch")
				}

				if actual.Done != expected.Done {
					result.AddDifference("done mismatch: expected chunking to continue (Done=false)")
				}

				if actual.NextPage == "" {
					result.AddDifference("expected a non-empty NextPage (window chunking must continue)")
				}

				return result
			},
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

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
		Metadata: map[string]string{
			"workspace": "test-company",
		},
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
