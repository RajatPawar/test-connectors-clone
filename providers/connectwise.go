package providers

// ConnectWise PSA connector.
// TODO: The spec server is https://na.myconnectwise.net/v4_6_release/apis/3.0 (no api- prefix).
// However, cloud docs mention api-na.myconnectwise.net. If the na.* host fails, try api-na.*.
const ConnectWise Provider = "connectWise"

func init() {
	SetInfo(ConnectWise, ProviderInfo{
		DisplayName: "ConnectWise PSA",
		AuthType:    Basic,
		// Base URL from spec servers[0].url — region segment varies per tenant.
		// Cloud hosts: na.myconnectwise.net, eu.myconnectwise.net, au.myconnectwise.net.
		BaseURL: "https://{{.region}}.myconnectwise.net/v4_6_release/apis/3.0",
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
					Name:         "region",
					DisplayName:  "Region",
					DefaultValue: "na",
					Prompt:       "The regional subdomain for your ConnectWise instance (e.g. na, eu, au).",
				},
				{
					Name:        "clientId",
					DisplayName: "Client ID",
					Prompt:      "Your ConnectWise integration clientId GUID, required on every API call.",
				},
			},
		},
		Media: &Media{
			DarkMode: &MediaTypeDarkMode{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1723040083/media/connectwise.com_1723040082.png",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1723040106/media/connectwise.com_1723040105.svg",
			},
			Regular: &MediaTypeRegular{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1723040083/media/connectwise.com_1723040082.png",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1723040106/media/connectwise.com_1723040105.svg",
			},
		},
	})
}
