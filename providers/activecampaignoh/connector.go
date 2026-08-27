// Package activecampaignoh provides a read-only connector for the
// ActiveCampaign v3 API.
//
// API Documentation: https://developers.activecampaign.com/reference
// Authentication: API key sent in the Api-Token header.
// Base URL: https://{workspace}.api-us1.com (per-tenant subdomain; the
// provider warns api-us1.com is not guaranteed for every account, but the
// OpenAPI spec's servers block is the authoritative source we follow here).
package activecampaignoh

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/components/operations"
	"github.com/amp-labs/connectors/internal/components/reader"
	"github.com/amp-labs/connectors/internal/components/schema"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/activecampaignoh/metadata"
)

// Connector is the ActiveCampaign connector. Read-only for this round: see
// providers/activecampaignoh/README.md.
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

	// "contacts" and "accounts" aren't in the embedded schemas.json (their
	// OpenAPI 200 responses only carry bare examples, no JSON Schema), so we
	// fall back to sampling one live record for those two objects.
	fallbackSchema := schema.NewObjectSchemaProvider(
		connector.HTTPClient().Client,
		schema.FetchModeParallel,
		operations.SingleObjectMetadataHandlers{
			BuildRequest:  connector.buildSingleObjectMetadataRequest,
			ParseResponse: connector.parseSingleObjectMetadataResponse,
			ErrorHandler:  common.InterpretError,
		},
	)

	connector.SchemaProvider = schema.NewCompositeSchemaProvider(
		schema.NewOpenAPISchemaProvider(common.ModuleRoot, metadata.Schemas),
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
