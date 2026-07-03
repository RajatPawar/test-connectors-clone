package providers

const BambooHR Provider = "bambooHR"

func init() {
	SetInfo(BambooHR, ProviderInfo{
		DisplayName: "BambooHR",
		AuthType:    Basic,
		BaseURL:     "https://{{.company}}.bamboohr.com",
		BasicOpts: &BasicAuthOpts{
			ApiKeyAsBasic: true,
			ApiKeyAsBasicOpts: &ApiKeyAsBasicOpts{
				FieldUsed: UsernameField,
			},
			DocsURL: "https://documentation.bamboohr.com/docs/getting-started#authentication",
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
		Media: &Media{
			DarkMode: &MediaTypeDarkMode{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1/media/bamboohr.com/icon.png",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1/media/bamboohr.com/logo.png",
			},
			Regular: &MediaTypeRegular{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1/media/bamboohr.com/icon.png",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1/media/bamboohr.com/logo.png",
			},
		},
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					// NOTE: named "company" (not the usual "workspace" convention) to match the
					// real test credentials file, which stores this value at metadata.company.
					// See forge_notes.md for details — a prior round's BaseURL rendered as
					// "https://.bamboohr.com" (empty subdomain) because of this exact mismatch.
					Name:         "company",
					DisplayName:  "Company Domain",
					DefaultValue: "mycompany",
					DocsURL:      "https://documentation.bamboohr.com/docs/getting-started#authentication",
				},
			},
		},
	})
}
