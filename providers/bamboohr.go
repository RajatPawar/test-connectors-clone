package providers

const BambooHR Provider = "bamboohr"

func init() {
	SetInfo(BambooHR, ProviderInfo{
		DisplayName: "BambooHR",
		AuthType:    Basic,
		BaseURL:     "https://{{.workspace}}.bamboohr.com",
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
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:        "workspace",
					DisplayName: "Company Domain",
					DocsURL:     "https://documentation.bamboohr.com/reference",
				},
			},
		},
	})
}
