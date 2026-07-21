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
	leaveRequestsWindow1 := testutils.DataFromFile(t, "leave_requests_window1.json")
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
			Name: "Read a fan-out object: recruitment/positions/applicants hits the " +
				"per-position sub-collection URL, not the parent collection",
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
						Id:     "7001",
						Fields: map[string]any{"id": float64(7001)},
						Raw:    map[string]any{"id": float64(7001)},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
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
