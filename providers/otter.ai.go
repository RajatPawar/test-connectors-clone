package providers

const OtterAI Provider = "Otter.ai"

func init() {
	// Otter.ai Public API (Enterprise-only). Auth is an API key sent as a Bearer
	// token in the Authorization header. Base URL is the bare host; the documented
	// /v1 version segment belongs in the request path (e.g. GET /v1/conversations).
	// Rate limit: Enterprise plans are limited to 10 requests/second (429 on excess).
	SetInfo(OtterAI, ProviderInfo{
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
			Proxy:     true,
			Read:      false,
			Write:     false,
			Subscribe: false,
		},
	})
}
