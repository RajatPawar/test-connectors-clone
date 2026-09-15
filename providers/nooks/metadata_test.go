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
			Input:      []string{"widgets"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Errors: map[string]error{
					"widgets": common.ErrObjectNotSupported,
				},
			},
		},
		{
			Name:       "Successfully describe calls and prospects",
			Input:      []string{"calls", "prospects"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"calls": {
						DisplayName: "Calls",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "id",
								ValueType:    "string",
								ProviderType: "string",
							},
							"time": {
								DisplayName:  "time",
								ValueType:    "string",
								ProviderType: "string",
							},
							"updatedAt": {
								DisplayName:  "updatedAt",
								ValueType:    "string",
								ProviderType: "string",
							},
						},
					},
					"prospects": {
						DisplayName: "Prospects",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "id",
								ValueType:    "string",
								ProviderType: "string",
							},
							"primaryEmail": {
								DisplayName:  "primaryEmail",
								ValueType:    "string",
								ProviderType: "string",
							},
							"updatedAt": {
								DisplayName:  "updatedAt",
								ValueType:    "string",
								ProviderType: "string",
							},
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
