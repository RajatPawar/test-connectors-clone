package providers

const Lightfield Provider = "lightfield"

func init() {
	SetInfo(Lightfield, ProviderInfo{
		DisplayName: "Lightfield",
		AuthType:    ApiKey,
		// The OpenAPI spec declares no servers; every example request in the docs
		// targets https://api.lightfield.app/v1/... (e.g. using-the-api/api-keys).
		BaseURL: "https://api.lightfield.app",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "Bearer ",
			},
			DocsURL: "https://docs.lightfield.app/using-the-api/api-keys",
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
