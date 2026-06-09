package providers

const ConnectWise Provider = "connectWise"

func init() {
	// ConnectWise PSA (Manage) Configuration
	//
	// Auth is HTTP Basic where the username is "companyId+publicKey" and the
	// password is the privateKey. Credentials are collected by the platform.
	//
	// The site (region) is supplied as the `workspace` metadata value, e.g.
	// `api-na.myconnectwise.net`, `api-eu.myconnectwise.net`, `api-au.myconnectwise.net`,
	// or `api-staging.connectwisedev.com`. `v4_6_release` is the default codebase
	// and is redirected to the partner's PSA version server-side.
	// Docs: https://developer.connectwise.com/Products/ConnectWise_PSA/Developer_Guide
	SetInfo(ConnectWise, ProviderInfo{
		DisplayName: "ConnectWise PSA",
		AuthType:    Basic,
		BaseURL:     "https://{{.workspace}}/v4_6_release/apis/3.0",
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
			Write:     true,
		},
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:        "workspace",
					DisplayName: "ConnectWise PSA site",
					Prompt:      "The regional PSA host, e.g. api-na.myconnectwise.net",
				},
				{
					// clientId is a per-integration GUID required on every API call.
					// It is stable for the lifetime of a connection.
					Name:        "clientId",
					DisplayName: "Client ID",
					Prompt:      "The clientId GUID assigned to your ConnectWise integration",
				},
			},
		},
	})
}
