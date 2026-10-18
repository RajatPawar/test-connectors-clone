package clarify

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

	responseSchemas := testutils.DataFromFile(t, "schemas.json")
	responseSchemasFirstPage := testutils.DataFromFile(t, "schemas-first-page.json")
	responseSchemasLastPage := testutils.DataFromFile(t, "schemas-last-page.json")
	responseUnauthorized := testutils.DataFromFile(t, "error-unauthorized.json")

	schemasServer := mockserver.Conditional{
		Setup: mockserver.ContentJSON(),
		If: mockcond.And{
			mockcond.MethodGET(),
			mockcond.Path("/v1/workspaces/acme/schemas"),
		},
		Then: mockserver.Response(http.StatusOK, responseSchemas),
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
			Input:      []string{"c_project"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"c_project": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Users are described without a request",
			Input:      []string{"users"},
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
							},
							"last_active_at": {
								DisplayName:  "Last Active At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
							},
							"is_service_account": {
								DisplayName:  "Is Service Account",
								ValueType:    common.ValueTypeBoolean,
								ProviderType: "boolean",
							},
							"_updated_at": {
								DisplayName:  "Updated At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
								ReadOnly:     new(true),
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:   "Deal is described by its entity schema",
			Input:  []string{"deal"},
			Server: schemasServer.Server(),
			// The to-many relationship is not an attribute Read returns.
			Comparator: func(serverURL string, actual, expected *common.ListObjectMetadataResult) *testutils.CompareResult {
				result := testconn.ComparatorSubsetMetadata(serverURL, actual, expected)
				if _, ok := actual.Result["deal"].Fields["people"]; ok {
					result.AddDiff("deal lists people, which records do not carry")
				}

				return result
			},
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"deal": {
						DisplayName: "Deals",
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
								ProviderType: "uuid",
								ReadOnly:     new(true),
							},
							"_updated_at": {
								DisplayName:  "Last updated",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
								ReadOnly:     new(true),
							},
							"_created_by": {
								DisplayName:  "Created By",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"name": {
								DisplayName:  "Name",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"description": {
								DisplayName:  "Description",
								ValueType:    common.ValueTypeString,
								ProviderType: "markdown",
							},
							"stage": {
								DisplayName:  "Stage",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								Values: []common.FieldValue{
									{Value: "Nurture", DisplayValue: "Nurture"},
									{Value: "Won", DisplayValue: "Won"},
									{Value: "Lost", DisplayValue: "Lost"},
								},
							},
							"amount": {
								DisplayName:  "Amount",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "number",
							},
							"close_date": {
								DisplayName:  "Close date",
								ValueType:    common.ValueTypeDate,
								ProviderType: "date",
							},
							"deal_churned": {
								DisplayName:  "Deal churned",
								ValueType:    common.ValueTypeBoolean,
								ProviderType: "boolean",
							},
							"labels": {
								DisplayName:  "Labels",
								ValueType:    common.ValueTypeOther,
								ProviderType: "object",
							},
							"deal_priority": {
								DisplayName:  "Priority",
								ValueType:    common.ValueTypeMultiSelect,
								ProviderType: "array",
								Values: []common.FieldValue{
									{Value: "High", DisplayValue: "High"},
									{Value: "Low", DisplayValue: "Low"},
								},
							},
							"summary": {
								DisplayName: "Summary",
								ValueType:   common.ValueTypeOther,
							},
							"company_id": {
								DisplayName:  "Company",
								ValueType:    common.ValueTypeReference,
								ProviderType: "many-to-one",
								ReferenceTo:  []string{"company"},
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			// The first page links to http://api.clarify.ai; following it as given would leave the mock server.
			Name:  "Schemas next link is followed on the request's scheme and host",
			Input: []string{"deal", "task"},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{{
					If: mockcond.And{
						mockcond.Path("/v1/workspaces/acme/schemas"),
						mockcond.QueryParam("page[cursor]", "c2NoZW1hcy0y"),
					},
					Then: mockserver.Response(http.StatusOK, responseSchemasLastPage),
				}, {
					If: mockcond.And{
						mockcond.Path("/v1/workspaces/acme/schemas"),
						mockcond.QueryParamsMissing("page[cursor]"),
					},
					Then: mockserver.Response(http.StatusOK, responseSchemasFirstPage),
				}},
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"deal": {
						DisplayName: "Deals",
						Fields: map[string]common.FieldMetadata{
							"name": {
								DisplayName:  "Name",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
						},
					},
					"task": {
						DisplayName: "Tasks",
						Fields: map[string]common.FieldMetadata{
							"due_date": {
								DisplayName:  "Due date",
								ValueType:    common.ValueTypeDate,
								ProviderType: "date",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:       "Object without an entity schema is reported per object",
			Input:      []string{"person", "meeting"},
			Server:     schemasServer.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"person": {
						DisplayName: "People",
						Fields: map[string]common.FieldMetadata{
							"job_title": {
								DisplayName:  "Job title",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
						},
					},
				},
				Errors: map[string]error{
					"meeting": mockutils.ExpectedSubsetErrors{common.ErrMissingMetadata},
				},
			},
		},
		{
			Name:  "Failed schemas request fails the schema objects only",
			Input: []string{"deal", "task", "users"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
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
							},
						},
					},
				},
				Errors: map[string]error{
					"deal": mockutils.ExpectedSubsetErrors{
						common.ErrAccessToken,
						testutils.StringError("Invalid API key"),
					},
					"task": mockutils.ExpectedSubsetErrors{common.ErrAccessToken},
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
	if err == nil {
		t.Fatal("expected an error without the workspace")
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
