package activecampaignoh

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

	tagsPage1 := testutils.DataFromFile(t, "tags_page1.json")
	contactsPage1 := testutils.DataFromFile(t, "contacts_page1.json")

	tests := []testroutines.Metadata{
		{
			Name:       "Describe a static-schema object (tags) and a sampled object (contacts)",
			Input:      []string{objectTags, objectContacts},
			Comparator: testroutines.ComparatorSubsetMetadata,
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/api/3/contacts"),
						Then: mockserver.Response(http.StatusOK, contactsPage1),
					},
					{
						If:   mockcond.Path("/api/3/tags"),
						Then: mockserver.Response(http.StatusOK, tagsPage1),
					},
				},
			}.Server(),
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectTags: {
						DisplayName: "Tags",
						Fields: map[string]common.FieldMetadata{
							"id":      {DisplayName: "id", ValueType: "string", ProviderType: "string"},
							"tag":     {DisplayName: "tag", ValueType: "string", ProviderType: "string"},
							"cdate":   {DisplayName: "cdate", ValueType: "string", ProviderType: "string"},
							"links":   {DisplayName: "links", ValueType: "other", ProviderType: "object"},
							"tagType": {DisplayName: "tagType", ValueType: "string", ProviderType: "string"},
						},
					},
					objectContacts: {
						DisplayName: objectContacts,
						Fields: map[string]common.FieldMetadata{
							"id":        {DisplayName: "id", ValueType: "string"},
							"email":     {DisplayName: "email", ValueType: "string"},
							"firstName": {DisplayName: "firstName", ValueType: "string"},
							"lastName":  {DisplayName: "lastName", ValueType: "string"},
							"cdate":     {DisplayName: "cdate", ValueType: "string"},
							"udate":     {DisplayName: "udate", ValueType: "string"},
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
