package orb

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestListObjectMetadata(t *testing.T) { // nolint:funlen
	t.Parallel()

	responseSubscriptions := testutils.DataFromFile(t, "subscriptions-sample.json")
	responseEmpty := testutils.DataFromFile(t, "subscriptions-empty.json")
	responseUnauthorized := testutils.DataFromFile(t, "error-unauthorized.json")

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object is reported per object",
			Input:      []string{"prices"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"prices": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Documented objects come from the static catalog",
			Input:      []string{"customers", "invoices", "credit_notes"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"customers": {
						DisplayName: "Customers",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "Id",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"created_at": {
								DisplayName:  "Created At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
							},
							"auto_collection": {
								DisplayName:  "Auto Collection",
								ValueType:    common.ValueTypeBoolean,
								ProviderType: "boolean",
							},
							"billing_address": {
								DisplayName:  "Billing Address",
								ValueType:    common.ValueTypeOther,
								ProviderType: "object",
							},
							"payment_provider": {
								DisplayName:  "Payment Provider",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "enum",
								Values: []common.FieldValue{
									{Value: "quickbooks", DisplayValue: "quickbooks"},
									{Value: "bill.com", DisplayValue: "bill.com"},
									{Value: "stripe_charge", DisplayValue: "stripe_charge"},
									{Value: "stripe_invoice", DisplayValue: "stripe_invoice"},
									{Value: "netsuite", DisplayValue: "netsuite"},
									{Value: "adyen", DisplayValue: "adyen"},
								},
							},
						},
					},
					"invoices": {
						DisplayName: "Invoices",
						Fields: map[string]common.FieldMetadata{
							"hidden_line_item_count": {
								DisplayName:  "Hidden Line Item Count",
								ValueType:    common.ValueTypeInt,
								ProviderType: "integer",
							},
							"line_items": {
								DisplayName:  "Line Items",
								ValueType:    common.ValueTypeOther,
								ProviderType: "array",
							},
						},
					},
					"credit_notes": {
						DisplayName: "Credit Notes",
						Fields: map[string]common.FieldMetadata{
							"type": {
								DisplayName:  "Type",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "enum",
								Values: []common.FieldValue{
									{Value: "refund", DisplayValue: "refund"},
									{Value: "adjustment", DisplayValue: "adjustment"},
								},
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Subscriptions are sampled from the first listed record",
			Input: []string{"subscriptions", "items"},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/subscriptions"),
					mockcond.QueryParam("limit", "1"),
				},
				Then: mockserver.Response(http.StatusOK, responseSubscriptions),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"subscriptions": {
						DisplayName: "Subscriptions",
						Fields: map[string]common.FieldMetadata{
							"id":              {DisplayName: "id", ValueType: common.ValueTypeString},
							"status":          {DisplayName: "status", ValueType: common.ValueTypeString},
							"customer":        {DisplayName: "customer", ValueType: common.ValueTypeOther},
							"net_terms":       {DisplayName: "net_terms", ValueType: common.ValueTypeFloat},
							"auto_collection": {DisplayName: "auto_collection", ValueType: common.ValueTypeBoolean},
						},
					},
					"items": {
						DisplayName: "Items",
						Fields: map[string]common.FieldMetadata{
							"archived_at": {
								DisplayName:  "Archived At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Subscriptions cannot be sampled from an empty account",
			Input: []string{"subscriptions", "plans"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"plans": {DisplayName: "Plans"},
				},
				Errors: map[string]error{
					"subscriptions": mockutils.ExpectedSubsetErrors{common.ErrMissingExpectedValues},
				},
			},
		},
		{
			Name:  "Subscriptions sampling error is reported per object",
			Input: []string{"subscriptions"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"subscriptions": mockutils.ExpectedSubsetErrors{common.ErrAccessToken},
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

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
