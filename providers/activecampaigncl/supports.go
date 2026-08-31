package activecampaigncl

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// Object names are copied verbatim from the v3 REST URL path segments per
// CLAUDE.md naming rules — e.g. pipelines live at /dealGroups, not /pipelines,
// so the object is named "dealGroups".
//
// TODO: objects intentionally NOT included this round, scoped out by the PM
// plan's priority list (contacts, deals, accounts, campaigns, users, lists,
// tags, dealGroups):
//   - dealStages, dealTasks — deal-pipeline-adjacent GET list endpoints that
//     exist in the spec but weren't in the priority list; candidates for a
//     future round.
//   - ecomCustomers, ecomOrders — documented GET list endpoints, but require
//     a connected ecommerce store (connectionid) and return empty in
//     unconfigured accounts (confirmed via connie_context vetted facts).
//
// TODO: contacts pagination could use the documented id_greater +
// orders[id]=ASC keyset strategy instead of limit/offset for better
// performance on large accounts; offset works correctly today, so this is a
// future enhancement, not a correctness gap.
const (
	objectContacts   = "contacts"
	objectDeals      = "deals"
	objectAccounts   = "accounts"
	objectCampaigns  = "campaigns"
	objectUsers      = "users"
	objectLists      = "lists"
	objectTags       = "tags"
	objectDealGroups = "dealGroups"
)

// readSupport is the complete list of objects this connector can read. It is
// also exposed via SupportedReadObjects for tooling (e.g. the live
// read/capture binary at test/activecampaigncl/read) that needs to iterate
// every supported object.
//
//nolint:gochecknoglobals
var readSupport = []string{
	objectContacts,
	objectDeals,
	objectAccounts,
	objectCampaigns,
	objectUsers,
	objectLists,
	objectTags,
	objectDealGroups,
}

// SupportedReadObjects returns every object name this connector can read.
func SupportedReadObjects() []string {
	return append([]string(nil), readSupport...)
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
