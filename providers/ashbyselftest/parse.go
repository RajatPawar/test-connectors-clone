package ashbyselftest

import (
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// timestampQuery locates the field this connector treats as a record's
// "last updated" time, for objects where the field isn't a plain top-level
// updatedAt.
type timestampQuery struct {
	zoom  []string
	field string
}

// updatedAtQuery maps each object to where its "last updated" timestamp
// lives. offer.list and opening.list responses have no updatedAt field at
// all (confirmed against the documented response shape) — but every edit to
// an offer or opening creates a new `latestVersion`, so latestVersion's own
// createdAt is the best available proxy for "last modified". This is a
// documented gap, not a guess: see connie_notes.md.
//
//nolint:gochecknoglobals
var updatedAtQuery = datautils.NewDefaultMap(
	datautils.Map[string, timestampQuery]{
		objectOffer:   {zoom: []string{"latestVersion"}, field: "createdAt"},
		objectOpening: {zoom: []string{"latestVersion"}, field: "createdAt"},
	},
	func(_ string) timestampQuery { return timestampQuery{field: "updatedAt"} },
)

// filterByUpdatedAt returns a RecordsFilterFunc that applies ReadParams'
// Since/Until window against each record's own updated-time field, and
// determines the next page token appropriately for the given object.
//
// Ashby list endpoints expose no updated_at/updated_since query parameter
// (see handlers.go), so filtering happens here, client-side, against data
// already fetched — matching the CLAUDE.md fallback for endpoints lacking a
// direct time-scoping request parameter. A record whose timestamp is absent
// or unparsable is kept rather than silently dropped: Ashby's own docs are
// inconsistent about which timestamp fields are "required" per object, so
// treating a missing field as fatal would risk failing an otherwise-healthy
// sync over a single sparse record.
func filterByUpdatedAt(objectName string) common.RecordsFilterFunc {
	query := updatedAtQuery.Get(objectName)

	return func(params common.ReadParams, body *ajson.Node, records []*ajson.Node) ([]*ajson.Node, string, error) {
		next, err := nextPageToken(objectName, body)
		if err != nil {
			return nil, "", err
		}

		if params.Since.IsZero() && params.Until.IsZero() {
			return records, next, nil
		}

		filtered := make([]*ajson.Node, 0, len(records))

		for _, record := range records {
			timestamp, ok := extractTimestamp(record, query)
			if !ok {
				filtered = append(filtered, record)

				continue
			}

			if !params.Since.IsZero() && timestamp.Before(params.Since) {
				continue
			}

			if !params.Until.IsZero() && timestamp.After(params.Until) {
				continue
			}

			filtered = append(filtered, record)
		}

		return filtered, next, nil
	}
}

// extractTimestamp reads and parses the record's update timestamp. It
// returns ok=false whenever the field is missing, null, or not a valid
// RFC3339 string, so the caller can fail open instead of erroring.
func extractTimestamp(record *ajson.Node, query timestampQuery) (time.Time, bool) {
	raw, err := jsonquery.New(record, query.zoom...).StringOptional(query.field)
	if err != nil || raw == nil {
		return time.Time{}, false
	}

	parsed, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}

// nextPageToken resolves the next page cursor for an object. jobPosting.list
// has no cursor/moreDataAvailable fields at all — it always returns every
// posting in one response — so it never has a next page.
func nextPageToken(objectName string, body *ajson.Node) (string, error) {
	if !supportsCursorPagination.Has(objectName) {
		return "", nil
	}

	moreDataAvailable, err := jsonquery.New(body).BoolWithDefault("moreDataAvailable", false)
	if err != nil {
		return "", err //nolint:wrapcheck
	}

	if !moreDataAvailable {
		return "", nil
	}

	cursor, err := jsonquery.New(body).StringOptional("nextCursor")
	if err != nil || cursor == nil {
		return "", nil //nolint:nilerr
	}

	return *cursor, nil
}
