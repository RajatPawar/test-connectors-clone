package bamboographtrial5

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

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	responseJobs := testutils.DataFromFile(t, "jobs.json")
	responseSchedulesLastPage := testutils.DataFromFile(t, "schedules-last-page.json")
	responseSchedulesFullPage := testutils.DataFromFile(t, "schedules-full-page.json")
	responsePolicies := testutils.DataFromFile(t, "policies.json")
	responseError := testutils.DataFromFile(t, "error-response.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name: "Provider returns 403 forbidden",
			Input: common.ReadParams{
				ObjectName: "jobs",
				Fields:     connectors.Fields(""),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/applicant_tracking/jobs"),
				Then:  mockserver.Response(http.StatusForbidden, responseError),
			}.Server(),
			ExpectedErrs: []error{common.ErrForbidden},
		},
		{
			Name: "Read list of jobs returns one record",
			Input: common.ReadParams{
				ObjectName: "jobs",
				Fields:     connectors.Fields("id", "title"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/applicant_tracking/jobs"),
				Then:  mockserver.Response(http.StatusOK, responseJobs),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{
							"id":    float64(1),
							"title": "Software Engineer",
						},
						Raw: map[string]any{
							"id":    float64(1),
							"title": "Software Engineer",
						},
						Id: "1",
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read schedules last page returns done",
			Input: common.ReadParams{
				ObjectName: "schedules",
				NextPage:   "2",
				Fields:     connectors.Fields("id", "name"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/scheduling/schedules"),
				Then:  mockserver.Response(http.StatusOK, responseSchedulesLastPage),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{
							"id":   float64(42),
							"name": "Standard Week",
						},
						Raw: map[string]any{
							"id":   float64(42),
							"name": "Standard Week",
						},
						Id: "42",
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read schedules full page returns next page token",
			Input: common.ReadParams{
				ObjectName: "schedules",
				Fields:     connectors.Fields(""),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/scheduling/schedules"),
				Then:  mockserver.Response(http.StatusOK, responseSchedulesFullPage),
			}.Server(),
			Comparator: testroutines.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     100,
				NextPage: "2",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read list of time off policies returns one record",
			Input: common.ReadParams{
				ObjectName: "policies",
				Fields:     connectors.Fields("id", "name"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/meta/time_off/policies"),
				Then:  mockserver.Response(http.StatusOK, responsePolicies),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{
							"id":   float64(10),
							"name": "PTO",
						},
						Raw: map[string]any{
							"id":   float64(10),
							"name": "PTO",
						},
						Id: "10",
					},
				},
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
