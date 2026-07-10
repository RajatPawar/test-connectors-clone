package bamboohr

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/bamboohr"
	"github.com/amp-labs/connectors/test/utils"
)

// GetBambooHRConnector builds a connector using credentials from the standard credscanning
// JSON file. Pass a non-nil onInteraction to observe every HTTP request/response the connector
// makes (used by the -out capture mode in test/bamboohr/read).
func GetBambooHRConnector(ctx context.Context, onInteraction InteractionRecorder) *bamboohr.Connector {
	filePath := credscanning.LoadPath(providers.BambooHR)
	reader := utils.MustCreateProvCredJSON(filePath, false)

	httpClient := &http.Client{
		Transport: &capturingTransport{
			base:          http.DefaultTransport,
			onInteraction: onInteraction,
		},
	}

	client, err := common.NewBasicAuthHTTPClient(
		ctx,
		reader.Get(credscanning.Fields.Username),
		reader.Get(credscanning.Fields.Password),
		common.WithHeaderClient(httpClient),
	)
	if err != nil {
		utils.Fail("error creating bamboohr basic auth client", "error", err)
	}

	conn, err := bamboohr.NewConnector(common.ConnectorParams{
		AuthenticatedClient: client,
		Workspace:           reader.Get(credscanning.Fields.Workspace),
	})
	if err != nil {
		utils.Fail("error creating bamboohr connector", "error", err)
	}

	return conn
}
