package metadata

import (
	_ "embed"

	"github.com/amp-labs/connectors/internal/objectcatalog"
)

var (
	// objectsMetadata is transcribed from the response schemas on Orb's fetch-object reference pages
	// (Orb publishes no OpenAPI spec), e.g. https://docs.withorb.com/api-reference/customer/fetch-customer.
	// Only top-level fields are listed; nested objects and arrays are typed as other.
	//
	//go:embed objectsMetadata.json
	objectsMetadata []byte

	Catalog = objectcatalog.MustLoad(objectsMetadata) // nolint:gochecknoglobals
)
