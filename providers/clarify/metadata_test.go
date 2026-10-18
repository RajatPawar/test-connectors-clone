package clarify

import (
	"errors"
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

	responseSchemas := testutils.DataFromFile(t, "schemas.json")
	responseRateLimit := testutils.DataFromFile(t, "error-rate-limit.json")

	schemasServer := func() *mockserver.Conditional {
		return &mockserver.Conditional{
			Setup: mockserver.ContentJSON(),
			If: mockcond.And{
				mockcond.MethodGET(),
				mockcond.Path("/v1/workspaces/acme/schemas"),
			},
			Then: mockserver.Response(http.StatusOK, responseSchemas),
		}
	}

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object is reported per object",
			Input:      []string{"c_project", "users"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"users": {
						DisplayName: "Users",
						Fields: map[string]common.FieldMetadata{
							"email": {
								DisplayName:  "Email",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"_updated_at": {
								DisplayName:  "Updated At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "string:date-time",
								ReadOnly:     new(true),
							},
						},
					},
				},
				Errors: map[string]error{
					"c_project": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Objects are described by their entity schema",
			Input:      []string{"person", "deal"},
			Server:     schemasServer().Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"person": {
						DisplayName: "People",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "ID",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"_id": {
								DisplayName:  "Record ID",
								ValueType:    common.ValueTypeString,
								ProviderType: "string:uuid",
								ReadOnly:     new(true),
							},
							"_updated_at": {
								DisplayName:  "Updated At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "string:date-time",
								ReadOnly:     new(true),
							},
							"name": {
								DisplayName:  "Name",
								ValueType:    common.ValueTypeOther,
								ProviderType: "object",
							},
							"job_title": {
								DisplayName:  "Job title",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"first_email_interaction": {
								DisplayName:  "First email interaction",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "string:date-time",
								ReadOnly:     new(true),
							},
							"company_id": {
								DisplayName:  "Company",
								ValueType:    common.ValueTypeReference,
								ProviderType: "string",
								ReferenceTo:  []string{"company"},
							},
						},
					},
					"deal": {
						DisplayName: "Deals",
						Fields: map[string]common.FieldMetadata{
							"amount": {
								DisplayName:  "Amount",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "number",
							},
							"stage": {
								DisplayName:  "Stage",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								Values: []common.FieldValue{
									{Value: "Negotiation", DisplayValue: "Negotiation"},
									{Value: "Proposal", DisplayValue: "Proposal"},
									{Value: "Won", DisplayValue: "Won"},
									{Value: "Lost", DisplayValue: "Lost"},
								},
							},
							"close_date": {
								DisplayName:  "Close date",
								ValueType:    common.ValueTypeDate,
								ProviderType: "string:date",
							},
							"is_renewal": {
								DisplayName:  "Renewal",
								ValueType:    common.ValueTypeBoolean,
								ProviderType: "boolean",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:   "To-many relationship properties are not listed, being absent from record attributes",
			Input:  []string{"person"},
			Server: schemasServer().Server(),
			Comparator: func(serverURL string, actual, expected *common.ListObjectMetadataResult) *testutils.CompareResult {
				result := testconn.ComparatorSubsetMetadata(serverURL, actual, expected)
				if _, ok := actual.Result["person"].Fields["deals"]; ok {
					result.AddDiff("Result[person] lists the to-many relationship deals")
				}

				return result
			},
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"person": {DisplayName: "People"},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:       "Object without an entity schema is reported per object",
			Input:      []string{"task", "deal"},
			Server:     schemasServer().Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"deal": {DisplayName: "Deals"},
				},
				Errors: map[string]error{
					"task": mockutils.ExpectedSubsetErrors{common.ErrMissingMetadata},
				},
			},
		},
		{
			Name:  "Failed schemas request fails every object that needs it, but not users",
			Input: []string{"person", "users"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusTooManyRequests, responseRateLimit),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"users": {DisplayName: "Users"},
				},
				Errors: map[string]error{
					"person": mockutils.ExpectedSubsetErrors{
						common.ErrLimitExceeded,
						testutils.StringError("API rate limit exceeded"),
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

func TestNewConnectorRequiresWorkspace(t *testing.T) {
	t.Parallel()

	_, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
	})
	if !errors.Is(err, common.ErrMissingWorkspace) {
		t.Fatalf("expected %v, got %v", common.ErrMissingWorkspace, err)
	}
}

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
		Workspace:           "acme",
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
