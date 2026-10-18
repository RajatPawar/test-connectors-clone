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
	responseWorkspaceNotFound := testutils.DataFromFile(t, "error-workspace-not-found.json")

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object is reported per object",
			Input:      []string{"campaigns"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"campaigns": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Users are described without a request",
			Input:      []string{"user"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"user": {
						DisplayName: "Users",
						Fields: map[string]common.FieldMetadata{
							"email": {
								DisplayName:  "Email",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"_updated_at": {
								DisplayName:  "Updated at",
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
			Name:  "Person is described by its entity schema",
			Input: []string{"person"},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/workspaces/acme/schemas"),
				},
				Then: mockserver.Response(http.StatusOK, responseSchemas),
			}.Server(),
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
							"_created_by": {
								DisplayName:  "Created by",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								ReadOnly:     new(true),
							},
							"_updated_at": {
								DisplayName:  "Last updated",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
								ReadOnly:     new(true),
								IsRequired:   new(false),
							},
							"name": {
								DisplayName: "Name",
								ValueType:   common.ValueTypeOther,
								IsRequired:  new(false),
							},
							"email_addresses": {
								DisplayName: "Email addresses",
								ValueType:   common.ValueTypeOther,
								IsRequired:  new(true),
							},
							"job_title": {
								DisplayName:  "Job title",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
								IsRequired:   new(false),
							},
							"company_id": {
								DisplayName:  "Company",
								ValueType:    common.ValueTypeReference,
								ProviderType: "string",
								IsRequired:   new(false),
								ReferenceTo:  []string{"company"},
							},
							"persona": {
								DisplayName:  "Persona",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								IsRequired:   new(false),
								Values: []common.FieldValue{
									{Value: "Champion", DisplayValue: "Champion"},
									{Value: "Decision maker", DisplayValue: "Decision maker"},
								},
							},
							"labels": {
								DisplayName:  "Labels",
								ValueType:    common.ValueTypeMultiSelect,
								ProviderType: "object",
								IsRequired:   new(false),
								Values: []common.FieldValue{
									{Value: "Newsletter", DisplayValue: "Newsletter"},
									{Value: "Customer", DisplayValue: "Customer"},
								},
							},
							"last_email_interaction": {
								DisplayName:  "Last email interaction",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
								IsRequired:   new(false),
								ReadOnly:     new(true),
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Object without an entity schema is reported per object",
			Input: []string{"deal"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusOK, responseSchemas),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"deal": mockutils.ExpectedSubsetErrors{common.ErrMissingExpectedValues},
				},
			},
		},
		{
			Name:  "Failed schemas request fails schema objects but not users",
			Input: []string{"person", "user"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusNotFound, responseWorkspaceNotFound),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"user": {
						DisplayName: "Users",
						Fields: map[string]common.FieldMetadata{
							"roles": {
								DisplayName:  "Roles",
								ValueType:    common.ValueTypeOther,
								ProviderType: "object",
								ReadOnly:     new(true),
							},
						},
					},
				},
				Errors: map[string]error{
					"person": mockutils.ExpectedSubsetErrors{
						common.ErrNotFound,
						testutils.StringError("Workspace not found"),
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

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
		Workspace:           "acme",
	})
	if err != nil {
		return nil, err
	}

	connector.SetUnitTestMockServerBaseUrl(serverURL)

	return connector, nil
}
