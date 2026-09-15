// Package metadata embeds the static object schema for the Nooks connector,
// generated from Nooks' OpenAPI spec (./docs/schemas.json in the build environment).
package metadata

import (
	_ "embed"

	"github.com/amp-labs/connectors/internal/staticschema"
	"github.com/amp-labs/connectors/tools/scrapper"
)

// nolint:gochecknoglobals
var (
	//go:embed schemas.json
	schemas []byte

	FileManager = scrapper.NewReader[staticschema.FieldMetadataMapV2](schemas)
	Schemas     = FileManager.MustLoadSchemas()
)
