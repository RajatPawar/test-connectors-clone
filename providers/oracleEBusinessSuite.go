package providers

// OracleEBusinessSuite is the identifier for the Oracle E-Business Suite provider.
const OracleEBusinessSuite Provider = "oracleEBusinessSuite"

//nolint:lll
func init() {
	SetInfo(OracleEBusinessSuite, ProviderInfo{
		DisplayName: "Oracle E-Business Suite",
		AuthType:    Basic,
		// The Integrated SOA Gateway OpenAPI spec's server URL is
		// https://{instance}.oracle.com/webservices/rest; the "instance" segment is templated as {{.workspace}}.
		// TODO: the customer spec says each connection supplies its instance's base URL. If a customer's EBS
		// host is not under oracle.com, this template can't express it; confirm with live captures.
		BaseURL: "https://{{.workspace}}.oracle.com/webservices/rest",
		BasicOpts: &BasicAuthOpts{
			DocsURL: "https://docs.oracle.com/cd/E26401_01/doc.122/e20927/toc.htm",
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
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:         "workspace",
					DisplayName:  "EBS Instance",
					Prompt:       "If your EBS REST endpoint is `https://ebs-host.oracle.com/webservices/rest`, then the instance is `ebs-host`.",
					DefaultValue: "ebs-host",
					DocsURL:      "https://docs.oracle.com/cd/E26401_01/doc.122/e20927/toc.htm",
				},
			},
		},
	})
}
