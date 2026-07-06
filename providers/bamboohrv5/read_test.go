package bamboohrv5

import (
	"net/http"
	"testing"

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

	responseJobs := testutils.DataFromFile(t, "jobs.json")
	responseRequests := testutils.DataFromFile(t, "requests.json")
	responseTimesheetEntries := testutils.DataFromFile(t, "timesheet_entries.json")
	responseSchedules := testutils.DataFromFile(t, "schedules.json")
	responseApplicationsFirstPage := testutils.DataFromFile(t, "applications-first-page.json")
	responseApplicationsLastPage := testutils.DataFromFile(t, "applications-second-page.json")
	responseEmployeesFirstPage := testutils.DataFromFile(t, "employees-first-page.json")
	responseEmployeesLastPage := testutils.DataFromFile(t, "employees-last-page.json")
	responseForbidden := testutils.DataFromFile(t, "error-forbidden.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectNameJobs},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Provider error response is passed through",
			Input: common.ReadParams{ObjectName: objectNameJobs, Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrForbidden,
				testutils.StringError(string(responseForbidden)),
			},
			Expected: nil,
		},
		{
			Name:  "Read list of jobs (no pagination, plain array response)",
			Input: common.ReadParams{ObjectName: objectNameJobs, Fields: connectors.Fields("id", "postedDate")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/applicant_tracking/jobs"),
				Then:  mockserver.Response(http.StatusOK, responseJobs),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":         float64(501),
						"posteddate": "2026-01-05",
					},
					Raw: map[string]any{
						"id":                    float64(501),
						"postedDate":            "2026-01-05",
						"newApplicantsCount":    float64(2),
						"totalApplicantsCount":  float64(10),
						"activeApplicantsCount": float64(5),
					},
					Id: "501",
				}},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read list of requests within the given date window",
			Input: common.ReadParams{
				ObjectName: objectNameRequests,
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/time_off/requests"),
				Then:  mockserver.Response(http.StatusOK, responseRequests),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{"id": float64(1348)},
					Raw: map[string]any{
						"id":         float64(1348),
						"employeeId": float64(123),
					},
					Id: "1348",
				}},
				Done: true,
			},
			ExpectedErrs: nil,
		},
		{
			Name:  "Read list of timesheet entries",
			Input: common.ReadParams{ObjectName: objectNameTimesheetEntries, Fields: connectors.Fields("id", "hours")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/time_tracking/timesheet_entries"),
				Then:  mockserver.Response(http.StatusOK, responseTimesheetEntries),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{"id": float64(9001), "hours": float64(8)},
					Raw: map[string]any{
						"id":         float64(9001),
						"employeeId": float64(123),
					},
					Id: "9001",
				}},
				Done: true,
			},
			ExpectedErrs: nil,
		},
		{
			Name:  "Read list of schedules",
			Input: common.ReadParams{ObjectName: objectNameSchedules, Fields: connectors.Fields("id", "name")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/scheduling/schedules"),
				Then:  mockserver.Response(http.StatusOK, responseSchedules),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":   "0199de9b-6bcb-77e7-941d-3caf74b9a372",
						"name": "Main Schedule",
					},
					Raw: map[string]any{
						"id":         "0199de9b-6bcb-77e7-941d-3caf74b9a372",
						"locationId": float64(123),
					},
					Id: "0199de9b-6bcb-77e7-941d-3caf74b9a372",
				}},
				Done: true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read applications, first of two pages",
			Input: common.ReadParams{
				ObjectName: objectNameApplications,
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/applicant_tracking/applications"),
				Then:  mockserver.Response(http.StatusOK, responseApplicationsFirstPage),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				NextPage: common.NextPageToken(
					"https://ampersand.bamboohr.com/api/v1/applicant_tracking/applications?page=2",
				),
				Done: false,
			},
			Comparator:   testroutines.ComparatorPagination,
			ExpectedErrs: nil,
		},
		{
			Name: "Read applications, last page has no next link",
			Input: common.ReadParams{
				ObjectName: objectNameApplications,
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/applicant_tracking/applications"),
				Then:  mockserver.Response(http.StatusOK, responseApplicationsLastPage),
			}.Server(),
			Expected: &common.ReadResult{
				Rows:     1,
				NextPage: "",
				Done:     true,
			},
			Comparator:   testroutines.ComparatorPagination,
			ExpectedErrs: nil,
		},
		{
			Name: "Read employees, first of two pages, id comes from employeeId",
			Input: common.ReadParams{
				ObjectName: objectNameEmployees,
				Fields:     connectors.Fields("firstName", "lastName"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/employees"),
				Then:  mockserver.Response(http.StatusOK, responseEmployeesFirstPage),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{"firstname": "Ava", "lastname": "Nguyen"},
						Raw:    map[string]any{"employeeId": "123"},
						Id:     "123",
					},
					{
						Fields: map[string]any{"firstname": "Sam", "lastname": "Park"},
						Raw:    map[string]any{"employeeId": "124"},
						Id:     "124",
					},
				},
				NextPage: common.NextPageToken(
					"https://ampersand.bamboohr.com/api/v1/employees?" +
						"page%5Blimit%5D=100&page%5Bafter%5D=eyJuZXh0RW1wbG95ZWUiOjEyNX0%3D",
				),
				Done: false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read employees, last page has a null next link",
			Input: common.ReadParams{
				ObjectName: objectNameEmployees,
				Fields:     connectors.Fields("firstName"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/employees"),
				Then:  mockserver.Response(http.StatusOK, responseEmployeesLastPage),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{"firstname": "Lena"},
					Raw:    map[string]any{"employeeId": "200"},
					Id:     "200",
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

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		AuthenticatedClient: mockutils.NewClient(),
		Metadata: map[string]string{
			"company": "ampersand",
		},
	})
	if err != nil {
		return nil, err
	}

	// for testing we want to redirect calls to our mock server
	connector.SetBaseURL(serverURL)

	return connector, nil
}
