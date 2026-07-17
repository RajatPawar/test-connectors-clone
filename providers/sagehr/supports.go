package sagehr

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// supportedOperations lists every Sage HR object exposed by this connector.
// Object names are truncations of their list-endpoint URL path, e.g.
// GET /leave-management/requests -> "leave-management/requests".
//
// The following documented GET endpoints are intentionally NOT exposed:
//   - /employees/{id}, /terminated-employees/{id}, /recruitment/positions/{id},
//     /recruitment/applicants/{id}, /leave-management/policies/{id}: single-record
//     "detail" variants of objects already covered by their list endpoint.
//   - /employees/{id}/compensations, /employees/{id}/custom-fields,
//     /employees/{id}/leave-management/balances, /recruitment/positions/{id}/applicants,
//     /recruitment/applicants/{id}/actions, /leave-management/kit-days: require a
//     parent record id as part of the path, so they cannot be listed generically.
//   - /leave-management/reports/individual-allowances,
//     /performance/goals/quarterly-progress/*: report/aggregate endpoints, not
//     persistent queryable entities.
//
// TODO: revisit whether the parent-id-scoped endpoints above should be exposed
// once the framework has a pattern for objects that require a parent record id.
func supportedOperations() components.EndpointRegistryInput {
	readSupport := []string{
		"teams",
		"employees",
		"positions",
		"termination-reasons",
		"documents/categories",
		"terminated-employees",
		"onboarding/categories",
		"recruitment/positions",
		"offboarding/categories",
		"leave-management/policies",
		"leave-management/requests",
		"leave-management/out-of-office-today",
	}

	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
