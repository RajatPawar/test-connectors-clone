package providers

// SageHR is the identifier for the Sage HR provider.
const SageHR Provider = "sageHR"

//nolint:lll
func init() {
	SetInfo(SageHR, ProviderInfo{
		DisplayName: "Sage HRIS",
		AuthType:    ApiKey,
		// Each Sage HR company has its own subdomain: https://{workspace}.sage.hr.
		// The OpenAPI spec's server URL is https://subdomain.sage.hr/api — the
		// "subdomain" segment is templated here as {{.workspace}}.
		BaseURL: "https://{{.workspace}}.sage.hr/api",
		// TODO: human feedback (b52064c0-8c04-4a72-b294-9b1c56774572) asked to
		// label the API-key credential input "API Token" and to set a connector
		// description "Sage HR HRIS integration". Neither ApiKeyOpts nor
		// ProviderInfo (providers/types.gen.go) exposes a field for a
		// user-facing display name on the ApiKey credential input, or a
		// free-text provider description — CustomAuthInput.DisplayName is the
		// only such field, and it only applies under AuthType: Custom, which
		// is out of scope here (would require replacing the X-Auth-Token
		// apiKey auth entirely). Left unimplemented; flagged in the PR
		// description for a human/schema-owner decision. The requested
		// description text is recorded in README.md in the meantime.
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name: "X-Auth-Token",
			},
			DocsURL: "https://developer.sage.com/hr/docs/v1.0.0/guides/get-started/quick-start",
		},
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			Proxy:     false,
			Read:      true,
			Subscribe: false,
			Write:     false,
		},
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:         "workspace",
					DisplayName:  "Sage HR Subdomain",
					Prompt:       "If your Sage HR URL is `https://mycompany.sage.hr`, then the subdomain is `mycompany`.",
					DefaultValue: "subdomain",
					DocsURL:      "https://developer.sage.com/hr/docs/v1.0.0/guides/get-started/quick-start",
				},
			},
		},
	})
}
