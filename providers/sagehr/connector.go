// Package sagehr provides a read-only connector for the Sage HR API.
//
// API Documentation: https://developer.sage.com/hr/apis/sagehr/v1.0.0/sage-hr-v1-0-swagger
// Authentication: API key sent in the X-Auth-Token header.
// Base URL: https://{workspace}.sage.hr/api (per-tenant subdomain).
package sagehr

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/components/operations"
	"github.com/amp-labs/connectors/internal/components/reader"
	"github.com/amp-labs/connectors/internal/components/schema"
	"github.com/amp-labs/connectors/providers"
)

// Connector is the Sage HR connector. Read-only: Sage HR's write endpoints
// are out of scope for this connector (see providers/sagehr/README.md).
type Connector struct {
	*components.Connector
	common.RequireAuthenticatedClient

	components.SchemaProvider
	components.Reader
}

func NewConnector(params common.ConnectorParams) (*Connector, error) {
	return components.Initialize(providers.SageHR, params, constructor)
}

func constructor(base *components.Connector) (*Connector, error) {
	connector := &Connector{Connector: base}

	// Sage HR has no metadata/describe endpoint and no pre-built schemas.json,
	// so field metadata is sampled live from each object's own read endpoint
	// (Priority 3 in CLAUDE.md: response sampling).
	connector.SchemaProvider = schema.NewObjectSchemaProvider(
		connector.HTTPClient().Client,
		schema.FetchModeParallel,
		operations.SingleObjectMetadataHandlers{
			BuildRequest:  connector.buildSingleObjectMetadataRequest,
			ParseResponse: connector.parseSingleObjectMetadataResponse,
		},
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
