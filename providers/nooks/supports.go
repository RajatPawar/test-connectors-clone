package nooks

import (
	"fmt"
	"slices"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// readObjects are the 11 list-able GET endpoints confirmed in Nooks' OpenAPI
// spec (docs/openapi_spec.json paths). Names are copied verbatim (exact case)
// from the URL path segments / docs/schemas.json object keys.
//
// emailTemplate is deliberately excluded: the spec only exposes
// GET /emailTemplate/{id} (singular, no list endpoint), so there is nothing to
// page through -- it fails the "no synthetic objects" naming rule.
// nolint:gochecknoglobals
var readObjects = []string{
	"calls",
	"tasks",
	"users",
	"emails",
	"accounts",
	"mailboxes",
	"prospects",
	"sequences",
	"sequenceSteps",
	"sequenceStates",
	"callDispositions",
}

// SupportedReadObjects returns the list of objects this connector can read,
// used by the capturekit live-read shim (test/nooks/read/main.go).
func SupportedReadObjects() []string {
	return slices.Clone(readObjects)
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
