package sagehr

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestListObjectMetadata(t *testing.T) { // nolint:funlen
	t.Parallel()

	teamsPage1 := testutils.DataFromFile(t, "teams_page1.json")
	leavePolicies := testutils.DataFromFile(t, "leave_policies.json")

	tests := []testroutines.Metadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Successfully describe multiple objects",
			Input:      []string{objectTeams, objectLeavePolicies},
			Comparator: testroutines.ComparatorSubsetMetadata,
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/api/teams"),
						Then: mockserver.Response(http.StatusOK, teamsPage1),
					},
					{
						If:   mockcond.Path("/api/leave-management/policies"),
						Then: mockserver.Response(http.StatusOK, leavePolicies),
					},
				},
			}.Server(),
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectTeams: {
						DisplayName: objectTeams,
						Fields: map[string]common.FieldMetadata{
							"id":           {DisplayName: "id", ValueType: "float"},
							"name":         {DisplayName: "name", ValueType: "string"},
							"manager_ids":  {DisplayName: "manager_ids", ValueType: "other"},
							"employee_ids": {DisplayName: "employee_ids", ValueType: "other"},
						},
					},
					objectLeavePolicies: {
						DisplayName: objectLeavePolicies,
						Fields: map[string]common.FieldMetadata{
							"id":                {DisplayName: "id", ValueType: "float"},
							"name":              {DisplayName: "name", ValueType: "string"},
							"unit":              {DisplayName: "unit", ValueType: "string"},
							"color":             {DisplayName: "color", ValueType: "string"},
							"accrue_type":       {DisplayName: "accrue_type", ValueType: "string"},
							"do_not_accrue":     {DisplayName: "do_not_accrue", ValueType: "boolean"},
							"max_carryover":     {DisplayName: "max_carryover", ValueType: "string"},
							"default_allowance": {DisplayName: "default_allowance", ValueType: "string"},
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
