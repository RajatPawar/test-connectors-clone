package connectwise

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/connectwise"
	"github.com/amp-labs/connectors/test/utils"
)

// Custom credential fields for ConnectWise PSA.
// nolint:gochecknoglobals
var (
	fieldRegion   = credscanning.Field{Name: "region", PathJSON: "metadata.region", SuffixENV: "REGION"}
	fieldClientId = credscanning.Field{Name: "clientId", PathJSON: "metadata.clientId", SuffixENV: "CLIENT_ID_HEADER"}
)

func GetConnectWiseConnector(ctx context.Context) *connectwise.Connector {
	filePath := credscanning.LoadPath(providers.ConnectWise)
	reader := utils.MustCreateProvCredJSON(filePath, false, fieldRegion, fieldClientId)

	// ConnectWise Basic auth: username = companyId+publicKey, password = privateKey.
	client := utils.NewBasicAuthClient(ctx, reader)

	conn, err := connectwise.NewConnector(
		common.ConnectorParams{
			AuthenticatedClient: client,
			Metadata: map[string]string{
				"region":   reader.Get(fieldRegion),
				"clientId": reader.Get(fieldClientId),
			},
		},
	)
	if err != nil {
		utils.Fail("error creating ConnectWise connector", "error", err)
	}

	return conn
}
