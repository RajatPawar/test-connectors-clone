// Package nooks provides a read-only connector for the Nooks Sequencing API.
//
// API Documentation: see ./docs (OpenAPI spec + scraped docs) in this repo.
// Authentication: Bearer token in the Authorization header — either a
// workspace-scoped `nooks-api-` API key or an OAuth2 access token.
// Base URL: https://partner-api.nooks.in (fixed, no template variables);
// the "/v1" version segment from the spec's server URL is applied in
// buildReadRequest rather than ProviderInfo.BaseURL (repo convention).
//
// This connector implements reads only (see providers/nooks/README.md);
// Nooks' write/action endpoints are out of scope for this round.
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

type Connector struct {
	// Basic connector
	*components.Connector

	// Require authenticated client
	common.RequireAuthenticatedClient

	// Supported operations
	components.SchemaProvider
	components.Reader
}

func NewConnector(params common.ConnectorParams) (*Connector, error) {
	// Create base connector with provider info
	return components.Initialize(providers.Nooks, params, constructor)
}

func constructor(base *components.Connector) (*Connector, error) {
	connector := &Connector{Connector: base}

	// Nooks ships a full OpenAPI spec; schemas.json (embedded in
	// providers/nooks/metadata) was pre-generated from it, so we use the
	// static schema provider rather than a live discovery/sampling call.
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
