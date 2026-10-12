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

	responseOpportunityDefinitions := testutils.DataFromFile(t, "opportunities-definitions.json")
	responseForbidden := testutils.DataFromFile(t, "error-forbidden.json")

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object is reported per object",
			Input:      []string{"notes"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"notes": common.ErrObjectNotSupported,
				},
			},
		},
		{
			Name:       "Members come from the static catalog without any request",
			Input:      []string{"members"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"members": {
						DisplayName: "Members",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "id",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"$email": {
								DisplayName:  "$email",
								ValueType:    common.ValueTypeString,
								ProviderType: "EMAIL",
							},
							"$name": {
								DisplayName:  "$name",
								ValueType:    common.ValueTypeOther,
								ProviderType: "FULL_NAME",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Opportunities combine catalog properties with field and relationship definitions",
			Input: []string{"opportunities"},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/opportunities/definitions"),
					mockcond.Header(http.Header{"Lightfield-Version": []string{"2026-03-01"}}),
				},
				Then: mockserver.Response(http.StatusOK, responseOpportunityDefinitions),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"opportunities": {
						DisplayName: "Opportunities",
						Fields: map[string]common.FieldMetadata{
							"updatedAt": {
								DisplayName: "updatedAt",
								ValueType:   common.ValueTypeOther,
							},
							"$name": {
								DisplayName:  "Name",
								ValueType:    common.ValueTypeString,
								ProviderType: "TEXT",
								ReadOnly:     new(false),
								IsCustom:     new(false),
							},
							"$stage": {
								DisplayName:  "Stage",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "SINGLE_SELECT",
								ReadOnly:     new(false),
								IsCustom:     new(false),
								Values: []common.FieldValue{
									{Value: "opt_prospecting", DisplayValue: "Prospecting"},
									{Value: "opt_closed_won", DisplayValue: "Closed Won"},
								},
							},
							"$amount": {
								DisplayName:  "Amount",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "CURRENCY",
								ReadOnly:     new(false),
								IsCustom:     new(false),
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
								ReadOnly:     new(false),
								IsCustom:     new(true),
							},
							"$account": {
								DisplayName:  "Account",
								ValueType:    common.ValueTypeReference,
								ProviderType: "HAS_ONE",
								ReferenceTo:  []string{"accounts"},
								ReadOnly:     new(false),
								IsCustom:     new(false),
							},
							"$owner": {
								DisplayName:  "Owner",
								ValueType:    common.ValueTypeOther,
								ProviderType: "HAS_ONE",
								ReadOnly:     new(false),
								IsCustom:     new(false),
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:  "Definitions failure is reported per object while others succeed",
			Input: []string{"accounts", "members"},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/accounts/definitions"),
				Then:  mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"members": {
						DisplayName: "Members",
						Fields: map[string]common.FieldMetadata{
							"$role": {
								DisplayName:  "$role",
								ValueType:    common.ValueTypeString,
								ProviderType: "TEXT",
							},
						},
					},
				},
				Errors: map[string]error{
					"accounts": mockutils.ExpectedSubsetErrors{
						common.ErrForbidden,
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
