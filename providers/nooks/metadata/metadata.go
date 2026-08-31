package metadata

import (
	_ "embed"

	"github.com/amp-labs/connectors/internal/staticschema"
	"github.com/amp-labs/connectors/tools/scrapper"
)

// Static object/field schema generated from the Nooks OpenAPI spec.
// See ./schemas.json and providers/nooks/README.md.
//
//nolint:gochecknoglobals
var (
	//go:embed schemas.json
	schemas []byte

	FileManager = scrapper.NewReader[staticschema.FieldMetadataMapV2](schemas)

	Schemas = FileManager.MustLoadSchemas()
)
