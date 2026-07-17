package sagehr

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestListObjectMetadata(t *testing.T) { //nolint:funlen,gocognit,cyclop
	t.Parallel()

	responseTeams := testutils.DataFromFile(t, "teams_page1.json")
	responseDocumentsCategories := testutils.DataFromFile(t, "documents_categories.json")

	tests := []testroutines.Metadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:  "Successfully describe teams and documents/categories metadata",
			Input: []string{"teams", "documents/categories"},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/api/documents/categories"),
						Then: mockserver.Response(http.StatusOK, responseDocumentsCategories),
					},
					{
						If:   mockcond.Path("/api/teams"),
						Then: mockserver.Response(http.StatusOK, responseTeams),
					},
				},
			}.Server(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"teams": {
						DisplayName: "Teams",
						FieldsMap: map[string]string{
							"id":           "id",
							"name":         "name",
							"manager_ids":  "manager_ids",
							"employee_ids": "employee_ids",
						},
					},
					"documents/categories": {
						DisplayName: "Documents/Categories",
						FieldsMap: map[string]string{
							"id":              "id",
							"name":            "name",
							"documents_count": "documents_count",
						},
					},
				},
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

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		Workspace:           "test-workspace",
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		return nil, err
	}

	// for testing we want to redirect calls to our mock server
	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
