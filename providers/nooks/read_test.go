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

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop,maintidx
	t.Parallel()

	responseCallsPage1 := testutils.DataFromFile(t, "calls-page1.json")
	responseCallsPage2 := testutils.DataFromFile(t, "calls-page2.json")
	responseUnauthorized := testutils.DataFromFile(t, "unauthorized.json")
	responseTasksPage1 := testutils.DataFromFile(t, "tasks-page1.json")

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
			Name:  "Provider error response is interpreted",
			Input: common.ReadParams{ObjectName: "calls", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrAccessToken,
				testutils.StringError("Invalid or missing API key"),
			},
		},
		{
			Name: "Read calls first page, native updatedAt filter is applied, relative next link is resolved",
			Input: common.ReadParams{
				ObjectName: "calls",
				Fields:     connectors.Fields("id", "direction"),
				Since:      time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/calls"),
					mockcond.QueryParam("filter[updatedAt][gte]", "2025-11-01T00:00:00Z"),
					// calls is not in objectsWithInclusiveUpperBound, so the exclusive "lt" param is used.
					mockcond.QueryParam("filter[updatedAt][lt]", "2025-11-03T00:00:00Z"),
				},
				Then: mockserver.Response(http.StatusOK, responseCallsPage1),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "ff0e8400-e29b-41d4-a716-446655440060",
						Fields: map[string]any{"id": "ff0e8400-e29b-41d4-a716-446655440060", "direction": "outgoing"},
						Raw:    map[string]any{"id": "ff0e8400-e29b-41d4-a716-446655440060", "direction": "outgoing"},
					},
					{
						Id:     "ff1e8400-e29b-41d4-a716-446655440061",
						Fields: map[string]any{"id": "ff1e8400-e29b-41d4-a716-446655440061", "direction": "outgoing"},
						Raw:    map[string]any{"id": "ff1e8400-e29b-41d4-a716-446655440061", "direction": "outgoing"},
					},
				},
				NextPage: testroutines.URLTestServer +
					"/v1/calls?page[size]=2&page[after]=eyJpZCI6ImZmMWU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDA2MSIsInYiOjF9", // nolint:lll
				Done: false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read calls second (last) page via resolved next-page token",
			Input: common.ReadParams{
				ObjectName: "calls",
				Fields:     connectors.Fields("id"),
				NextPage: testroutines.URLTestServer +
					"/v1/calls?page[size]=2&page[after]=eyJpZCI6ImZmMWU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDA2MSIsInYiOjF9", // nolint:lll
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/calls"),
				Then:  mockserver.Response(http.StatusOK, responseCallsPage2),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "ff2e8400-e29b-41d4-a716-446655440062",
						Fields: map[string]any{"id": "ff2e8400-e29b-41d4-a716-446655440062"},
						Raw:    map[string]any{"id": "ff2e8400-e29b-41d4-a716-446655440062"},
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "tasks has no native updatedAt filter; Since/Until are applied connector-side",
			Input: common.ReadParams{
				ObjectName: "tasks",
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/tasks"),
					// tasks has no filter[updatedAt] parameter in the spec at all.
					mockcond.QueryParamsMissing("filter[updatedAt][gte]", "filter[updatedAt][lt]", "filter[updatedAt][lte]"), // nolint:lll
				},
				Then: mockserver.Response(http.StatusOK, responseTasksPage1),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				// Only the second fixture record (updatedAt 2026-03-18) is on/after Since (2026-03-01);
				// the first (updatedAt 2026-01-01) is filtered out connector-side.
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "dd1e8400-e29b-41d4-a716-446655440051",
						Fields: map[string]any{"id": "dd1e8400-e29b-41d4-a716-446655440051"},
						Raw:    map[string]any{"id": "dd1e8400-e29b-41d4-a716-446655440051"},
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

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		return nil, err
	}

	// for testing we want to redirect calls to our mock server
	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
