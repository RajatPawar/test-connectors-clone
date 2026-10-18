package orb

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers/orb/internal/metadata"
)

// ListObjectMetadata serves the static catalog. Orb has no describe endpoint, and its records
// have no user-defined fields beyond the free-form `metadata` map, which is listed as one field.
func (c *Connector) ListObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	return metadata.Catalog.SelectResult(objectNames)
}
