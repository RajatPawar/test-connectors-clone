package otterai

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestListObjectMetadata(t *testing.T) { // nolint:funlen
	t.Parallel()

	channelsResponse := testutils.DataFromFile(t, "channels.json")
	workspaceResponse := testutils.DataFromFile(t, "workspace.json")

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:  "Successfully describe channels and workspace",
			Input: []string{objectNameChannels, objectNameWorkspace},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/v1/channels"),
						Then: mockserver.Response(http.StatusOK, channelsResponse),
					},
					{
						If:   mockcond.Path("/v1/workspace"),
						Then: mockserver.Response(http.StatusOK, workspaceResponse),
					},
				},
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectNameChannels: {
						DisplayName: "Channels",
						Fields: map[string]common.FieldMetadata{
							"id":              {DisplayName: "id", ValueType: common.ValueTypeString},
							"name":            {DisplayName: "name", ValueType: common.ValueTypeString},
							"member_count":    {DisplayName: "member_count", ValueType: common.ValueTypeFloat},
							"owner":           {DisplayName: "owner", ValueType: common.ValueTypeOther},
							"discoverability": {DisplayName: "discoverability", ValueType: common.ValueTypeString},
						},
					},
					objectNameWorkspace: {
						DisplayName: "Workspace",
						Fields: map[string]common.FieldMetadata{
							"id":           {DisplayName: "id", ValueType: common.ValueTypeFloat},
							"name":         {DisplayName: "name", ValueType: common.ValueTypeString},
							"owner":        {DisplayName: "owner", ValueType: common.ValueTypeOther},
							"member_count": {DisplayName: "member_count", ValueType: common.ValueTypeFloat},
							"handle":       {DisplayName: "handle", ValueType: common.ValueTypeString},
							"type":         {DisplayName: "type", ValueType: common.ValueTypeString},
						},
					},
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
