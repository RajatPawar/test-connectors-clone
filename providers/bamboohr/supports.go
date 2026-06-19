package bamboohr

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

func supportedOperations() components.EndpointRegistryInput {
	// Objects derived from endpoint URL paths per docs/schemas.json.
	// Slash-prefixed paths preserved; no underscores substituted for slashes.
	readSupport := []string{
		"api/v1/employees",
		"webhooks",
		"fields",
		"api/v1/meta/timezones",
		"custom-reports",
		"api/v1_2/datasets",
	}

	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
