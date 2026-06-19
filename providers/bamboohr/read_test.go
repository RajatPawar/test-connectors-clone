package bamboohr

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

	responseWebhooks := testutils.DataFromFile(t, "webhooks.json")
	responseFields := testutils.DataFromFile(t, "fields.json")
	responseDatasets := testutils.DataFromFile(t, "api_v1_2_datasets.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "Unsupported object returns ErrOperationNotSupportedForObject",
			Input:        common.ReadParams{ObjectName: "nonexistent", Fields: connectors.Fields("")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name: "Server error is propagated",
			Input: common.ReadParams{
				ObjectName: "webhooks",
				Fields:     connectors.Fields("id"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/webhooks"),
				Then:  mockserver.Response(http.StatusInternalServerError, []byte(`{"error":"server error"}`)),
			}.Server(),
			ExpectedErrs: []error{common.ErrServer},
		},
		{
			Name: "Successfully read webhooks",
			Input: common.ReadParams{
				ObjectName: "webhooks",
				Fields:     connectors.Fields("id", "name", "url"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/webhooks"),
				Then:  mockserver.Response(http.StatusOK, responseWebhooks),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id: "42",
						Fields: map[string]any{
							"id":   "42",
							"name": "Employee Update Hook",
							"url":  "https://example.com/bamboohr/webhook",
						},
						Raw: map[string]any{
							"id":       "42",
							"name":     "Employee Update Hook",
							"url":      "https://example.com/bamboohr/webhook",
							"created":  "2024-01-15 10:30:00",
							"lastSent": "2024-06-01 09:00:00",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Successfully read fields (root-array response)",
			Input: common.ReadParams{
				ObjectName: "fields",
				Fields:     connectors.Fields("id", "name"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/meta/fields"),
				Then:  mockserver.Response(http.StatusOK, responseFields),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id: "firstName",
						Fields: map[string]any{
							"id":   "firstName",
							"name": "First Name",
						},
						Raw: map[string]any{
							"id":         "firstName",
							"name":       "First Name",
							"type":       "text",
							"alias":      "firstName",
							"deprecated": false,
						},
					},
					{
						Id: "lastName",
						Fields: map[string]any{
							"id":   "lastName",
							"name": "Last Name",
						},
						Raw: map[string]any{
							"id":         "lastName",
							"name":       "Last Name",
							"type":       "text",
							"alias":      "lastName",
							"deprecated": false,
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Successfully read datasets catalog",
			Input: common.ReadParams{
				ObjectName: "api/v1_2/datasets",
				Fields:     connectors.Fields("name", "label"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1_2/datasets"),
				Then:  mockserver.Response(http.StatusOK, responseDatasets),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Fields: map[string]any{
							"name":  "employees",
							"label": "Employees",
						},
						Raw: map[string]any{
							"name":  "employees",
							"label": "Employees",
						},
					},
					{
						Fields: map[string]any{
							"name":  "jobInfo",
							"label": "Job Info",
						},
						Raw: map[string]any{
							"name":  "jobInfo",
							"label": "Job Info",
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
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
		Workspace:           "testcompany",
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
