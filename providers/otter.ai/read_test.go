package otterai

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	conversationsFirstPage := testutils.DataFromFile(t, "conversations-first-page.json")
	conversationsSecondPage := testutils.DataFromFile(t, "conversations-second-page.json")
	channelsResponse := testutils.DataFromFile(t, "channels.json")
	channelMembersResponse := testutils.DataFromFile(t, "channel-members.json")
	workspaceResponse := testutils.DataFromFile(t, "workspace.json")

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectNameChannels},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Provider error response is interpreted",
			Input: common.ReadParams{ObjectName: objectNameConversations, Fields: connectors.Fields("id", "title")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/conversations"),
				Then:  mockserver.Response(http.StatusBadRequest, []byte(`{"error":"Invalid request parameters"}`)),
			}.Server(),
			ExpectedErrs: []error{common.ErrCaller},
		},
		{
			Name: "Successful read of conversations, first page",
			Input: common.ReadParams{
				ObjectName: objectNameConversations,
				Fields:     connectors.Fields("id", "title", "abstract_summary"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/conversations"),
				Then:  mockserver.Response(http.StatusOK, conversationsFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "conversation_id",
					Fields: map[string]any{
						"id":               "conversation_id",
						"title":            "product launch meeting",
						"abstract_summary": "This meeting discussed the next steps in the project launch",
					},
					Raw: map[string]any{
						"id":               "conversation_id",
						"title":            "product launch meeting",
						"created_at":       "2025-07-31T12:00:00Z",
						"abstract_summary": "This meeting discussed the next steps in the project launch",
					},
				}},
				NextPage: "next_cursor_position",
				Done:     false,
			},
		},
		{
			Name: "Successful read of conversations, last page",
			Input: common.ReadParams{
				ObjectName: objectNameConversations,
				Fields:     connectors.Fields("id", "title"),
				NextPage:   "next_cursor_position",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/conversations"),
				Then:  mockserver.Response(http.StatusOK, conversationsSecondPage),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     1,
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name:  "Successful read of channels",
			Input: common.ReadParams{ObjectName: objectNameChannels, Fields: connectors.Fields("id", "name", "member_count")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/channels"),
				Then:  mockserver.Response(http.StatusOK, channelsResponse),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "channel_id",
					Fields: map[string]any{
						"id":           "channel_id",
						"name":         "backend product meeting",
						"member_count": float64(50),
					},
					Raw: map[string]any{
						"id":              "channel_id",
						"name":            "backend product meeting",
						"member_count":    float64(50),
						"discoverability": "private",
					},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name:  "Successful read of workspace",
			Input: common.ReadParams{ObjectName: objectNameWorkspace, Fields: connectors.Fields("id", "name", "handle")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/workspace"),
				Then:  mockserver.Response(http.StatusOK, workspaceResponse),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "42",
					Fields: map[string]any{
						"id":     float64(42),
						"name":   "workspace name",
						"handle": "otter.ai",
					},
					Raw: map[string]any{
						"id":           float64(42),
						"name":         "workspace name",
						"member_count": float64(50),
						"handle":       "otter.ai",
						"type":         "business",
					},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Successful read of channels/members fans out over every channel",
			Input: common.ReadParams{
				ObjectName: objectNameChannelsMembers,
				Fields:     connectors.Fields("id", "name", "email"),
			},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/v1/channels"),
						Then: mockserver.Response(http.StatusOK, channelsResponse),
					},
					{
						If:   mockcond.Path("/v1/channels/channel_id/members"),
						Then: mockserver.Response(http.StatusOK, channelMembersResponse),
					},
				},
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 3,
				Data: []common.ReadResultRow{
					{
						Id: "a1B2c3D4e5F6g7H8",
						Fields: map[string]any{
							"id":    "a1B2c3D4e5F6g7H8",
							"name":  "Jane Doe",
							"email": "jane.doe@example.com",
						},
						Raw: map[string]any{
							"id":    "a1B2c3D4e5F6g7H8",
							"name":  "Jane Doe",
							"email": "jane.doe@example.com",
						},
					},
					{
						Id: "h8G7f6E5d4C3b2A1",
						Fields: map[string]any{
							"id":    "h8G7f6E5d4C3b2A1",
							"name":  "John Smith",
							"email": "john.smith@example.com",
						},
						Raw: map[string]any{
							"id":    "h8G7f6E5d4C3b2A1",
							"name":  "John Smith",
							"email": "john.smith@example.com",
						},
					},
					{
						Id: "z9Y8x7W6v5U4t3S2",
						Fields: map[string]any{
							"id":    "z9Y8x7W6v5U4t3S2",
							"name":  "Alex Johnson",
							"email": "alex.johnson@example.com",
						},
						Raw: map[string]any{
							"id":    "z9Y8x7W6v5U4t3S2",
							"name":  "Alex Johnson",
							"email": "alex.johnson@example.com",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
		},
	}

	for _, tt := range tests {
		// nolint:varnamelen
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (testconn.TestableReader, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		return nil, err
	}

	// for testing we want to redirect calls to our mock server
	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
