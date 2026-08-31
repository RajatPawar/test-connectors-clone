package nooks

import (
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
)

func TestListObjectMetadata(t *testing.T) { // nolint:funlen
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
			Input:      []string{"transcripts"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Errors: map[string]error{
					"transcripts": common.ErrObjectNotSupported,
				},
			},
		},
		{
			// Metadata comes entirely from the embedded static schema
			// (providers/nooks/metadata/schemas.json), so no HTTP call is made.
			Name:       "Successfully describe users and callDispositions",
			Input:      []string{"users", "callDispositions"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"users": {
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
					"callDispositions": {
						DisplayName: "Call Dispositions",
						Fields: map[string]common.FieldMetadata{
							"id":          {DisplayName: "id", ValueType: "string", ProviderType: "string"},
							"name":        {DisplayName: "name", ValueType: "string", ProviderType: "string"},
							"order":       {DisplayName: "order", ValueType: "int", ProviderType: "integer"},
							"createdAt":   {DisplayName: "createdAt", ValueType: "string", ProviderType: "string"},
							"updatedAt":   {DisplayName: "updatedAt", ValueType: "string", ProviderType: "string"},
							"callOutcome": {DisplayName: "callOutcome", ValueType: "string", ProviderType: "string"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
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

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
