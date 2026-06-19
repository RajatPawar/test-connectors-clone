package bamboohr

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/bamboohr"
	"github.com/amp-labs/connectors/test/utils"
)

// GetBambooHRConnector creates a BambooHR connector from credentials on disk.
// The credentials file must provide:
//   - username: BambooHR API key
//   - password: empty string (BambooHR uses blank password with API key)
//   - workspace: company domain (e.g. "mycompany" for mycompany.bamboohr.com)
func GetBambooHRConnector(ctx context.Context) *bamboohr.Connector {
	filePath := credscanning.LoadPath(providers.BambooHR)
	reader := utils.MustCreateProvCredJSON(filePath, false)

	conn, err := bamboohr.NewConnector(
		common.ConnectorParams{
			AuthenticatedClient: utils.NewBasicAuthClient(ctx, reader),
			Workspace:           reader.Get(credscanning.Fields.Workspace),
		},
	)
	if err != nil {
		utils.Fail("error creating BambooHR connector", "error", err)
	}

	return conn
}
