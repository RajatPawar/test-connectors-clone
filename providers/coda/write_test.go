package coda

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

func TestWrite(t *testing.T) { // nolint:funlen
	t.Parallel()

	insertResponse := testutils.DataFromFile(t, "insert_rows.json")
	updateResponse := testutils.DataFromFile(t, "update_row.json")

	const (
		rowsObject = "docs/AbCDeFGH/tables/grid-pqRstU/rows"
		rowID      = "i-tuVwxYz01"
	)

	tests := []testroutines.Write{
		{
			Name:         "Write object must be included",
			Input:        common.WriteParams{},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "Write needs record data",
			Input:        common.WriteParams{ObjectName: rowsObject},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingRecordData},
		},
		{
			Name: "Insert a row uses POST with a cells payload",
			Input: common.WriteParams{
				ObjectName: rowsObject,
				RecordData: map[string]any{
					"Name":   "Task A",
					"Status": "Open",
				},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPOST(),
					mockcond.Path("/apis/v1/docs/AbCDeFGH/tables/grid-pqRstU/rows"),
					mockcond.Body(`{"rows":[{"cells":[{"column":"Name","value":"Task A"},{"column":"Status","value":"Open"}]}]}`),
				},
				Then: mockserver.Response(http.StatusAccepted, insertResponse),
			}.Server(),
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: rowID,
				Errors:   nil,
				Data: map[string]any{
					"requestId":   "abc-123-def-456",
					"addedRowIds": []any{"i-tuVwxYz01"},
				},
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Upsert lifts keyColumns to the request top level",
			Input: common.WriteParams{
				ObjectName: rowsObject,
				RecordData: map[string]any{
					"Name":       "Task A",
					"keyColumns": []any{"Name"},
				},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPOST(),
					mockcond.Body(`{"keyColumns":["Name"],"rows":[{"cells":[{"column":"Name","value":"Task A"}]}]}`),
				},
				Then: mockserver.Response(http.StatusAccepted, insertResponse),
			}.Server(),
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: rowID,
				Data: map[string]any{
					"requestId":   "abc-123-def-456",
					"addedRowIds": []any{"i-tuVwxYz01"},
				},
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Update a row uses PUT with the row id in the path",
			Input: common.WriteParams{
				ObjectName: rowsObject,
				RecordId:   rowID,
				RecordData: map[string]any{
					"Status": "Done",
				},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPUT(),
					mockcond.Path("/apis/v1/docs/AbCDeFGH/tables/grid-pqRstU/rows/i-tuVwxYz01"),
					mockcond.Body(`{"row":{"cells":[{"column":"Status","value":"Done"}]}}`),
				},
				Then: mockserver.Response(http.StatusAccepted, updateResponse),
			}.Server(),
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: rowID,
				Data: map[string]any{
					"requestId": "abc-123-def-456",
					"id":        "i-tuVwxYz01",
				},
			},
			ExpectedErrs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.WriteConnector, error) {
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
