package tipaltitest

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/tipalti"
	"github.com/amp-labs/connectors/test/utils"
	"golang.org/x/oauth2"
)

func GetConnector(ctx context.Context) *tipalti.Connector {
	filePath := credscanning.LoadPath(providers.Tipalti)
	reader := utils.MustCreateProvCredJSON(filePath, true)

	client := utils.NewOauth2Client(ctx, reader, oauthConfig)

	conn, err := tipalti.NewConnector(common.ConnectorParams{
		AuthenticatedClient: client,
	})
	if err != nil {
		utils.Fail("error creating Tipalti connector", "error", err)
	}

	return conn
}

func oauthConfig(reader *credscanning.ProviderCredentials) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     reader.Get(credscanning.Fields.ClientId),
		ClientSecret: reader.Get(credscanning.Fields.ClientSecret),
		RedirectURL:  "https://dev-api.withampersand.com/callbacks/v1/oauth",
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://sso.tipalti.com/connect/authorize",
			TokenURL:  "https://sso.tipalti.com/connect/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}
