package nooks

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/providers/nooks/metadata"
)

// SupportedReadObjects returns every object name this connector can read.
// Used by the live read/capture shim at test/nooks/read.
func SupportedReadObjects() []string {
	return metadata.Schemas.ObjectNames().GetList(common.ModuleRoot)
}

// supportedOperations declares the read-only endpoint registry. Every object
// in schemas.json is a top-level GET list endpoint from the Nooks OpenAPI
// spec (calls, tasks, users, emails, accounts, mailboxes, prospects,
// sequences, sequenceSteps, sequenceStates, callDispositions).
//
// Not included: emailTemplate — only exposed as GET /emailTemplate/{id}
// (singular, no list endpoint), so it cannot be a top-level read object per
// CLAUDE.md's "no synthetic objects" rule.
func supportedOperations() components.EndpointRegistryInput {
	readSupport := SupportedReadObjects()

	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
