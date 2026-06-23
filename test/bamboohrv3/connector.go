package bamboohrv3test

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/bamboohrv3"
	testUtils "github.com/amp-labs/connectors/test/utils"
)

// GetBambooHRConnector builds a live BambooHR connector from credentials on disk.
// Pass a custom httpClient (with a logging Transport) to capture HTTP interactions;
// pass nil to use http.DefaultClient.
func GetBambooHRConnector(ctx context.Context, httpClient *http.Client) *bamboohrv3.Connector {
	filePath := credscanning.LoadPath(providers.BambooHR)
	reader := testUtils.MustCreateProvCredJSON(filePath, false, credscanning.Fields.Workspace)

	opts := []common.HeaderAuthClientOption{}
	if httpClient != nil {
		opts = append(opts, common.WithHeaderClient(httpClient))
	}

	client, err := common.NewBasicAuthHTTPClient(
		ctx,
		reader.Get(credscanning.Fields.Username),
		reader.Get(credscanning.Fields.Password),
		opts...,
	)
	if err != nil {
		testUtils.Fail("error creating basic auth client", "error", err)
	}

	conn, err := bamboohrv3.NewConnector(common.ConnectorParams{
		AuthenticatedClient: client,
		Workspace:           reader.Get(credscanning.Fields.Workspace),
	})
	if err != nil {
		testUtils.Fail("error creating BambooHR connector", "error", err)
	}

	return conn
}
