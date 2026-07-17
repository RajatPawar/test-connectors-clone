package sagehr

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/sagehr"
	"github.com/amp-labs/connectors/test/utils"
)

// GetSageHRConnector builds a connector using the default HTTP transport.
func GetSageHRConnector(ctx context.Context) *sagehr.Connector {
	return getConnector(ctx, http.DefaultTransport)
}

// GetSageHRConnectorWithTransport builds a connector whose HTTP client uses the
// given transport, so callers can wrap it (e.g. to log/capture requests and
// responses) without duplicating credential-loading logic.
func GetSageHRConnectorWithTransport(ctx context.Context, transport http.RoundTripper) *sagehr.Connector {
	return getConnector(ctx, transport)
}

func getConnector(ctx context.Context, transport http.RoundTripper) *sagehr.Connector {
	filePath := credscanning.LoadPath(providers.SageHR)
	reader := utils.MustCreateProvCredJSON(filePath, false)

	client, err := common.NewApiKeyHeaderAuthHTTPClient(ctx, "X-Auth-Token", reader.Get(credscanning.Fields.ApiKey),
		common.WithHeaderClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		utils.Fail("error creating client", "error", err)
	}

	conn, err := sagehr.NewConnector(
		common.ConnectorParams{
			AuthenticatedClient: client,
			Workspace:           reader.Get(credscanning.Fields.Workspace),
		},
	)
	if err != nil {
		utils.Fail("error creating connector", "error", err)
	}

	return conn
}
