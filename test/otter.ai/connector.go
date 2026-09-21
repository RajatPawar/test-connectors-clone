package otterai

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	otterai "github.com/amp-labs/connectors/providers/otter.ai"
	"github.com/amp-labs/connectors/test/utils"
)

func GetOtterAIConnector(ctx context.Context) *otterai.Connector {
	filePath := credscanning.LoadPath(providers.OtterAI)
	reader := utils.MustCreateProvCredJSON(filePath, false)

	client := utils.NewAPIKeyClient(ctx, reader, providers.OtterAI)

	conn, err := otterai.NewConnector(common.ConnectorParams{
		AuthenticatedClient: client,
	})
	if err != nil {
		utils.Fail("error creating Otter.ai connector", "error", err)
	}

	return conn
}
