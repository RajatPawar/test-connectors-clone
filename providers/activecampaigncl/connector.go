// Package activecampaigncl provides a read-only connector for the
// ActiveCampaign v3 REST API.
//
// API Documentation: https://developers.activecampaign.com/reference/overview
// Authentication: API key sent in the Api-Token header (per-user key from
// Settings > Developer).
// Base URL: https://{workspace}.api-us1.com (per-tenant subdomain); the
// /api/3 version prefix is appended by the request builders in handlers.go,
// not baked into the catalog BaseURL — see providers/activeCampaign.go.
//
// TODO: v3 is rate limited to 5 requests/second/account (shared by REST and
// eComm GraphQL); per CLAUDE.md this should be registered in
// server/shared/limiter/defaults.go, but that file does not exist anywhere
// in this repository checkout (confirmed via `find . -iname defaults.go`) —
// it likely lives in a separate service repo not included here. Documented
// in README.md instead; register it once that file is available.
package activecampaigncl

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/components/operations"
	"github.com/amp-labs/connectors/internal/components/reader"
	"github.com/amp-labs/connectors/internal/components/schema"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/activecampaigncl/metadata"
)

// restAPIVersion is the version path segment ActiveCampaign requires after
// the per-tenant host (https://developers.activecampaign.com/reference/url).
const restAPIVersion = "api/3"

// Connector is the ActiveCampaign connector. Read-only: this round is scoped
// to reads only, see providers/activecampaigncl/README.md.
type Connector struct {
	*components.Connector
	common.RequireAuthenticatedClient

	components.SchemaProvider
	components.Reader
}

func NewConnector(params common.ConnectorParams) (*Connector, error) {
	return components.Initialize(providers.ActiveCampaign, params, constructor)
}

func constructor(base *components.Connector) (*Connector, error) {
	connector := &Connector{Connector: base}

	// contacts, accounts, and dealGroups are missing from the embedded
	// schemas.json (see metadata package doc comment), so we fall back to
	// live response sampling for whatever the static provider can't resolve.
	fallbackSchema := schema.NewObjectSchemaProvider(
		connector.HTTPClient().Client,
		schema.FetchModeParallel,
		operations.SingleObjectMetadataHandlers{
			BuildRequest:  connector.buildSingleObjectMetadataRequest,
			ParseResponse: connector.parseSingleObjectMetadataResponse,
		},
	)

	connector.SchemaProvider = schema.NewCompositeSchemaProvider(
		schema.NewOpenAPISchemaProvider(connector.ProviderContext.Module(), metadata.Schemas),
		fallbackSchema,
	)

	registry, err := components.NewEndpointRegistry(supportedOperations())
	if err != nil {
		return nil, err
	}

	connector.Reader = reader.NewHTTPReader(
		connector.HTTPClient().Client,
		registry,
		common.ModuleRoot,
		operations.ReadHandlers{
			BuildRequest:  connector.buildReadRequest,
			ParseResponse: connector.parseReadResponse,
			ErrorHandler:  common.InterpretError,
		},
	)

	return connector, nil
}
