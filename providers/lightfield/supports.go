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
	objectNameMembers       = "members"
)

// Members are the workspace users; they have no definitions endpoint, so their schema is the static catalog.
var supportedReadObjects = []string{ // nolint:gochecknoglobals
	objectNameAccounts,
	objectNameContacts,
	objectNameOpportunities,
	objectNameMembers,
}

// Objects described by GET /v1/{object}/definitions.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#supported-object-types
var objectsWithDefinitions = datautils.NewSet( // nolint:gochecknoglobals
	objectNameAccounts,
	objectNameContacts,
	objectNameOpportunities,
)

// SupportedReadObjects lists the objects the connector reads.
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
