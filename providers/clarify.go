package providers

const Clarify Provider = "clarify"

func init() {
	SetInfo(Clarify, ProviderInfo{
		DisplayName: "Clarify",
		AuthType:    ApiKey,
		// Every endpoint is under /v1/workspaces/{workspace}, e.g.
		// https://api.clarify.ai/v1/workspaces/acme/objects/person/resources.
		// The deep connector adds that prefix from the `workspace` metadata; proxy callers include it.
		BaseURL: "https://api.clarify.ai",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "api-key ",
			},
			DocsURL: "https://developer.clarify.ai/docs/getting-started/authentication#api-keys",
		},
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:        "workspace",
					DisplayName: "Workspace Slug",
					DocsURL:     "https://developer.clarify.ai/docs/getting-started/key-concepts#workspace",
					Prompt: "The workspace slug: the {slug} segment of your Clarify workspace URL, " +
						"also shown under the user avatar menu.",
				},
			},
		},
		Media: &Media{
			DarkMode: &MediaTypeDarkMode{
				IconURL: "https://developer.clarify.ai/favicon-dark.png",
				LogoURL: "https://developer.clarify.ai/logo-dark.svg",
			},
			Regular: &MediaTypeRegular{
				IconURL: "https://developer.clarify.ai/favicon.png",
				LogoURL: "https://developer.clarify.ai/logo.svg",
			},
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
