package providers

// Nooks is the identifier for the Nooks provider.
const Nooks Provider = "Nooks"

//nolint:lll
func init() {
	SetInfo(Nooks, ProviderInfo{
		DisplayName: "Nooks",
		AuthType:    ApiKey,
		// Server URL from openapi_spec.json `servers` — copied verbatim, no templated segments.
		BaseURL: "https://partner-api.nooks.in/v1",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "Bearer ",
			},
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
