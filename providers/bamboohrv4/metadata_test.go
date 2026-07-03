package bamboohrv4

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
			Input:      []string{"unknown-object"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Errors: map[string]error{
					"unknown-object": common.ErrObjectNotSupported,
				},
			},
		},
		{
			Name:       "Successfully describe jobs and requests",
			Input:      []string{objectNameJobs, objectNameRequests},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameJobs: {
						DisplayName: "Job Summaries List",
						Fields: map[string]common.FieldMetadata{
							"id":     {DisplayName: "id", ValueType: "int", ProviderType: "integer"},
							"title":  {DisplayName: "title", ValueType: "other", ProviderType: "object"},
							"status": {DisplayName: "status", ValueType: "other", ProviderType: "object"},
						},
					},
					objectNameRequests: {
						DisplayName: "Time Off Requests Response",
						Fields: map[string]common.FieldMetadata{
							"id":         {DisplayName: "id", ValueType: "int", ProviderType: "integer"},
							"employeeId": {DisplayName: "employeeId", ValueType: "int", ProviderType: "integer"},
							"status":     {DisplayName: "status", ValueType: "other", ProviderType: "object"},
						},
					},
				},
			},
		},
		{
			Name:       "Employee id field is named employeeId",
			Input:      []string{objectNameEmployees},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameEmployees: {
						DisplayName: "Employees",
						Fields: map[string]common.FieldMetadata{
							"employeeId": {DisplayName: "employeeId", ValueType: "string", ProviderType: "string"},
							"firstName":  {DisplayName: "firstName", ValueType: "string", ProviderType: "string"},
							"lastName":   {DisplayName: "lastName", ValueType: "string", ProviderType: "string"},
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
