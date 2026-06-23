package bamboohrv3

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

func TestRead(t *testing.T) { //nolint:funlen,gocognit,cyclop,maintidx
	t.Parallel()

	responseEmployeesFirstPage := testutils.DataFromFile(t, "read-employees-first-page.json")
	responseEmployeesLastPage := testutils.DataFromFile(t, "read-employees-last-page.json")
	responseJobs := testutils.DataFromFile(t, "read-jobs.json")
	responseApplicationsFirstPage := testutils.DataFromFile(t, "read-applications-first-page.json")
	responseForbidden := testutils.DataFromFile(t, "error-forbidden.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectNameEmployees},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "unknown_object", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider returns forbidden error",
			Input: common.ReadParams{ObjectName: objectNameJobs, Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			ExpectedErrs: []error{common.ErrForbidden},
		},
		{
			Name: "Read employees first page returns cursor for next page",
			Input: common.ReadParams{
				ObjectName: objectNameEmployees,
				Fields:     connectors.Fields("employeeId", "firstName", "lastName"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/v1/employees"),
					mockcond.QueryParam("page[limit]", "250"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmployeesFirstPage),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{
							"employeeid": "100",
							"firstname":  "John",
							"lastname":   "Smith",
						},
						Raw: map[string]any{
							"employeeId": "100",
						},
					},
					{
						Fields: map[string]any{
							"employeeid": "101",
							"firstname":  "Jane",
							"lastname":   "Doe",
						},
						Raw: map[string]any{
							"employeeId": "101",
						},
					},
				},
				NextPage: "eyJuZXh0RW1wbG95ZWUiOjEwMn0=",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read employees last page has no next cursor",
			Input: common.ReadParams{
				ObjectName: objectNameEmployees,
				Fields:     connectors.Fields("employeeId", "firstName"),
				NextPage:   common.NextPageToken("eyJuZXh0RW1wbG95ZWUiOjEwMX0="),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/v1/employees"),
					mockcond.QueryParam("page[after]", "eyJuZXh0RW1wbG95ZWUiOjEwMX0="),
				},
				Then: mockserver.Response(http.StatusOK, responseEmployeesLastPage),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"employeeid": "102",
					},
					Raw: map[string]any{
						"employeeId": "102",
					},
				}},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read jobs returns root array with no pagination",
			Input: common.ReadParams{
				ObjectName: objectNameJobs,
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/applicant_tracking/jobs"),
				Then:  mockserver.Response(http.StatusOK, responseJobs),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{"id": float64(1)},
						Raw:    map[string]any{"id": float64(1)},
					},
					{
						Fields: map[string]any{"id": float64(2)},
						Raw:    map[string]any{"id": float64(2)},
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read applications first page extracts page number for next page",
			Input: common.ReadParams{
				ObjectName: objectNameApplications,
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/applicant_tracking/applications"),
				Then:  mockserver.Response(http.StatusOK, responseApplicationsFirstPage),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{"id": float64(1001)},
						Raw:    map[string]any{"id": float64(1001)},
					},
				},
				NextPage: "2",
				Done:     false,
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
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
