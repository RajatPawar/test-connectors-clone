package providers

const Coda Provider = "coda"

func init() {
	// Coda Configuration
	SetInfo(Coda, ProviderInfo{
		DisplayName: "Coda",
		AuthType:    ApiKey,
		// Base host taken from the OpenAPI spec `servers`/description
		// ("This API uses a base path of `https://docs.superhuman.com/apis/v1`") — Coda
		// has rebranded to Superhuman Docs; docs.superhuman.com is the canonical host
		// used throughout the current OpenAPI spec and code samples.
		// Per repo convention the base URL carries no version suffix; the `v1`
		// version segment is added by the connector's request builder.
		BaseURL: "https://docs.superhuman.com/apis",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "Bearer ",
			},
			DocsURL: "https://coda.io/developers/apis/v1#section/Introduction",
		},
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: true,
				Update: true,
				Upsert: true,
				Delete: false,
			},
			Proxy:     true,
			Read:      false,
			Subscribe: false,
			Write:     true,
		},
		Media: &Media{
			DarkMode: &MediaTypeDarkMode{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722459966/media/coda_1722459965.png",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722459898/media/coda_1722459896.svg",
			},
			Regular: &MediaTypeRegular{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722459941/media/coda_1722459941.svg",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722459917/media/coda_1722459916.svg",
			},
		},
	})
}
