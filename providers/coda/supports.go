package coda

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

const objectNameDocs = "docs"

// Only top-level collections are writable. Tables and columns have no create/update endpoint in
// the Coda API v1, and rows live under /docs/{docId}/tables/{tableIdOrName}/rows, so they are
// reachable through the proxy only.
var writeSupport = []string{ // nolint:gochecknoglobals
	objectNameDocs,
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(writeSupport, ",")),
				Support:  components.WriteSupport,
			},
		},
	}
}
