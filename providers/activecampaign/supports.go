package activecampaign

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// readSupport lists the v3 list endpoints the connector can read.
// Object names are the exact (lowercase) URL path segments; list responses
// nest the collection under a top-level plural key with the same name.
//
// Only contacts and deals expose native updated-at incremental filters
// (filters[updated_after]); accounts, campaigns, lists, tags and users have no
// updated-at filter and are always read in full. See README for details.
//nolint:gochecknoglobals
var readSupport = []string{
	"contacts",
	"deals",
	"accounts",
	"campaigns",
	"lists",
	"tags",
	"users",
}

// SupportedReadObjects returns every object name this connector can read. It is
// consumed by tooling such as the live read/capture binary at
// test/activeCampaign/read.
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
