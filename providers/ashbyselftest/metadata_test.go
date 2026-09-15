package ashbyselftest

import (
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
)

func TestListObjectMetadata(t *testing.T) { // nolint:funlen
	t.Parallel()

	tests := []testroutines.Metadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object requested",
			Input:      []string{"unknown"},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Errors: map[string]error{
					"unknown": common.ErrObjectNotSupported,
				},
			},
		},
		{
			Name:       "Successfully describe candidate and jobPosting objects",
			Input:      []string{objectCandidate, objectJobPosting},
			Server:     mockserver.Dummy(),
			Comparator: testroutines.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					objectCandidate: {
						DisplayName: "Candidate",
						Fields: map[string]common.FieldMetadata{
							"id":         {DisplayName: "id", ValueType: "string", ProviderType: "uuid"},
							"name":       {DisplayName: "name", ValueType: "string", ProviderType: "string"},
							"updatedAt":  {DisplayName: "updatedAt", ValueType: "datetime", ProviderType: "date-time"},
							"profileUrl": {DisplayName: "profileUrl", ValueType: "string", ProviderType: "string"},
							"fraudStatus": {
								DisplayName:  "fraudStatus",
								ValueType:    "singleSelect",
								ProviderType: "string",
								Values: common.FieldValues{
									{Value: "Fraudulent", DisplayValue: "Fraudulent"},
									{Value: "NotFraudulent", DisplayValue: "NotFraudulent"},
									{Value: "Unsure", DisplayValue: "Unsure"},
									{Value: "Unreviewed", DisplayValue: "Unreviewed"},
									{Value: "PassedFraudCheck", DisplayValue: "PassedFraudCheck"},
								},
							},
						},
					},
					objectJobPosting: {
						DisplayName: "Job Posting",
						Fields: map[string]common.FieldMetadata{
							"id":        {DisplayName: "id", ValueType: "string", ProviderType: "uuid"},
							"title":     {DisplayName: "title", ValueType: "string", ProviderType: "string"},
							"updatedAt": {DisplayName: "updatedAt", ValueType: "datetime", ProviderType: "date-time"},
							"workplaceType": {
								DisplayName:  "workplaceType",
								ValueType:    "singleSelect",
								ProviderType: "string",
								Values: common.FieldValues{
									{Value: "OnSite", DisplayValue: "OnSite"},
									{Value: "Hybrid", DisplayValue: "Hybrid"},
									{Value: "Remote", DisplayValue: "Remote"},
								},
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
