package nooks

import (
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
)

func TestListObjectMetadata(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []testroutines.Metadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object requested",
			Input:      []string{"contacts"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Errors: map[string]error{
					"contacts": common.ErrObjectNotSupported,
				},
			},
		},
		{
			// Metadata comes from the embedded static schema (metadata/schemas.json),
			// so no HTTP request is made.
			Name:       "Successfully describe users and accounts",
			Input:      []string{objectNameUsers, objectNameAccounts},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameUsers: {
						DisplayName: "Users",
						Fields: map[string]common.FieldMetadata{
							"id":        {DisplayName: "id", ValueType: "string", ProviderType: "string"},
							"name":      {DisplayName: "name", ValueType: "string", ProviderType: "string"},
							"crmId":     {DisplayName: "crmId", ValueType: "string", ProviderType: "string"},
							"email":     {DisplayName: "email", ValueType: "string", ProviderType: "string"},
							"createdAt": {DisplayName: "createdAt", ValueType: "string", ProviderType: "string"},
							"updatedAt": {DisplayName: "updatedAt", ValueType: "string", ProviderType: "string"},
						},
					},
					objectNameAccounts: {
						DisplayName: "Accounts",
						Fields: map[string]common.FieldMetadata{
							"id":           {DisplayName: "id", ValueType: "string", ProviderType: "string"},
							"name":         {DisplayName: "name", ValueType: "string", ProviderType: "string"},
							"domain":       {DisplayName: "domain", ValueType: "string", ProviderType: "string"},
							"createdAt":    {DisplayName: "createdAt", ValueType: "string", ProviderType: "string"},
							"updatedAt":    {DisplayName: "updatedAt", ValueType: "string", ProviderType: "string"},
							"description":  {DisplayName: "description", ValueType: "string", ProviderType: "string"},
							"linkedInUrl":  {DisplayName: "linkedInUrl", ValueType: "string", ProviderType: "string"},
							"numEmployees": {DisplayName: "numEmployees", ValueType: "int", ProviderType: "integer"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		// nolint:varnamelen
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ObjectMetadataConnector, error) {
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

	// For testing we want to redirect calls to our mock server.
	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
