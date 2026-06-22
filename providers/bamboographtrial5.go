package providers

const BambooGraphTrial5 Provider = "bambooHR"

func init() {
	SetInfo(BambooGraphTrial5, ProviderInfo{
		DisplayName: "BambooHR",
		AuthType:    Basic,
		BaseURL:     "https://{{.companyDomain}}.bamboohr.com",
		BasicOpts: &BasicAuthOpts{
			// BambooHR uses the API key as the username in Basic Auth; password is ignored.
			ApiKeyAsBasic: true,
			ApiKeyAsBasicOpts: &ApiKeyAsBasicOpts{
				FieldUsed: UsernameField,
			},
			DocsURL: "https://documentation.bamboohr.com/docs/getting-started",
		},
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			Proxy:     false,
			Read:      true,
			Subscribe: false,
			Write:     false,
		},
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:         "companyDomain",
					DisplayName:  "Company Domain",
					DocsURL:      "https://documentation.bamboohr.com/docs/getting-started",
					DefaultValue: "mycompany",
				},
			},
		},
	})
}
