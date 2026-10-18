package metadata

import (
	_ "embed"

	"github.com/amp-labs/connectors/internal/objectcatalog"
)

var (
	// objectsMetadata is written by hand: Orb publishes no OpenAPI spec we can generate from.
	// Fields and types follow the response examples of each list endpoint in the API reference
	// (https://docs.withorb.com/api-reference), and select values the enums the reference lists.
	//
	//go:embed objectsMetadata.json
	objectsMetadata []byte

	Catalog = objectcatalog.MustLoad(objectsMetadata) // nolint:gochecknoglobals
)
