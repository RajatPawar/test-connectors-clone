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
	// The workspace metadata field's ProviderInfo entry has a DefaultValue
	// ("subdomain"), which makes RequiresWorkspace() return false — so the
	// credentials reader will NOT auto-register the "workspace" JSON path
	// unless it is passed explicitly here. Omitting this silently resolves
	// {{.workspace}} in BaseURL to "", producing an invalid host
	// (https://.sage.hr/api) with no compile error. See CLAUDE.md.
	reader := utils.MustCreateProvCredJSON(filePath, false, credscanning.Fields.Workspace)

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
