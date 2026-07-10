package bamboohr

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { //nolint:funlen
	t.Parallel()

	responsePage1 := testutils.DataFromFile(t, "read/employees/page1.json")
	responsePage2 := testutils.DataFromFile(t, "read/employees/page2.json")
	responseForbidden := testutils.DataFromFile(t, "read/employees/forbidden.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{},
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
			Name: "Provider error is propagated",
			Input: common.ReadParams{
				ObjectName: objectNameEmployees,
				Fields:     connectors.Fields("employeeId"),
			},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			ExpectedErrs: []error{common.ErrForbidden},
		},
		{
			Name: "Read employees first page, records extracted with id and fields",
			Input: common.ReadParams{
				ObjectName: objectNameEmployees,
				Fields:     connectors.Fields("firstName", "lastName", "status"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/v1/employees"),
					mockcond.QueryParam("page[limit]", "250"),
				},
				Then: mockserver.Response(http.StatusOK, responsePage1),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"firstname": "Ava",
						"lastname":  "Nguyen",
						"status":    "Active",
					},
					Raw: map[string]any{
						"employeeId": "123",
						"firstName":  "Ava",
						"lastName":   "Nguyen",
						"status":     "Active",
					},
					Id: "123",
				}},
				NextPage: "https://test-workspace.bamboohr.com/api/v1/employees?" +
					"page%5Blimit%5D=100&page%5Bafter%5D=eyJuZXh0RW1wbG95ZWUiOjEyNX0%3D",
				Done: false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read employees final page, no more pages",
			Input: common.ReadParams{
				ObjectName: objectNameEmployees,
				Fields:     connectors.Fields("firstName", "lastName"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/v1/employees"),
					mockcond.QueryParam("page[limit]", "250"),
				},
				Then: mockserver.Response(http.StatusOK, responsePage2),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"firstname": "Sam",
						"lastname":  "Park",
					},
					Raw: map[string]any{
						"employeeId": "124",
						"firstName":  "Sam",
						"lastName":   "Park",
					},
					Id: "124",
				}},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
	}

	for _, tt := range tests {
		// nolint:varnamelen
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ReadConnector, error) {
				return constructTestConnector(tt.Server)
			})
		})
	}
}
