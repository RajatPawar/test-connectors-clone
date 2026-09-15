package nooks

import (
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
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
			Input:      []string{"unknownObject"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Errors: map[string]error{
					"unknownObject": common.ErrObjectNotSupported,
				},
			},
		},
		{
			Name:       "Successfully describe multiple objects with metadata",
			Input:      []string{objectCallDispositions, objectAccounts},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectCallDispositions: {
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
					objectAccounts: {
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
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ObjectMetadataConnector, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}
