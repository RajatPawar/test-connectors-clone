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

func TestRead(t *testing.T) { //nolint:funlen
	t.Parallel()

	responseCallsPage1 := testutils.DataFromFile(t, "calls-page1.json")
	responseCallsPage2 := testutils.DataFromFile(t, "calls-page2.json")
	responseProspects := testutils.DataFromFile(t, "prospects.json")
	responseTasks := testutils.DataFromFile(t, "tasks.json")
	responseBadRequest := testutils.DataFromFile(t, "error-bad-request.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectNameCalls},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name: "Object outside the exposed read set is not supported",
			Input: common.ReadParams{
				ObjectName: "mailboxes",
				Fields:     connectors.Fields("id"),
			},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name: "Provider error response surfaces as a caller error",
			Input: common.ReadParams{
				ObjectName: objectNameCalls,
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusBadRequest, responseBadRequest),
			}.Server(),
			ExpectedErrs: []error{common.ErrCaller},
		},
		{
			Name: "Read calls first page, following relative links.next and filtering by updatedAt",
			Input: common.ReadParams{
				ObjectName: objectNameCalls,
				Fields:     connectors.Fields("id", "direction", "duration"),
				Since:      time.Date(2025, time.November, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/calls"),
					mockcond.QueryParam("filter[updatedAt][gte]", "2025-11-01T00:00:00Z"),
					mockcond.QueryParam("filter[updatedAt][lt]", "2025-12-01T00:00:00Z"),
				},
				Then: mockserver.Response(http.StatusOK, responseCallsPage1),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{"id": "ff0e8400-e29b-41d4-a716-446655440060", "direction": "outgoing", "duration": 120.5},
						Raw: map[string]any{
							"id":        "ff0e8400-e29b-41d4-a716-446655440060",
							"direction": "outgoing",
							"duration":  120.5,
						},
					},
					{
						Fields: map[string]any{"id": "ff1e8400-e29b-41d4-a716-446655440061", "direction": "outgoing", "duration": float64(45)},
						Raw: map[string]any{
							"id":        "ff1e8400-e29b-41d4-a716-446655440061",
							"direction": "outgoing",
							"duration":  float64(45),
						},
					},
				},
				// The response's links.next is a relative reference; it must be
				// resolved against the request's origin (the mock server's URL),
				// not left as-is.
				NextPage: testroutines.URLTestServer +
					"/v1/calls?page[after]=eyJpZCI6ImZmMWU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDA2MSIsInYiOjF9&page[size]=50", //nolint:lll
				Done: false,
			},
		},
		{
			Name: "Read calls second page via the resolved absolute NextPage URL",
			Input: common.ReadParams{
				ObjectName: objectNameCalls,
				Fields:     connectors.Fields("id"),
				NextPage: testroutines.URLTestServer +
					"/v1/calls?page[after]=eyJpZCI6ImZmMWU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDA2MSIsInYiOjF9&page[size]=50", //nolint:lll
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusOK, responseCallsPage2),
			}.Server(),
			Comparator: testroutines.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     1,
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Read prospects applies the inclusive filter[updatedAt][lte] upper bound",
			Input: common.ReadParams{
				ObjectName: objectNameProspects,
				Fields:     connectors.Fields("id", "name"),
				Since:      time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, time.March, 31, 23, 59, 59, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/prospects"),
					mockcond.QueryParam("filter[updatedAt][gte]", "2026-03-01T00:00:00Z"),
					mockcond.QueryParam("filter[updatedAt][lte]", "2026-03-31T23:59:59Z"),
					mockcond.QueryParamsMissing("filter[updatedAt][lt]"),
				},
				Then: mockserver.Response(http.StatusOK, responseProspects),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{"id": "990e8400-e29b-41d4-a716-446655440020", "name": "Jane Doe"},
					Raw: map[string]any{
						"id":   "990e8400-e29b-41d4-a716-446655440020",
						"name": "Jane Doe",
					},
				}},
				Done: true,
			},
		},
		{
			Name: "Read tasks has no server-side updatedAt filter; Since/Until are applied connector-side",
			Input: common.ReadParams{
				ObjectName: objectNameTasks,
				Fields:     connectors.Fields("id", "note"),
				Since:      time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/tasks"),
					mockcond.QueryParamsMissing(
						"filter[updatedAt][gte]", "filter[updatedAt][lt]", "filter[updatedAt][lte]",
					),
				},
				Then: mockserver.Response(http.StatusOK, responseTasks),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				// Only the second task (updatedAt 2026-03-18) falls inside the
				// requested window; the first (updatedAt 2026-01-01) is discarded.
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "dd1e8400-e29b-41d4-a716-446655440051",
					Fields: map[string]any{"id": "dd1e8400-e29b-41d4-a716-446655440051", "note": "Follow-up call"},
					Raw: map[string]any{
						"id":   "dd1e8400-e29b-41d4-a716-446655440051",
						"note": "Follow-up call",
					},
				}},
				Done: true,
			},
		},
	}

	for _, tt := range tests {
		// nolint:varnamelen
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ReadConnector, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}
