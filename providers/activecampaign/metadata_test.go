package activecampaign

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

func TestListObjectMetadata(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	contactsResponse := testutils.DataFromFile(t, "contacts.json")
	dealsResponse := testutils.DataFromFile(t, "deals.json")

	tests := []testroutines.Metadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:  "Successfully describe contacts and deals from sampled responses",
			Input: []string{"contacts", "deals"},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{{
					If:   mockcond.Path("/api/3/contacts"),
					Then: mockserver.Response(http.StatusOK, contactsResponse),
				}, {
					If:   mockcond.Path("/api/3/deals"),
					Then: mockserver.Response(http.StatusOK, dealsResponse),
				}},
			}.Server(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"contacts": {
						DisplayName: "Contacts",
						FieldsMap: map[string]string{
							"id":        "id",
							"email":     "email",
							"firstName": "firstName",
							"lastName":  "lastName",
						},
					},
					"deals": {
						DisplayName: "Deals",
						FieldsMap: map[string]string{
							"id":    "id",
							"title": "title",
							"value": "value",
						},
					},
				},
				Errors: nil,
			},
			ExpectedErrs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ObjectMetadataConnector, error) {
				return constructTestConnector(tt.Server)
			})
		})
	}
}
