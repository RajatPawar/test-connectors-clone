package bamboographtrial5

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/bamboographtrial5"
	"github.com/amp-labs/connectors/test/utils"
)

// fieldCompanyDomain reads the BambooHR company subdomain from the creds file.
// In the JSON creds file this lives at metadata.companyDomain; as an env var use
// BAMBOOHR_COMPANY_DOMAIN (prefixed by credscanning's provider-name convention).
var fieldCompanyDomain = credscanning.Field{
	Name:      "companyDomain",
	PathJSON:  "metadata.companyDomain",
	SuffixENV: "COMPANY_DOMAIN",
}

// GetConnector constructs a live BambooHR connector from the creds file.
// The creds file must contain:
//   - username: BambooHR API key (used as Basic auth username)
//   - password: any value (BambooHR ignores the password; "x" is conventional)
//   - metadata.companyDomain: the BambooHR company subdomain
func GetConnector(ctx context.Context) *bamboographtrial5.Connector {
	filePath := credscanning.LoadPath(providers.BambooGraphTrial5)
	reader := utils.MustCreateProvCredJSON(filePath, false, fieldCompanyDomain)

	conn, err := bamboographtrial5.NewConnector(common.ConnectorParams{
		AuthenticatedClient: utils.NewBasicAuthClient(ctx, reader),
		Metadata: map[string]string{
			"companyDomain": reader.Get(fieldCompanyDomain),
		},
	})
	if err != nil {
		utils.Fail("error creating BambooHR connector", "error", err)
	}

	return conn
}
