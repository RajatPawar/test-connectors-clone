package tipalti

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

func TestListObjectMetadata(t *testing.T) { //nolint:funlen,gocognit,cyclop
	t.Parallel()

	responsePayees := testutils.DataFromFile(t, "payees.json")
	responsePayments := testutils.DataFromFile(t, "payments.json")

	tests := []testroutines.Metadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Successfully describe payees object",
			Input:      []string{"payees"},
			Server:     mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/payees"),
				Then:  mockserver.Response(http.StatusOK, responsePayees),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"payees": {
						DisplayName: "Payees",
						Fields: common.FieldsMetadata{
							"id": {
								DisplayName: "id",
							},
							"status": {
								DisplayName: "status",
							},
						},
					},
				},
			},
		},
		{
			Name:  "Successfully describe payments object",
			Input: []string{"payments"},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/v1/payments"),
				Then:  mockserver.Response(http.StatusOK, responsePayments),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"payments": {
						DisplayName: "Payments",
						Fields: common.FieldsMetadata{
							"id": {
								DisplayName: "id",
							},
							"amount": {
								DisplayName: "amount",
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
