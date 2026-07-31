package providers

// Nooks is the identifier for the Nooks provider (Sequencing/SEP API).
const Nooks Provider = "nooks"

func init() {
	// Nooks Connector Configuration.
	//
	// Auth: the Authorization header accepts either a long-lived, workspace-
	// scoped API key (prefixed `nooks-api-`, from Developer Settings -> API
	// Keys) or a short-lived OAuth 2.0 access token, on the same header. The
	// API auto-detects which format was sent. We model this as an ApiKey with
	// a Bearer-prefixed Authorization header, since the API key is the
	// simplest and most stable credential for a server-to-server sync (no
	// scopes, full workspace read access, no 1-hour expiry to manage).
	SetInfo(Nooks, ProviderInfo{
		DisplayName: "Nooks",
		AuthType:    ApiKey,
		// Single fixed production server (docs/openapi_spec.json `servers[0].url`).
		// No per-tenant subdomain or path variable: the workspace is resolved
		// from the token, not the URL.
		BaseURL: "https://partner-api.nooks.in/v1",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "Bearer ",
			},
			DocsURL: "https://www.nooks.in",
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
	})
}
