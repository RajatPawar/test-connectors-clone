package lightfield

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/datautils"
)

const (
	objectNameAccounts      = "accounts"
	objectNameContacts      = "contacts"
	objectNameOpportunities = "opportunities"
	// Workspace members, the CRM's users. https://docs.lightfield.app/api/resources/member/methods/list
	objectNameMembers = "members"
)

var supportedReadObjects = []string{ // nolint:gochecknoglobals
	objectNameAccounts,
	objectNameContacts,
	objectNameOpportunities,
	objectNameMembers,
}

// objectsWithDefinitions describe their standard and custom fields via GET /v1/{object}/definitions.
// Members have no definitions endpoint.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#supported-object-types
var objectsWithDefinitions = datautils.NewSet( // nolint:gochecknoglobals
	objectNameAccounts,
	objectNameContacts,
	objectNameOpportunities,
)

// SupportedReadObjects lists the objects this connector reads.
func SupportedReadObjects() []string {
	return supportedReadObjects
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(supportedReadObjects, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
