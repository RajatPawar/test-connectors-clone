package coda

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

const objectNameDocs = "docs"

// Only top-level docs are writable. Tables, columns and rows live under
// /docs/{docId}/tables/{tableIdOrName}/..., which the connector cannot address
// (and tables/columns have no create/update endpoints at all).
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
