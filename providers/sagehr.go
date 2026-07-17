package providers

const SageHR Provider = "sageHR"

func init() {
	// Sage HR Connector Configuration
	SetInfo(SageHR, ProviderInfo{
		DisplayName: "Sage HR",
		AuthType:    ApiKey,
		BaseURL:     "https://{{.workspace}}.sage.hr/api",
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
					DisplayName:  "Subdomain",
					DefaultValue: "subdomain",
					DocsURL:      "https://developer.sage.com/hr/docs/v1.0.0/guides/get-started/quick-start",
				},
			},
		},
	})
}
