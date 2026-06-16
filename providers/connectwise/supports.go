package connectwise

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// readObjects lists the objects exposed for read.
// Names are the keys used in schemas.json (which may differ from their URL path).
// nolint:gochecknoglobals
var readObjects = []string{
	"service/tickets",
	"companies",
	"contacts",
	"projects",
	"activities",
	"time/entries",
	"invoices",
	"schedule/entries",
	"opportunities",
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readObjects, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
