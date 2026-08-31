package nooks

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

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	callsPage1 := testutils.DataFromFile(t, "calls_page1.json")
	callsPage2 := testutils.DataFromFile(t, "calls_page2.json")
	errUnauthorized := testutils.DataFromFile(t, "error_unauthorized.json")
	tasks := testutils.DataFromFile(t, "tasks.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "calls"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object name is not supported",
			Input:        common.ReadParams{ObjectName: "transcripts", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider error response is surfaced",
			Input: common.ReadParams{ObjectName: "calls", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusUnauthorized, errUnauthorized),
			}.Server(),
			ExpectedErrs: []error{common.ErrAccessToken},
		},
		{
			Name:  "Read calls, first page leads to a second page",
			Input: common.ReadParams{ObjectName: "calls", Fields: connectors.Fields("id", "direction")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusOK, callsPage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "ff0e8400-e29b-41d4-a716-446655440060",
						Fields: map[string]any{"id": "ff0e8400-e29b-41d4-a716-446655440060", "direction": "outgoing"},
						Raw: map[string]any{
							"id":       "ff0e8400-e29b-41d4-a716-446655440060",
							"duration": 120.5,
						},
					},
					{
						Id:     "ff1e8400-e29b-41d4-a716-446655440061",
						Fields: map[string]any{"id": "ff1e8400-e29b-41d4-a716-446655440061", "direction": "outgoing"},
						Raw: map[string]any{
							"id":       "ff1e8400-e29b-41d4-a716-446655440061",
							"duration": float64(45),
						},
					},
				},
				NextPage: testroutines.URLTestServer +
					"/v1/calls?page%5Bsize%5D=50&page%5Bafter%5D=eyJpZCI6ImZmMWU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDA2MSIsInYiOjF9", //nolint:lll
				Done: false,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "Read calls, second (last) page",
			Input: common.ReadParams{ObjectName: "calls", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusOK, callsPage2),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "ff2e8400-e29b-41d4-a716-446655440062",
						Fields: map[string]any{"id": "ff2e8400-e29b-41d4-a716-446655440062"},
						Raw: map[string]any{
							"id":        "ff2e8400-e29b-41d4-a716-446655440062",
							"direction": "incoming",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Since/Until are sent as filter[updatedAt][gte]/[lt] for calls",
			Input: common.ReadParams{
				ObjectName: "calls",
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2025, time.November, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2025, time.November, 3, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/calls"),
					mockcond.QueryParam("filter[updatedAt][gte]", "2025-11-01T00:00:00Z"),
					mockcond.QueryParam("filter[updatedAt][lt]", "2025-11-03T00:00:00Z"),
				},
				Then: mockserver.Response(http.StatusOK, callsPage2),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "ff2e8400-e29b-41d4-a716-446655440062",
						Fields: map[string]any{"id": "ff2e8400-e29b-41d4-a716-446655440062"},
						Raw:    map[string]any{"id": "ff2e8400-e29b-41d4-a716-446655440062"},
					},
				},
				Done: true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "tasks have no server-side updatedAt filter; Since filters connector-side",
			Input: common.ReadParams{
				ObjectName: "tasks",
				Fields:     connectors.Fields("id", "note"),
				Since:      time.Date(2024, time.June, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/tasks"),
					mockcond.QueryParamsMissing("filter[updatedAt][gte]", "filter[updatedAt][lt]"),
				},
				Then: mockserver.Response(http.StatusOK, tasks),
			}.Server(),
			Expected: &common.ReadResult{
				// Only the 2 tasks with updatedAt >= 2024-06-01 survive connector-side
				// filtering; the 2024-01-01 task is dropped.
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "aa11e8400-e29b-41d4-a716-446655440102",
						Fields: map[string]any{"id": "aa11e8400-e29b-41d4-a716-446655440102", "note": "Send pricing sheet"},
						Raw: map[string]any{
							"id":   "aa11e8400-e29b-41d4-a716-446655440102",
							"note": "Send pricing sheet",
						},
					},
					{
						Id:     "aa12e8400-e29b-41d4-a716-446655440103",
						Fields: map[string]any{"id": "aa12e8400-e29b-41d4-a716-446655440103", "note": "Book demo"},
						Raw: map[string]any{
							"id":   "aa12e8400-e29b-41d4-a716-446655440103",
							"note": "Book demo",
						},
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
