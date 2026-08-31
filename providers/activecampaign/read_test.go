package activecampaign

import (
	"net/http"
	"net/http/httptest"
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

	contactsResponse := testutils.DataFromFile(t, "contacts.json")
	dealsResponse := testutils.DataFromFile(t, "deals.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:  "Read requires fields",
			Input: common.ReadParams{ObjectName: "contacts"},
			Server: mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Unauthorized request returns an access-token error",
			Input: common.ReadParams{ObjectName: "contacts", Fields: connectors.Fields("email")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/contacts"),
				Then:  mockserver.Response(http.StatusUnauthorized, []byte(`{"message":"No Result found"}`)),
			}.Server(),
			ExpectedErrs: []error{common.ErrAccessToken},
		},
		{
			Name:  "Read contacts first page yields a next-page offset",
			Input: common.ReadParams{ObjectName: "contacts", Fields: connectors.Fields("email")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/contacts"),
				Then:  mockserver.Response(http.StatusOK, contactsResponse),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "1",
						Fields: map[string]any{"email": "alice@example.com"},
						Raw: map[string]any{
							"id":        "1",
							"email":     "alice@example.com",
							"firstName": "Alice",
							"lastName":  "Anderson",
						},
					},
					{
						Id:     "2",
						Fields: map[string]any{"email": "bob@example.com"},
						Raw: map[string]any{
							"id":    "2",
							"email": "bob@example.com",
						},
					},
				},
				NextPage: "100",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name:  "Read deals last page is marked done",
			Input: common.ReadParams{ObjectName: "deals", Fields: connectors.Fields("title")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/deals"),
				Then:  mockserver.Response(http.StatusOK, dealsResponse),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "10",
						Fields: map[string]any{"title": "Big Deal"},
						Raw: map[string]any{
							"id":    "10",
							"title": "Big Deal",
							"value": "50000",
						},
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
				return constructTestConnector(tt.Server)
			})
		})
	}
}

func constructTestConnector(server *httptest.Server) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		AuthenticatedClient: server.Client(),
		Workspace:           "test-account",
	})
	if err != nil {
		return nil, err
	}

	// Redirect all calls to the mock server.
	connector.SetBaseURL(server.URL)

	return connector, nil
}
