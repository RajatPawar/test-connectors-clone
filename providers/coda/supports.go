package coda

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// The catalog BaseURL is https://coda.io/apis; every documented request URL is
// https://coda.io/apis/v1/<path>.
const apiVersion = "v1"

const objectNameDocs = "docs"

// writeObjects have a top-level create (POST /<object>) and update (PATCH /<object>/{id}).
//
// Requested but not built:
//   - tables, columns: the API has no create/update endpoint for either, and every table/column
//     path needs a docId (GET /docs/{docId}/tables, GET /docs/{docId}/tables/{tableIdOrName}/columns).
//   - rows: POST /docs/{docId}/tables/{tableIdOrName}/rows and
//     PUT /docs/{docId}/tables/{tableIdOrName}/rows/{rowIdOrName} need builder-chosen doc and table
//     ids in the path, and insert is bulk-only (`rows: [...]`). Builders use the proxy for these.
func writeObjects() []string {
	return []string{objectNameDocs}
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(writeObjects(), ",")),
				Support:  components.WriteSupport,
			},
		},
	}
}
