package tipalti

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

func TestRead(t *testing.T) { //nolint:funlen,gocognit,cyclop
	t.Parallel()

	responsePayees := testutils.DataFromFile(t, "payees.json")
	responsePayments := testutils.DataFromFile(t, "payments.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "Unknown object returns unsupported error",
			Input:        common.ReadParams{ObjectName: "unknownObject", Fields: connectors.Fields("")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Read list of payees returns first page with next cursor",
			Input: common.ReadParams{ObjectName: "payees", Fields: connectors.Fields("")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/payees"),
				Then:  mockserver.Response(http.StatusOK, responsePayees),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{},
						Raw: map[string]any{
							"id":                    "payee-001",
							"refCode":               "VENDOR001",
							"status":                "active",
							"type":                  "company",
							"lastChangeDateTimeUTC": "2024-01-15T10:00:00.000Z",
							"email":                 "vendor@example.com",
							"name":                  "Acme Corp",
						},
						Id: "payee-001",
					},
				},
				NextPage: "cursor-abc123",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Read second page of payees using cursor",
			Input: common.ReadParams{
				ObjectName: "payees",
				Fields:     connectors.Fields(""),
				NextPage:   "cursor-abc123",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/v1/payees"),
					mockcond.QueryParam("pageCursor", "cursor-abc123"),
				},
				Then: mockserver.Response(http.StatusOK, responsePayments),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{},
						Raw: map[string]any{
							"id":          "payment-001",
							"status":      "paid",
							"amount":      float64(1500),
							"currency":    "USD",
							"payeeId":     "payee-001",
							"paymentDate": "2024-01-10T00:00:00.000Z",
						},
						Id: "payment-001",
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name:  "Read list of payments completes without next page",
			Input: common.ReadParams{ObjectName: "payments", Fields: connectors.Fields("")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/payments"),
				Then:  mockserver.Response(http.StatusOK, responsePayments),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{},
						Raw: map[string]any{
							"id":          "payment-001",
							"status":      "paid",
							"amount":      float64(1500),
							"currency":    "USD",
							"payeeId":     "payee-001",
							"paymentDate": "2024-01-10T00:00:00.000Z",
						},
						Id: "payment-001",
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

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
