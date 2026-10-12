package providers

const Lightfield Provider = "lightfield"

func init() {
	// Every request must also carry the `Lightfield-Version` header.
	// The deep connector sets it; proxy callers must add it themselves.
	// https://docs.lightfield.app/using-the-api/api-keys
	SetInfo(Lightfield, ProviderInfo{
		DisplayName: "Lightfield",
		AuthType:    ApiKey,
		BaseURL:     "https://api.lightfield.app",
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
			Proxy:     false,
			Read:      false,
			Subscribe: false,
			Write:     false,
		},
	})
}
