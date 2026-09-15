package providers

// Nooks is the identifier for the Nooks provider.
// Base URL and auth scheme are per Nooks' OpenAPI spec (docs/openapi_spec.json):
// servers[0].url = https://partner-api.nooks.in/v1, securitySchemes.BearerAuth.
const Nooks Provider = "Nooks"

//nolint:lll
func init() {
	// Nooks configuration
	SetInfo(Nooks, ProviderInfo{
		DisplayName: "Nooks",
		AuthType:    ApiKey,
		BaseURL:     "https://partner-api.nooks.in/v1",
		ApiKeyOpts: &ApiKeyOpts{
			// The Authorization header accepts either a long-lived nooks-api-... key
			// (Developer Settings -> API Keys) or an OAuth2 access token issued by
			// https://oauth.nooks.in -- both formats are validated on the same header,
			// so this is modeled as a Bearer-prefixed API key.
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
