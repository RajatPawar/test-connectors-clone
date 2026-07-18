package sagehr

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// Object names — copied verbatim from the endpoint URL paths per CLAUDE.md
// naming rules (nouns, path prefixes preserved, no invented segments).
const (
	objectTeams                = "teams"
	objectEmployees            = "employees"
	objectPositions            = "positions"
	objectTerminationReasons   = "termination-reasons"
	objectTerminatedEmployees  = "terminated-employees"
	objectRecruitmentPositions = "recruitment/positions"
	objectLeavePolicies        = "leave-management/policies"
	objectLeaveRequests        = "leave-management/requests"
	objectDocumentCategories   = "documents/categories"

	// Fan-out objects: the provider has no company-wide listing for these —
	// only a per-parent endpoint (/employees/{id}/... or
	// /recruitment/positions/{id}/...). Reading them means listing the parent
	// object first, then fanning out one request per parent id. See parse.go.
	objectEmployeeCompensations = "employees/compensations"
	objectEmployeeCustomFields  = "employees/custom-fields"
	objectEmployeeLeaveBalances = "employees/leave-management/balances"
	objectRecruitmentApplicants = "recruitment/positions/applicants"
)

// Not implemented this round, with reasons:
//   - "documents": the OpenAPI spec only defines POST /documents (multipart
//     file upload). There is no GET/list endpoint for documents, so this is a
//     write-only endpoint and out of scope for a read-only connector.
//   - "leave-management/kit-days": GET requires BOTH policy_id AND employee_id
//     as required query params. There is no endpoint to enumerate valid
//     (policy, employee) pairs, so a company-wide listing would require an
//     unbounded cross-product of every employee against every policy. Left
//     unimplemented rather than guessing which pairs are valid.
//   - "recruitment/positions/{id}/applicants/{id}/actions": would require a
//     third level of fan-out (positions -> applicants -> actions), which is
//     unbounded in request count and not implemented this round.
//   - "onboarding/*", "offboarding/*", "performance/goals/*",
//     "leave-management/reports/individual-allowances", "vikarina/*": not in
//     the requested priority object list; vikarina/* in particular appears to
//     be a duplicate export surface for a named third-party payroll
//     integration, not a primary data source.

// readSupport is the complete list of objects this connector can read. It is
// also exposed via SupportedReadObjects for tooling (e.g. the live read/capture
// binary at test/sagehr/read) that needs to iterate every supported object.
//
//nolint:gochecknoglobals
var readSupport = []string{
	objectTeams,
	objectEmployees,
	objectPositions,
	objectTerminationReasons,
	objectTerminatedEmployees,
	objectRecruitmentPositions,
	objectLeavePolicies,
	objectLeaveRequests,
	objectDocumentCategories,
	objectEmployeeCompensations,
	objectEmployeeCustomFields,
	objectEmployeeLeaveBalances,
	objectRecruitmentApplicants,
}

// SupportedReadObjects returns every object name this connector can read.
func SupportedReadObjects() []string {
	return append([]string(nil), readSupport...)
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
