package activecampaignoh

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// Object names — copied verbatim from the v3 REST endpoint URL paths
// (docs/openapi_spec.json) per CLAUDE.md naming rules. For every one of
// these, the URL path segment, the response envelope key, and the object
// name are identical (e.g. GET /api/3/deals -> {"deals": [...]}).
const (
	objectContacts  = "contacts"
	objectDeals     = "deals"
	objectAccounts  = "accounts"
	objectLists     = "lists"
	objectCampaigns = "campaigns"
	objectTags      = "tags"
	objectUsers     = "users"
	objectDealTasks = "dealTasks"
)

// SupportedReadObjects lists every object this connector can read. Exported
// so the live-read capture shim (test/activecampaignoh/read/main.go) can
// drive capturekit without duplicating the object list.
func SupportedReadObjects() []string {
	return []string{
		objectContacts,
		objectDeals,
		objectAccounts,
		objectLists,
		objectCampaigns,
		objectTags,
		objectUsers,
		objectDealTasks,
	}
}

func supportedOperations() components.EndpointRegistryInput {
	readSupport := SupportedReadObjects()

	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
