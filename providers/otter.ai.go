package providers

const OtterAI Provider = "Otter.ai"

func init() {
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
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			Proxy:     true,
			Read:      false,
			Subscribe: false,
			Write:     false,
		},
	})
}
