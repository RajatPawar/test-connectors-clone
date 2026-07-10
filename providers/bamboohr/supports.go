package bamboohr

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

const (
	objectNameEmployees         = "employees"
	objectNameEmployeeDirectory = "employees/directory"
	objectNameCustomReports     = "custom-reports"
	objectNameTimeOffRequests   = "time_off/requests"
)

// standardTableNames are the standard employee-table aliases documented in the TableName
// schema (GET /api/v1/employees/{id}/tables/{table}). Custom, company-defined tables
// (e.g. "custom1") also exist, but their names can only be discovered per-company via
// GET /api/v1/meta/tables (list-tabular-fields) — a discovery endpoint, not a static list.
// TODO: consider dynamically registering custom tables via a discovery step.
//
//nolint:gochecknoglobals
var standardTableNames = []string{
	"jobInfo", "jobInformation", "compensation", "employmentStatus", "contacts",
	"emergencyContacts", "dependents", "earnings", "bonus", "commission", "benefit_class",
	"employeeVisas", "employeeEducation", "employeePassports", "employeeDriverLicenses",
	"employeeCertifications", "employeeStockOptions", "employeeAssets", "employeeCreditCards",
	"employeeCovidTests", "employeeCovidVaccinations", "employeeCovidVaccinationExemptions",
	"employeeCovidExposures", "employeeEquityGrants", "levelsAndBands", "employeeProjectPayRates",
}

func isTableObject(objectName string) bool {
	for _, name := range standardTableNames {
		if name == objectName {
			return true
		}
	}

	return false
}

func supportedOperations() components.EndpointRegistryInput {
	readSupport := append([]string{
		objectNameEmployees,
		objectNameEmployeeDirectory,
		objectNameCustomReports,
		objectNameTimeOffRequests,
	}, standardTableNames...)

	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
