package otterai

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

const (
	objectNameConversations   = "conversations"
	objectNameChannels        = "channels"
	objectNameChannelsMembers = "channels/members"
	objectNameWorkspace       = "workspace"
)

// readSupport lists every object this connector exposes. Otter's Public API also
// documents POST /conversations (create a conversation from a file) and
// GET /conversations/{id}/audio (an MP3 download link, not a queryable data
// object) — both are out of scope for a read-only, data-model connector.
//
// TODO: GET /conversations/{id}?include=action_items,insights,outline,transcript
// exposes per-conversation relationship data (action items, insights, outline,
// transcript, custom_prompt) that is never fetched here — buildReadRequest only
// calls the list endpoint GET /conversations. There is no bulk/list-all form of
// these relationships (each requires fetching one conversation detail at a
// time), so surfacing them as first-class objects would need an N+1 fan-out
// over every conversation, similar to (but heavier than) the channels/members
// fan-out below. Deferred this round to stay within the PM plan's 4 target
// objects; flagged for human review on whether action_items/insights/outline
// should become their own read objects in a follow-up round.
// See README.md.
func readSupport() []string {
	return []string{
		objectNameConversations,
		objectNameChannels,
		objectNameChannelsMembers,
		objectNameWorkspace,
	}
}

// SupportedReadObjects returns every object name this connector can read.
// Used by the live-capture harness shim (test/otter.ai/read/main.go).
func SupportedReadObjects() []string {
	return readSupport()
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport(), ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
