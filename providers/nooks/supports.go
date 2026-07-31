package nooks

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// Object names, copied verbatim from docs/openapi_spec.json path segments
// (e.g. "/prospects" -> "prospects").
const (
	objectNameProspects      = "prospects"
	objectNameAccounts       = "accounts"
	objectNameUsers          = "users"
	objectNameSequences      = "sequences"
	objectNameSequenceStates = "sequenceStates"
	objectNameTasks          = "tasks"
	objectNameEmails         = "emails"
	objectNameCalls          = "calls"
)

// readObjectNames are the objects this connector exposes for read.
//
// docs/openapi_spec.json defines a larger set of list endpoints (also
// mailboxes, sequenceSteps, callDispositions), but this run is scoped to the
// advisor-prioritized core CRM/SEP entities. mailboxes, sequenceSteps, and
// callDispositions are readable via the same pattern and could be added
// later; emailTemplate has no list endpoint at all (singular
// GET /emailTemplate/{id} only, reachable via a sequence step's `template`
// reference) so it is not a listable object and is intentionally omitted.
var readObjectNames = []string{ //nolint:gochecknoglobals
	objectNameProspects,
	objectNameAccounts,
	objectNameUsers,
	objectNameSequences,
	objectNameSequenceStates,
	objectNameTasks,
	objectNameEmails,
	objectNameCalls,
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readObjectNames, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}

// SupportedReadObjects returns the objects this connector supports reading.
// Used by the live-capture shim (test/nooks/read/main.go).
func SupportedReadObjects() []string {
	return readObjectNames
}
