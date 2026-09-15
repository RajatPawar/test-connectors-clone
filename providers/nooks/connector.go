// Package nooks provides a read-only connector for the Nooks Sequencing API.
//
// API Documentation: Nooks' OpenAPI spec (servers[0].url = https://partner-api.nooks.in/v1).
// Authentication: Bearer token (nooks-api-... API key or OAuth2 access token) in the
// Authorization header.
package nooks

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/components/operations"
	"github.com/amp-labs/connectors/internal/components/reader"
	"github.com/amp-labs/connectors/internal/components/schema"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/nooks/metadata"
)

// Connector is the Nooks connector. Read-only: writes (tasks, sequences,
// sequenceStates, notes, prospect CRM sync) are out of scope for this round —
// see providers/nooks/README.md.
type Connector struct {
	*components.Connector
	common.RequireAuthenticatedClient

	components.SchemaProvider
	components.Reader
}

func NewConnector(params common.ConnectorParams) (*Connector, error) {
	return components.Initialize(providers.Nooks, params, constructor)
}

func constructor(base *components.Connector) (*Connector, error) {
	connector := &Connector{Connector: base}

	// Field metadata comes from the pre-generated schemas.json (OpenAPI-derived),
	// per CLAUDE.md's Priority 1 metadata source. Nooks has no live describe endpoint.
	connector.SchemaProvider = schema.NewOpenAPISchemaProvider(connector.ProviderContext.Module(), metadata.Schemas)

	registry, err := components.NewEndpointRegistry(supportedOperations())
	if err != nil {
		return nil, err
	}

	connector.Reader = reader.NewHTTPReader(
		connector.HTTPClient().Client,
		registry,
		connector.ProviderContext.Module(),
		operations.ReadHandlers{
			BuildRequest:  connector.buildReadRequest,
			ParseResponse: connector.parseReadResponse,
			ErrorHandler:  common.InterpretError,
		},
	)

	return connector, nil
}
