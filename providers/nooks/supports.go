package nooks

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/datautils"
)

// Object names — copied verbatim from the endpoint URL paths (e.g. GET /calls,
// GET /sequenceSteps), per CLAUDE.md naming rules. Nooks' own path segments are
// already camelCase nouns, so no case normalization is applied.
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

// Not implemented this round, with reasons:
//   - "emailTemplate": the OpenAPI spec only defines GET /emailTemplate/{id}
//     (read-by-id). There is no GET /emailTemplate list endpoint, so this is
//     not a listable/pollable object — per CLAUDE.md rule 6 (no synthetic
//     objects) and the requirement that objects be persistent, *queryable*
//     entities. Template ids can be discovered via
//     GET /sequenceSteps/{id}?include=template, but that's a per-parent
//     lookup, not a company-wide listing.
//   - "notes": /accounts/{id}/notes and /prospects/{id}/notes are POST-only
//     (create a CRM note) with no corresponding GET/list endpoint anywhere in
//     the spec. Not readable, and write is out of scope this round anyway.
//   - "/integrations/prospects/sync", "/tasks/{id}/skip",
//     "/tasks/{id}/complete", "/sequenceStates/{id}/actions/finish": action
//     endpoints (verbs), not objects, per naming rule 7 — skipped.
//   - "search:read" scope is documented in the OAuth scopes table but no
//     corresponding /search endpoint exists anywhere in the spec paths, so
//     there is nothing to wire up.

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
// Used by the live-read/capture shim at test/nooks/read.
func SupportedReadObjects() []string {
	return append([]string(nil), readSupport...)
}

// objectsWithoutUpdatedAtFilter lists objects for which the OpenAPI spec does
// not expose any filter[updatedAt] query parameter at all (checked against
// every parameter listed for each endpoint in openapi_spec.json):
//   - tasks: only filter[dueAt][gte]/[lte] (due date, not update time)
//   - callDispositions: no time-scoping parameter of any kind
//
// Both objects still return an `updatedAt` field on every record, so
// Since/Until incremental filtering is applied connector-side instead — see
// parse.go / handlers.go.
//
//nolint:gochecknoglobals
var objectsWithoutUpdatedAtFilter = datautils.NewSet(objectTasks, objectCallDispositions)

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
