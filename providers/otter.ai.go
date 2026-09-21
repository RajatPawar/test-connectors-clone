package providers

// OtterAI is the identifier for the Otter.ai provider.
//
// NOTE: The repo's camelCase provider-id convention (e.g. openAI, instantlyAI)
// would suggest an id like `otterAI`. This run's build parameters fix the
// provider id to "Otter.ai" and the file name to otter.ai.go, so those values
// are used here to stay consistent with the run's tooling/registration. The
// naming conflict is tracked in run feedback (R2).
const OtterAI Provider = "Otter.ai"

func init() {
	// Otter.ai proxy (auth) connector.
	//
	// Auth: API key sent as a Bearer token — "Authorization: Bearer <API_KEY>"
	// (docs: Integrations → Developer, Enterprise workspaces only).
	//
	// BaseURL: bare host, no version suffix. Every example request uses
	// https://api.otter.ai/v1/... but the /v1 segment belongs to the caller's
	// path (builders choose the API version in their proxy calls), so the
	// catalog BaseURL is the smallest host. No per-tenant/region templating —
	// the host is fixed for all consumers.
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
		// Net-new provider: all support flags submitted OFF. Support.Proxy is
		// flipped to true only after the proxy is exercised against the real API
		// (make test-proxy) with a live Enterprise-tier key.
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			Proxy:     false,
			Read:      false,
			Write:     false,
			Subscribe: false,
		},
	})
}
