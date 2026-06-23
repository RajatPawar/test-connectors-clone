package bamboohrv3

import (
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
)

func TestListObjectMetadata(t *testing.T) {
	t.Parallel()

	tests := []testroutines.Metadata{
		{
			Name:       "Employees metadata loads from static schema",
			Input:      []string{objectNameEmployees},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameEmployees: {
						DisplayName: "Api/v1/employees",
						Fields: map[string]common.FieldMetadata{
							"employeeId": {
								DisplayName:  "employeeId",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"firstName": {
								DisplayName:  "firstName",
								ValueType:    common.ValueTypeString,
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
			Name:       "Jobs metadata loads from static schema",
			Input:      []string{objectNameJobs},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameJobs: {
						DisplayName: "Job Summaries List",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "id",
								ValueType:    common.ValueTypeInt,
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
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ObjectMetadataConnector, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}

func TestMetadataConnectorInit(t *testing.T) {
	t.Parallel()

	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	if connector == nil {
		t.Fatal("connector is nil")
	}
}
