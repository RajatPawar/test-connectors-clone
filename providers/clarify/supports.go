package clarify

import (
	"fmt"
	"slices"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/datautils"
)

const (
	objectNamePerson  = "person"
	objectNameCompany = "company"
	objectNameDeal    = "deal"
	objectNameMeeting = "meeting"
	objectNameTask    = "task"
	// Workspace members, GET /v1/workspaces/{workspace}/users.
	objectNameUsers = "users"
)

var supportedObjects = []string{ // nolint:gochecknoglobals
	objectNamePerson,
	objectNameCompany,
	objectNameDeal,
	objectNameMeeting,
	objectNameTask,
	objectNameUsers,
}

// Built-in objects read through GET /v1/workspaces/{workspace}/objects/{object}/resources.
// Their object name is the {object} path segment, and GET /schemas describes them.
// https://developer.clarify.ai/docs/getting-started/key-concepts#objects
var resourceObjects = datautils.NewSet( // nolint:gochecknoglobals
	objectNamePerson,
	objectNameCompany,
	objectNameDeal,
	objectNameMeeting,
	objectNameTask,
)

// SupportedReadObjects lists the objects this connector reads.
func SupportedReadObjects() []string {
	return slices.Clone(supportedObjects)
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(supportedObjects, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
