package providers

// Nooks is the identifier for the Nooks Sequencing API provider.
const Nooks Provider = "nooks"

//nolint:lll
func init() {
	SetInfo(Nooks, ProviderInfo{
		DisplayName: "Nooks",
		AuthType:    ApiKey,
		// The OpenAPI spec's only server is https://partner-api.nooks.in/v1 — a single
		// fixed production host with no per-tenant subdomain/path segment; the
		// workspace is determined entirely by the credential (see README).
		// Per repo convention (BaseURL must not embed API version info), the
		// "/v1" segment is added by the connector's path builder instead of
		// living here; the actual request URL is unchanged.
		BaseURL: "https://partner-api.nooks.in",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				// Nooks accepts either a workspace-scoped `nooks-api-` API key or an
				// OAuth2 access token on the same Authorization header (auto-detected).
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
