package coda

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// rowsEndpointGlob matches the rows collection of any table in any doc:
//
//	docs/{docId}/tables/{tableIdOrName}/rows
//
// The doc and table are per-write context, so they travel in the object name
// (see handlers.go / README.md) rather than in connector metadata.
const rowsEndpointGlob = "docs/*/tables/*/rows"

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: rowsEndpointGlob,
				Support:  components.WriteSupport,
			},
		},
	}
}
