package coda

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/coda"
	"github.com/amp-labs/connectors/test/utils"
)

func GetCodaConnector(ctx context.Context) *coda.Connector {
	filePath := credscanning.LoadPath(providers.Coda)
	reader := utils.MustCreateProvCredJSON(filePath, false)

	client := utils.NewAPIKeyClient(ctx, reader, providers.Coda)

	conn, err := coda.NewConnector(common.ConnectorParams{
		AuthenticatedClient: client,
	})
	if err != nil {
		utils.Fail("error creating coda connector", "error", err)
	}

	return conn
}
