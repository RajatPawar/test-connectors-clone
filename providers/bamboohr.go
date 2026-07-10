package providers

const BambooHR Provider = "bambooHR"

func init() {
	// BambooHR configuration
	// Auth: HTTP Basic — API key as username, arbitrary/blank password.
	// https://documentation.bamboohr.com/reference/create-employee (Credentials: Basic, username:password)
	SetInfo(BambooHR, ProviderInfo{
		DisplayName: "BambooHR",
		AuthType:    Basic,
		// Base URL from the OpenAPI spec's servers[].url: https://{companyDomain}.bamboohr.com
		BaseURL: "https://{{.workspace}}.bamboohr.com",
		BasicOpts: &BasicAuthOpts{
			DocsURL: "https://documentation.bamboohr.com/docs/getting-started",
		},
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			Proxy:     true,
			Read:      true,
			Subscribe: false,
			Write:     false,
		},
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:         "workspace",
					DisplayName:  "Company Domain",
					DefaultValue: "yourcompany",
					DocsURL:      "https://documentation.bamboohr.com/docs/getting-started",
				},
			},
		},
	})
}
