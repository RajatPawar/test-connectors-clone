// Package ashbyselftest is a read-only, docs-only rebuild of the Ashby
// connector (see providers/ashbyselftest/README.md for the reasoning behind
// building this in parallel with the existing providers/ashby package).
//
// API Documentation: https://developers.ashbyhq.com
// Authentication: HTTP Basic — API key as username, blank password.
// Base URL: https://api.ashbyhq.com (fixed, no per-tenant variable).
package ashbyselftest

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/components/operations"
	"github.com/amp-labs/connectors/internal/components/reader"
	"github.com/amp-labs/connectors/internal/components/schema"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/ashbyselftest/metadata"
)

// Connector is a read-only connector for the Ashby API.
type Connector struct {
	*components.Connector
	common.RequireAuthenticatedClient

	components.SchemaProvider
	components.Reader
}

func NewConnector(params common.ConnectorParams) (*Connector, error) {
	// Reuses the existing providers.Ashby catalog entry (value "ashby") rather
	// than declaring a new Provider constant with the same string value — Go
	// does not allow two switch cases in connector/new.go to share a value, and
	// providers.Ashby already satisfies the required provider identity exactly.
	return components.Initialize(providers.Ashby, params, constructor)
}

func constructor(base *components.Connector) (*Connector, error) {
	connector := &Connector{Connector: base}

	connector.SchemaProvider = schema.NewOpenAPISchemaProvider(
		connector.ProviderContext.Module(), metadata.Schemas,
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
