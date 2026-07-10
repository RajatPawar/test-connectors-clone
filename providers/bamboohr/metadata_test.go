package bamboohr

import (
	"net/http/httptest"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
)

func TestListObjectMetadata(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []testroutines.Metadata{
		{
			Name:       "Successful metadata for employees",
			Input:      []string{objectNameEmployees},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameEmployees: {
						DisplayName: "Employees",
						Fields: map[string]common.FieldMetadata{
							"employeeId": {
								DisplayName:  "employeeId",
								ValueType:    "string",
								ProviderType: "string",
							},
							"firstName": {
								DisplayName:  "firstName",
								ValueType:    "string",
								ProviderType: "string",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
			ExpectedErrs: nil,
		},
		{
			Name:       "Successful metadata for time off requests",
			Input:      []string{objectNameTimeOffRequests},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameTimeOffRequests: {
						DisplayName: "Time Off Requests",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "id",
								ValueType:    "int",
								ProviderType: "integer",
							},
							"employeeId": {
								DisplayName:  "employeeId",
								ValueType:    "int",
								ProviderType: "integer",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
			ExpectedErrs: nil,
		},
	}

	for _, tt := range tests {
		// nolint:varnamelen
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ObjectMetadataConnector, error) {
				return constructTestConnector(tt.Server)
			})
		})
	}
}

func constructTestConnector(server *httptest.Server) (*Connector, error) {
	connector, err := NewConnector(
		common.ConnectorParams{
			Module:              common.ModuleRoot,
			AuthenticatedClient: server.Client(),
			Workspace:           "test-workspace",
		},
	)
	if err != nil {
		return nil, err
	}

	// Override the base URL to point to the test server
	connector.SetUnitTestBaseURL(server.URL)

	return connector, nil
}
