package connectwise

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

func TestRead(t *testing.T) { //nolint:funlen,cyclop
	t.Parallel()

	responsePage1 := testutils.DataFromFile(t, "read-tickets-page1.json")
	responsePage2 := testutils.DataFromFile(t, "read-tickets-page2.json")
	responseError := testutils.DataFromFile(t, "error-unauthorized.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "service/tickets"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Provider returns 401 unauthorized",
			Input: common.ReadParams{ObjectName: "service/tickets", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/service/tickets"),
				Then:  mockserver.Response(http.StatusUnauthorized, responseError),
			}.Server(),
			ExpectedErrs: []error{common.ErrAccessToken},
		},
		{
			Name:  "First page returns records with next page link",
			Input: common.ReadParams{ObjectName: "service/tickets", Fields: connectors.Fields("id", "summary")},
			Server: mockserver.Conditional{
				Setup: mockserver.ResponseChainedFuncs(
					mockserver.ContentJSON(),
					mockserver.Header(
						"Link",
						`<https://na.myconnectwise.net/v4_6_release/apis/3.0/service/tickets?pageSize=1000&page=2>; rel="next"`,
					),
				),
				If:   mockcond.Path("/service/tickets"),
				Then: mockserver.Response(http.StatusOK, responsePage1),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":      float64(12345),
						"summary": "Cannot connect to VPN",
					},
					Raw: map[string]any{
						"id":       float64(12345),
						"summary":  "Cannot connect to VPN",
						"severity": "High",
						"impact":   "High",
					},
				}},
				NextPage: "https://na.myconnectwise.net/v4_6_release/apis/3.0/service/tickets?pageSize=1000&page=2",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name:  "Last page has no Link header, read is complete",
			Input: common.ReadParams{ObjectName: "service/tickets", Fields: connectors.Fields("id", "summary")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/service/tickets"),
				Then:  mockserver.Response(http.StatusOK, responsePage2),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":      float64(12346),
						"summary": "Printer not working",
					},
					Raw: map[string]any{
						"id":       float64(12346),
						"summary":  "Printer not working",
						"severity": "Medium",
						"impact":   "Medium",
					},
				}},
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
	connector, err := NewConnector(
		common.ConnectorParams{
			Module:              common.ModuleRoot,
			AuthenticatedClient: mockutils.NewClient(),
			Metadata: map[string]string{
				"region":   "na",
				"clientId": "test-client-id",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	// Use mock server URL directly so paths resolve without the version prefix.
	connector.SetBaseURL(serverURL)

	return connector, nil
}
