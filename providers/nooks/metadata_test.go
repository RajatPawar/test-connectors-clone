package nooks

import (
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
)

func TestListObjectMetadata(t *testing.T) {
	t.Parallel()

	tests := []testroutines.Metadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Successfully describe multiple objects with static schema",
			Input:      []string{objectCalls, objectProspects},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectCalls: {
						DisplayName: "Calls",
						Fields: map[string]common.FieldMetadata{
							"id":        {DisplayName: "id", ValueType: "string", ProviderType: "string"},
							"direction": {DisplayName: "direction", ValueType: "string", ProviderType: "string"},
							"duration":  {DisplayName: "duration", ValueType: "other", ProviderType: "number"},
						},
					},
					objectProspects: {
						DisplayName: "Prospects",
						Fields: map[string]common.FieldMetadata{
							"id":           {DisplayName: "id", ValueType: "string", ProviderType: "string"},
							"primaryEmail": {DisplayName: "primaryEmail", ValueType: "string", ProviderType: "string"},
							"firstName":    {DisplayName: "firstName", ValueType: "string", ProviderType: "string"},
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
