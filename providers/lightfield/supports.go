package lightfield

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/datautils"
)

const (
	objectAccounts      = "accounts"
	objectContacts      = "contacts"
	objectOpportunities = "opportunities"
	objectMembers       = "members"
)

// Each object is read from GET /v1/{object}.
// https://docs.lightfield.app/api
var readObjects = []string{ // nolint:gochecknoglobals
	objectAccounts,
	objectContacts,
	objectOpportunities,
	objectMembers,
}

// Objects with a GET /v1/{object}/definitions describe endpoint. Members have none.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#supported-object-types
var objectsWithDefinitions = datautils.NewSet( // nolint:gochecknoglobals
	objectAccounts,
	objectContacts,
	objectOpportunities,
)

// SupportedReadObjects lists the objects this connector reads.
func SupportedReadObjects() []string {
	return readObjects
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
