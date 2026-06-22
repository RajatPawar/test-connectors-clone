package bamboographtrial5

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// readObjects lists the object names exposed by this connector.
// Keys match the top-level object keys in metadata/schemas.json.
var readObjects = []string{
	"jobs",                    // /api/v1/applicant_tracking/jobs
	"api/v1/hris/org/locations", // /api/v1/hris/org/locations
	"schedules",               // /api/v1/scheduling/schedules
	"policies",                // /api/v1/meta/time_off/policies
	"benefitcoverages",        // /api/v1/benefitcoverages
	"whos_out",                // /api/v1/time_off/whos_out
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
