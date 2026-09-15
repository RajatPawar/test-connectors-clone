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

func TestRead(t *testing.T) { // nolint:funlen
	t.Parallel()

	callsPage1 := testutils.DataFromFile(t, "calls_page1.json")
	callsPage2 := testutils.DataFromFile(t, "calls_page2.json")
	errUnauthorized := testutils.DataFromFile(t, "error_unauthorized.json")
	callDispositions := testutils.DataFromFile(t, "call_dispositions.json")

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
			Name:  "Read calls, first page leads to a second page",
			Input: common.ReadParams{ObjectName: objectCalls, Fields: connectors.Fields("id", "duration")},
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
						Fields: map[string]any{"id": "ff0e8400-e29b-41d4-a716-446655440060", "duration": float64(120.5)},
						Raw: map[string]any{
							"id":       "ff0e8400-e29b-41d4-a716-446655440060",
							"duration": float64(120.5),
							"source":   "nooks",
						},
					},
				},
				NextPage: "https://partner-api.nooks.in/v1/calls?page[size]=50&page[after]=" +
					"eyJpZCI6ImZmMWU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDA2MSIsInYiOjF9",
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
						Raw: map[string]any{
							"id":     "ff1e8400-e29b-41d4-a716-446655440061",
							"owner":  nil,
							"source": "nooks",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			// callDispositions has no filter[updatedAt] query parameter in the spec, so
			// Since/Until is enforced connector-side (see supports.go/handlers.go). Only
			// the disposition whose updatedAt falls inside [Since, Until] should survive.
			Name: "Read callDispositions applies connector-side updatedAt filtering",
			Input: common.ReadParams{
				ObjectName: objectCallDispositions,
				Fields:     connectors.Fields("id", "name"),
				Since:      time.Date(2025, time.November, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/callDispositions"),
				Then:  mockserver.Response(http.StatusOK, callDispositions),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "dd0e8400-e29b-41d4-a716-446655440071",
						Fields: map[string]any{"id": "dd0e8400-e29b-41d4-a716-446655440071", "name": "Left Voicemail"},
						Raw: map[string]any{
							"id":          "dd0e8400-e29b-41d4-a716-446655440071",
							"name":        "Left Voicemail",
							"callOutcome": "voicemail",
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
