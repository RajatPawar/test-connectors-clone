package nooks

import (
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// responseDataField is the envelope field holding the record array. Every
// Nooks list endpoint uses the same shape: {"data": [...], "links": {...}}
// (see schemas.json responseKey and every list example in openapi_spec.json).
const responseDataField = "data"

// updatedAtTimeFormat matches the `updatedAt` timestamps Nooks returns, e.g.
// "2026-03-18T10:00:00.000Z" (RFC3339 with milliseconds).
const updatedAtTimeFormat = time.RFC3339Nano

// records extracts the "data" array common to every Nooks list response.
func records() common.RecordsFunc {
	return common.ExtractRecordsFromPath(responseDataField)
}

// nextRecordsURL extracts the opaque next-page URL from the "links.next" field.
// It is null once the last page has been reached (see PaginationLinks in
// openapi_spec.json).
func nextRecordsURL() common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		return jsonquery.New(node, "links").StrWithDefault("next", "")
	}
}
