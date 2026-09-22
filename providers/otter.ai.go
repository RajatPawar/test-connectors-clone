package providers

// OtterAI is the identifier for the Otter.ai provider.
const OtterAI Provider = "Otter.ai"

//nolint:lll
func init() {
	// Otter.ai configuration
	// Proxy (auth) connector: authenticated raw passthrough to Otter.ai's Public API.
	// The API is Enterprise-only and must be enabled by an Otter account manager;
	// auth is a Bearer token in the Authorization header.
	SetInfo(OtterAI, ProviderInfo{
		DisplayName: "Otter.ai",
		AuthType:    ApiKey,
		// Smallest host — the documented /v1 version segment belongs in the
		// caller's request path, not the catalog BaseURL. The host is constant
		// (no per-tenant/region subdomain), so nothing is templated.
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
			Proxy:     false,
			Read:      false,
			Subscribe: false,
			Write:     false,
		},
	})
}
