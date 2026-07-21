package sagehr

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/sagehr"
	"github.com/amp-labs/connectors/test/utils"
)

func GetConnector(ctx context.Context) *sagehr.Connector {
	return getConnector(ctx, nil)
}

// GetConnectorWithClientWrapper builds the connector the same way as
// GetConnector, but first passes the underlying authenticated HTTP client
// through wrap. Used by the capture-mode read binary (test/sagehr/read) to
// record the full request/response of every call the connector makes.
func GetConnectorWithClientWrapper(
	ctx context.Context, wrap func(common.AuthenticatedHTTPClient) common.AuthenticatedHTTPClient,
) *sagehr.Connector {
	return getConnector(ctx, wrap)
}

func getConnector(
	ctx context.Context, wrap func(common.AuthenticatedHTTPClient) common.AuthenticatedHTTPClient,
) *sagehr.Connector {
	filePath := credscanning.LoadPath(providers.SageHR)
	reader := utils.MustCreateProvCredJSON(filePath, false)

	client := utils.NewAPIKeyClient(ctx, reader, providers.SageHR)
	if wrap != nil {
		client = wrap(client)
	}

	conn, err := sagehr.NewConnector(common.ConnectorParams{
		AuthenticatedClient: client,
		Workspace:           reader.Get(credscanning.Fields.Workspace),
	})
	if err != nil {
		utils.Fail("error creating connector", "error", err)
	}

	return conn
}
