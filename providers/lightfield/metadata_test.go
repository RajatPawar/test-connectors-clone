package lightfield

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

	responseOpportunities := testutils.DataFromFile(t, "opportunities-definitions.json")
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
			Input:      []string{"tasks"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{"tasks": common.ErrObjectNotSupported},
			},
		},
		{
			Name:  "Opportunities are described by their definitions, members by their documented schema",
			Input: []string{"opportunities", "members"},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/opportunities/definitions"),
					mockcond.Header(http.Header{"Lightfield-Version": []string{"2026-03-01"}}),
				},
				Then: mockserver.Response(http.StatusOK, responseOpportunities),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"opportunities": {
						DisplayName: "Opportunities",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "ID",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"updatedAt": {
								DisplayName:  "Updated At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "string",
							},
							"$name": {
								DisplayName:  "Name",
								ValueType:    common.ValueTypeString,
								ProviderType: "TEXT",
								IsCustom:     new(false),
								FieldId:      new("ad_name"),
							},
							"$stage": {
								DisplayName:  "Stage",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "SINGLE_SELECT",
								IsCustom:     new(false),
								FieldId:      new("ad_stage"),
								Values: []common.FieldValue{
									{Value: "opt_prospecting", DisplayValue: "Prospecting"},
									{Value: "opt_closed_won", DisplayValue: "Closed Won"},
								},
							},
							"$amount": {
								DisplayName:  "Amount",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "CURRENCY",
								IsCustom:     new(false),
								FieldId:      new("ad_amount"),
							},
							"$summary": {
								DisplayName:  "Summary",
								ValueType:    common.ValueTypeString,
								ProviderType: "READONLY_MARKDOWN",
								ReadOnly:     new(true),
								IsCustom:     new(false),
							},
							"priority": {
								DisplayName:  "Priority",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "NUMBER",
								IsCustom:     new(true),
								FieldId:      new("ad_priority"),
							},
							"$account": {
								DisplayName:  "Account",
								ValueType:    common.ValueTypeReference,
								ProviderType: "HAS_ONE",
								IsCustom:     new(false),
								ReferenceTo:  []string{"accounts"},
							},
							"$owner": {
								DisplayName:  "Owner",
								ValueType:    common.ValueTypeReference,
								ProviderType: "HAS_ONE",
								IsCustom:     new(false),
							},
						},
					},
					"members": {
						DisplayName: "Members",
						Fields: map[string]common.FieldMetadata{
							"$email": {
								DisplayName:  "Email",
								ValueType:    common.ValueTypeString,
								ProviderType: "EMAIL",
							},
							"$user": {
								DisplayName:  "User",
								ValueType:    common.ValueTypeReference,
								ProviderType: "HAS_ONE",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Provider error on definitions is reported per object",
			Input: []string{"accounts"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"accounts": mockutils.ExpectedSubsetErrors{
						common.ErrAccessToken,
						testutils.StringError("[unauthorized] Invalid API key."),
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
