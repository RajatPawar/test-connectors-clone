package metadata

import (
	_ "embed"

	"github.com/amp-labs/connectors/internal/staticschema"
	"github.com/amp-labs/connectors/tools/scrapper"
)

//go:embed schemas.json
var schemas []byte

var (
	FileManager = scrapper.NewReader[staticschema.FieldMetadataMapV2](schemas) // nolint:gochecknoglobals
	Schemas     = FileManager.MustLoadSchemas()                                 // nolint:gochecknoglobals
)
