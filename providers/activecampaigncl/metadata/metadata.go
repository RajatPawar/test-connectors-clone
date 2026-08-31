// Package metadata embeds the static, OpenAPI-derived field schema for the
// ActiveCampaign v3 REST API (copied verbatim from ./docs/schemas.json — see
// CLAUDE.md's metadata priority rules). It covers 5 of our 8 read objects
// (deals, campaigns, users, lists, tags); contacts, accounts, and dealGroups
// are missing from it because their OpenAPI response definitions only
// declare a free-text `examples` block with no JSON Schema for the generator
// to extract fields from. Those three objects fall back to live response
// sampling — see providers/activecampaigncl/connector.go.
package metadata

import (
	_ "embed"

	"github.com/amp-labs/connectors/internal/staticschema"
	"github.com/amp-labs/connectors/tools/scrapper"
)

//nolint:gochecknoglobals
var (
	//go:embed schemas.json
	schemas []byte

	FileManager = scrapper.NewReader[staticschema.FieldMetadataMapV2](schemas)
	Schemas     = FileManager.MustLoadSchemas()
)
