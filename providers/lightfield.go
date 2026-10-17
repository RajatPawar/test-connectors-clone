package providers

const Lightfield Provider = "lightfield"

func init() {
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
		// TODO: Media (icon and logo) is not set: no Lightfield assets are uploaded to the media host yet.
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
