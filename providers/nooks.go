package providers

// Nooks is the identifier for the Nooks provider.
const Nooks Provider = "Nooks"

//nolint:lll
func init() {
	// Nooks configuration
	SetInfo(Nooks, ProviderInfo{
		DisplayName: "Nooks",
		AuthType:    ApiKey,
		// Spec declares a single fixed production server, no per-tenant path segment;
		// the workspace is resolved from the bearer token itself.
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
