package nooks

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

	callsPage1 := testutils.DataFromFile(t, "calls_page1.json")
	callsPage2 := testutils.DataFromFile(t, "calls_page2.json")
	errUnauthorized := testutils.DataFromFile(t, "error_unauthorized.json")
	prospects := testutils.DataFromFile(t, "prospects.json")
	tasks := testutils.DataFromFile(t, "tasks.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectCalls},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Unknown object name is not supported",
			Input: common.ReadParams{ObjectName: "unknown", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusOK, callsPage1),
			}.Server(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider error response is surfaced",
			Input: common.ReadParams{ObjectName: objectCalls, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusUnauthorized, errUnauthorized),
			}.Server(),
			ExpectedErrs: []error{common.ErrAccessToken},
		},
		{
			Name:  "Read calls, first page leads to a second page (relative links.next is resolved)",
			Input: common.ReadParams{ObjectName: objectCalls, Fields: connectors.Fields("id", "direction")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusOK, callsPage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "ff0e8400-e29b-41d4-a716-446655440060",
						Fields: map[string]any{"id": "ff0e8400-e29b-41d4-a716-446655440060", "direction": "outgoing"},
						Raw:    map[string]any{"id": "ff0e8400-e29b-41d4-a716-446655440060", "direction": "outgoing"},
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
			Input: common.ReadParams{ObjectName: objectCalls, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusOK, callsPage2),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "ff1e8400-e29b-41d4-a716-446655440061",
						Fields: map[string]any{"id": "ff1e8400-e29b-41d4-a716-446655440061"},
						Raw:    map[string]any{"id": "ff1e8400-e29b-41d4-a716-446655440061", "direction": "outgoing"},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Read prospects: Until is sent as the inclusive filter[updatedAt][lte]",
			Input: common.ReadParams{
				ObjectName: objectProspects,
				Fields:     connectors.Fields("id", "primaryEmail"),
				Since:      time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2025, time.October, 31, 23, 59, 59, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/prospects"),
					mockcond.QueryParam("filter[updatedAt][gte]", "2025-10-01T00:00:00Z"),
					mockcond.QueryParam("filter[updatedAt][lte]", "2025-10-31T23:59:59Z"),
				},
				Then: mockserver.Response(http.StatusOK, prospects),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "770e8400-e29b-41d4-a716-446655440003",
						Fields: map[string]any{"id": "770e8400-e29b-41d4-a716-446655440003", "primaryemail": "jane@acme.com"},
						Raw:    map[string]any{"id": "770e8400-e29b-41d4-a716-446655440003", "primaryEmail": "jane@acme.com"},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Read tasks: no server-side updatedAt filter, connector filters locally",
			Input: common.ReadParams{
				ObjectName: objectTasks,
				Fields:     connectors.Fields("id", "note"),
				Since:      time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/tasks"),
					mockcond.QueryParamsMissing("filter[updatedAt][gte]", "filter[updatedAt][lte]", "filter[updatedAt][lt]"),
				},
				Then: mockserver.Response(http.StatusOK, tasks),
			}.Server(),
			Expected: &common.ReadResult{
				// Only the task with updatedAt inside [Since, Until] survives;
				// the ones before Since and after Until are dropped connector-side.
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "bb1e8400-e29b-41d4-a716-446655440011",
						Fields: map[string]any{"id": "bb1e8400-e29b-41d4-a716-446655440011", "note": "Send pricing deck"},
						Raw:    map[string]any{"id": "bb1e8400-e29b-41d4-a716-446655440011", "note": "Send pricing deck"},
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
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
