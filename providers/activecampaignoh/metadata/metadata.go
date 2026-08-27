// Package metadata embeds the pre-generated OpenAPI-derived schema for the
// ActiveCampaign v3 API (source: docs/schemas.json). It covers most of the
// objects this connector exposes; "contacts" and "accounts" are absent from
// the spec's response schema (their 200 responses use bare examples, not a
// JSON Schema) and fall back to live response sampling — see connector.go.
package metadata

import (
	_ "embed"

	"github.com/amp-labs/connectors/internal/staticschema"
	"github.com/amp-labs/connectors/tools/fileconv"
	"github.com/amp-labs/connectors/tools/scrapper"
)

// nolint:gochecknoglobals
var (
	//go:embed schemas.json
	schemaContent []byte

	FileManager = scrapper.NewMetadataFileManager[staticschema.FieldMetadataMapV2](
		schemaContent,
		fileconv.NewSiblingFileLocator(),
	)

	Schemas = FileManager.MustLoadSchemas()
)
