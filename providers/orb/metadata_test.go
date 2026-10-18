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
	responseEmpty := testutils.DataFromFile(t, "empty.json")
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
			Input:      []string{"events"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"events": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Static objects come from the catalog without calling the provider",
			Input:      []string{"invoices", "credit_notes", "customers"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"invoices": {
						DisplayName: "Invoices",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName: "Id",
								ValueType:   common.ValueTypeString,
							},
							"invoice_date": {
								DisplayName: "Invoice Date",
								ValueType:   common.ValueTypeDateTime,
							},
							"status": {
								DisplayName:  "Status",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "enum<string>",
								Values: []common.FieldValue{
									{Value: "draft", DisplayValue: "draft"},
									{Value: "issued", DisplayValue: "issued"},
									{Value: "paid", DisplayValue: "paid"},
									{Value: "synced", DisplayValue: "synced"},
									{Value: "void", DisplayValue: "void"},
								},
							},
							"hidden_line_item_count": {
								DisplayName: "Hidden Line Item Count",
								ValueType:   common.ValueTypeInt,
							},
						},
					},
					"credit_notes": {
						DisplayName: "Credit Notes",
						Fields: map[string]common.FieldMetadata{
							"invoice_id": {
								DisplayName: "Invoice Id",
								ValueType:   common.ValueTypeReference,
								ReferenceTo: []string{"invoices"},
							},
						},
					},
					"customers": {
						DisplayName: "Customers",
						Fields: map[string]common.FieldMetadata{
							"auto_collection": {
								DisplayName: "Auto Collection",
								ValueType:   common.ValueTypeBoolean,
							},
							"billing_address": {
								DisplayName: "Billing Address",
								ValueType:   common.ValueTypeOther,
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Subscription fields missing from the catalog are sampled",
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
							// Catalog types are kept for sampled fields the catalog knows.
							"created_at": {
								DisplayName: "Created At",
								ValueType:   common.ValueTypeDateTime,
							},
							"active_plan_phase_order": {
								DisplayName: "Active Plan Phase Order",
								ValueType:   common.ValueTypeInt,
							},
							"price_intervals": {
								DisplayName: "Price Intervals",
								ValueType:   common.ValueTypeOther,
							},
						},
					},
					"items": {
						DisplayName: "Items",
						Fields: map[string]common.FieldMetadata{
							"archived_at": {
								DisplayName: "Archived At",
								ValueType:   common.ValueTypeDateTime,
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Account without subscriptions gets the catalog fields",
			Input: []string{"subscriptions"},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/subscriptions"),
				Then:  mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: func(serverURL string, actual, expected *common.ListObjectMetadataResult) *testutils.CompareResult {
				result := testconn.ComparatorSubsetMetadata(serverURL, actual, expected)
				if _, ok := actual.Result["subscriptions"].Fields["price_intervals"]; ok {
					result.AddDiff("price_intervals was not sampled, but is listed")
				}

				return result
			},
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"subscriptions": {
						DisplayName: "Subscriptions",
						Fields: map[string]common.FieldMetadata{
							"plan": {
								DisplayName: "Plan",
								ValueType:   common.ValueTypeOther,
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Failed sample is reported for subscriptions only",
			Input: []string{"subscriptions", "coupons"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
			Comparator: func(serverURL string, actual, expected *common.ListObjectMetadataResult) *testutils.CompareResult {
				result := testconn.ComparatorSubsetMetadata(serverURL, actual, expected)
				if _, ok := actual.Result["subscriptions"]; ok {
					result.AddDiff("Result[subscriptions] exists, but the object failed")
				}

				return result
			},
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"coupons": {
						DisplayName: "Coupons",
						Fields: map[string]common.FieldMetadata{
							"redemption_code": {
								DisplayName: "Redemption Code",
								ValueType:   common.ValueTypeString,
							},
						},
					},
				},
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
