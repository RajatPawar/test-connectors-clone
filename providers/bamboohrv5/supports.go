package bamboohrv5

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// SupportedObjects is the list of objects exposed for reading by this connector.
var SupportedObjects = []string{ // nolint:gochecknoglobals
	objectNameEmployees,
	objectNameJobs,
	objectNameApplications,
	objectNameRequests,
	objectNameSchedules,
	objectNameTimesheetEntries,
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(SupportedObjects, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
