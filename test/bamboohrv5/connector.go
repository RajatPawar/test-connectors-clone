package bamboohrv5test

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/bamboohrv5"
	testUtils "github.com/amp-labs/connectors/test/utils"
)

// fieldCompany is the BambooHR company subdomain, e.g. "ampersand" in
// https://ampersand.bamboohr.com. Named "company" (not "workspace") to match
// providers.BambooHR's declared metadata input and the real creds file, which
// stores this value at metadata.company.
var fieldCompany = credscanning.Field{ //nolint:gochecknoglobals
	Name:      "company",
	PathJSON:  "metadata.company",
	SuffixENV: "COMPANY",
}

// GetBambooHRConnector builds a live BambooHR connector from credentials on disk.
// Pass a custom httpClient (with a logging Transport) to capture HTTP interactions;
// pass nil to use the default client.
func GetBambooHRConnector(ctx context.Context, httpClient *http.Client) *bamboohrv5.Connector {
	filePath := credscanning.LoadPath(providers.BambooHR)
	reader := testUtils.MustCreateProvCredJSON(filePath, false, fieldCompany)

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

	conn, err := bamboohrv5.NewConnector(common.ConnectorParams{
		AuthenticatedClient: client,
		Metadata: map[string]string{
			"company": reader.Get(fieldCompany),
		},
	})
	if err != nil {
		testUtils.Fail("error creating BambooHR connector", "error", err)
	}

	return conn
}
