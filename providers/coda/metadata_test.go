package coda

import (
	"testing"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
)

func TestListObjectMetadata(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object is reported per object",
			Input:      []string{"docs", "tables"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"docs": {
						DisplayName: "Docs",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "id",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"title": {
								DisplayName:  "title",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(false),
							},
							"name": {
								DisplayName:  "name",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"updatedAt": {
								DisplayName:  "updatedAt",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"type": {
								DisplayName:  "type",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								ReadOnly:     new(true),
								Values:       []common.FieldValue{{Value: "doc", DisplayValue: "doc"}},
							},
						},
					},
				},
				Errors: map[string]error{
					"tables": common.ErrObjectNotSupported,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (testconn.TestableMetadataReader, error) {
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

	connector.SetUnitTestMockServerBaseUrl(serverURL)

	return connector, nil
}
