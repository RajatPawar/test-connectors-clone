package providers

const OtterAI Provider = "Otter.ai"

func init() {
	// Otter.ai Connector Configuration
	SetInfo(OtterAI, ProviderInfo{
		DisplayName: "Otter.ai",
		AuthType:    ApiKey,
		// Docs + examples agree on a single fixed host with no per-tenant/region
		// segment; the version prefix ("v1") is applied per-request in the
		// connector, not baked into the catalog BaseURL.
		// https://help.otter.ai/hc/en-us/articles/36130822688279-Otter-ai-Public-API
		BaseURL: "https://api.otter.ai",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "Bearer ",
			},
			DocsURL: "https://help.otter.ai/hc/en-us/articles/36130822688279-Otter-ai-Public-API",
		},
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			// Proxy has not been validated in this round; only the read (deep)
			// scope is being built here.
			Proxy:     false,
			Read:      true,
			Subscribe: false,
			Write:     false,
		},
	})
}
