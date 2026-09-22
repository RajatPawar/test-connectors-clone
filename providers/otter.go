package providers

const Otter Provider = "otter"

func init() {
	// Otter.ai Public API — proxy (auth) connector.
	// Auth is a Bearer token in the Authorization header: "Authorization: Bearer YOUR_API_KEY".
	// BaseURL is the bare host with NO /v1 suffix; the documented curls hit
	// https://api.otter.ai/v1/... so the version segment belongs in the caller's path.
	// The host is fixed for all consumers — there is no region/tenant/subdomain
	// segment to template.
	// The Public API is Enterprise-only and must be enabled by an Otter account manager.
	// Rate limit: 10 requests/second per Enterprise user; exceeding it returns HTTP 429.
	// This is a passthrough proxy, so the caller is responsible for staying within it.
	SetInfo(Otter, ProviderInfo{
		DisplayName: "Otter.ai",
		AuthType:    ApiKey,
		BaseURL:     "https://api.otter.ai",
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
			// Proxy stays false on a net-new entry until Ampersand validates
			// the proxy against the real API.
			Proxy:     false,
			Read:      false,
			Subscribe: false,
			Write:     false,
		},
	})
}
