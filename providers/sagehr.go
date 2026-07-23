package providers

// SageHR is the identifier for the Sage HR provider.
const SageHR Provider = "sageHR"

//nolint:lll
func init() {
	SetInfo(SageHR, ProviderInfo{
		DisplayName: "Sage People",
		AuthType:    ApiKey,
		// Each Sage HR company has its own subdomain: https://{workspace}.sage.hr.
		// The OpenAPI spec's server URL is https://subdomain.sage.hr/api — the
		// "subdomain" segment is templated here as {{.workspace}}.
		BaseURL: "https://{{.workspace}}.sage.hr/api",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name: "X-Auth-Token",
			},
			DocsURL: "https://developer.sage.com/hr/docs/v1.0.0/guides/get-started/quick-start",
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
					DisplayName:  "Sage HR Subdomain",
					Prompt:       "If your Sage HR URL is `https://mycompany.sage.hr`, then the subdomain is `mycompany`.",
					DefaultValue: "subdomain",
					DocsURL:      "https://developer.sage.com/hr/docs/v1.0.0/guides/get-started/quick-start",
				},
			},
		},
	})
}
