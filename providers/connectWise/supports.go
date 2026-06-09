package connectWise

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

// Object names are derived directly from the ConnectWise URL paths
// ({service}/{object}) and preserve their service prefix. Every object below
// supports a GET list and a POST create; updates use PATCH on /{id}.
//
// TODO: ConnectWise exposes ~3,100 endpoints. This first version covers the
// core, persistent PSA entities. Additional objects can be added to these
// slices once confirmed against the docs.
func supportedOperations() components.EndpointRegistryInput {
	readSupport := []string{
		"company/companies",
		"company/contacts",
		"company/configurations",
		"service/tickets",
		"sales/opportunities",
		"sales/activities",
		"project/projects",
		"finance/agreements",
		"procurement/products",
		"time/entries",
		"schedule/entries",
	}

	writeSupport := []string{
		"company/companies",
		"company/contacts",
		"company/configurations",
		"service/tickets",
		"sales/opportunities",
		"sales/activities",
		"project/projects",
		"finance/agreements",
		"procurement/products",
		"time/entries",
		"schedule/entries",
	}

	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(writeSupport, ",")),
				Support:  components.WriteSupport,
			},
		},
	}
}
