package files

import (
	_ "embed"

	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
)

var (
	// Static file containing openapi spec.
	// Source: https://storage.googleapis.com/stainless-sdk-openapi-specs/lightfield/lightfield-17ee537d0758c4a85d993301a200bff23a0d407be2e1ceaf2603746cdd51cfa7.yml
	//
	//go:embed openapi.json
	apiFile []byte

	OpenApiFile = api.NewFile(apiFile)                     // nolint:gochecknoglobals
	OutputDir   = "providers/lightfield/internal/metadata" // nolint:gochecknoglobals
)
