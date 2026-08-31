package activecampaigncl

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

	dealsSingle := testutils.DataFromFile(t, "deals_single.json")
	contactsPage1 := testutils.DataFromFile(t, "contacts_page1.json")

	tests := []testroutines.Metadata{
		{
			Name:       "Describe an object present in the static schema (deals) and one requiring live sampling (contacts)", //nolint:lll
			Input:      []string{objectDeals, objectContacts},
			Comparator: testroutines.ComparatorSubsetMetadata,
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.Path("/api/3/deals"),
						Then: mockserver.Response(http.StatusOK, dealsSingle),
					},
					{
						If:   mockcond.Path("/api/3/contacts"),
						Then: mockserver.Response(http.StatusOK, contactsPage1),
					},
				},
			}.Server(),
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectDeals: {
						DisplayName: "Deals",
						Fields: map[string]common.FieldMetadata{
							"id":    {DisplayName: "id", ValueType: "string", ProviderType: "string"},
							"title": {DisplayName: "title", ValueType: "string", ProviderType: "string"},
							"value": {DisplayName: "value", ValueType: "string", ProviderType: "string"},
						},
					},
					objectContacts: {
						DisplayName: objectContacts,
						Fields: map[string]common.FieldMetadata{
							"id":        {DisplayName: "id", ValueType: "string"},
							"email":     {DisplayName: "email", ValueType: "string"},
							"firstName": {DisplayName: "firstName", ValueType: "string"},
							"lastName":  {DisplayName: "lastName", ValueType: "string"},
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
