package nooks

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// Object names — copied verbatim from the Nooks OpenAPI spec's list-endpoint
// paths (calls, tasks, users, emails, accounts, mailboxes, prospects,
// sequences, sequenceSteps, sequenceStates, callDispositions). All 11 objects
// exposed in docs/schemas.json are implemented.
//
// emailTemplate is NOT included: the spec only exposes GET /emailTemplate/{id}
// (singular, no list endpoint) — reached via a sequence step's `template`
// reference. Per the "no synthetic objects" naming rule, this connector does
// not invent an `emailTemplates` collection for it.
const (
	objectCalls            = "calls"
	objectTasks            = "tasks"
	objectUsers            = "users"
	objectEmails           = "emails"
	objectAccounts         = "accounts"
	objectMailboxes        = "mailboxes"
	objectProspects        = "prospects"
	objectSequences        = "sequences"
	objectSequenceSteps    = "sequenceSteps"
	objectSequenceStates   = "sequenceStates"
	objectCallDispositions = "callDispositions"
)

// readSupport is the complete list of objects this connector can read. Also
// exposed via SupportedReadObjects for the live read/capture shim
// (test/nooks/read), which needs to iterate every supported object.
//
//nolint:gochecknoglobals
var readSupport = []string{
	objectCalls,
	objectTasks,
	objectUsers,
	objectEmails,
	objectAccounts,
	objectMailboxes,
	objectProspects,
	objectSequences,
	objectSequenceSteps,
	objectSequenceStates,
	objectCallDispositions,
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
