package bamboohrv3

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

const (
	objectNameEmployees        = "api/v1/employees"
	objectNameApplications     = "applications"
	objectNameJobs             = "jobs"
	objectNameRequests         = "requests"
	objectNameTimesheetEntries = "timesheet_entries"
	objectNameSchedules        = "schedules"
)

// SupportedObjects is the list of objects exposed by this connector.
// nolint:gochecknoglobals
var SupportedObjects = []string{
	objectNameEmployees,
	objectNameApplications,
	objectNameJobs,
	objectNameRequests,
	objectNameTimesheetEntries,
	objectNameSchedules,
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
