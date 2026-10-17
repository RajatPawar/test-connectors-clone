package lightfield

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

const (
	keyFields        = "fields"
	keyRelationships = "relationships"
)

// flattenRecord lifts every `fields[key].value` and `relationships[key].values` to the top level,
// keyed exactly as Lightfield names them (`$name`, `$owner`, custom slugs).
// On a key clash the envelope (`id`, `createdAt`, ...) wins over fields, and fields win over relationships;
// metadata applies the same precedence.
// https://docs.lightfield.app/using-the-api/fields-and-relationships
func flattenRecord(node *ajson.Node) (map[string]any, error) {
	record, err := jsonquery.Convertor.ObjectToMap(node)
	if err != nil {
		return nil, err
	}

	flat := make(map[string]any, len(record))

	if relationships, ok := record[keyRelationships].(map[string]any); ok {
		for key, relationship := range relationships {
			if rel, ok := relationship.(map[string]any); ok {
				flat[key] = rel["values"]
			}
		}
	}

	if fields, ok := record[keyFields].(map[string]any); ok {
		for key, field := range fields {
			if typed, ok := field.(map[string]any); ok {
				flat[key] = typed["value"]
			}
		}
	}

	for key, value := range record {
		if key != keyFields && key != keyRelationships {
			flat[key] = value
		}
	}

	return flat, nil
}

// makeFilterFunc filters records by Since (exclusive) and Until (inclusive) on the connector side.
//
// The list endpoints document no filter on the record's update time: `$field[operator]` filters apply to
// entries of `fields`, while `updatedAt` sits on the record envelope. Lists are sorted by createdAt (newest
// first), not by updatedAt, so no page can be skipped and every page is scanned.
// `updatedAt` is documented as nullable; such records fall back to `createdAt`.
// TODO: Confirm with Lightfield whether a provider-side updated-since filter exists.
// https://docs.lightfield.app/using-the-api/list-endpoints#filtering
func makeFilterFunc(requestURL *url.URL) common.RecordsFilterFunc {
	nextPageFunc := makeNextPageFunc(requestURL)
	boundary := readhelper.NewTimeBoundary(readhelper.ExcludeSince())

	return func(params common.ReadParams, body *ajson.Node, records []*ajson.Node) ([]*ajson.Node, string, error) {
		next, err := nextPageFunc(body)
		if err != nil {
			return nil, "", err
		}

		if params.Since.IsZero() && params.Until.IsZero() {
			return records, next, nil
		}

		filtered := make([]*ajson.Node, 0, len(records))

		for _, record := range records {
			timestamp, err := recordUpdatedAt(record)
			if err != nil {
				return nil, "", err
			}

			if boundary.Contains(params, timestamp) {
				filtered = append(filtered, record)
			}
		}

		return filtered, next, nil
	}
}

func recordUpdatedAt(record *ajson.Node) (time.Time, error) {
	value, err := jsonquery.New(record).StringOptional("updatedAt")
	if err != nil {
		return time.Time{}, err
	}

	if value == nil {
		value, err = jsonquery.New(record).StringOptional("createdAt")
		if err != nil {
			return time.Time{}, err
		}
	}

	if value == nil {
		return time.Time{}, fmt.Errorf("%w: record has neither updatedAt nor createdAt", common.ErrMissingExpectedValues)
	}

	return time.Parse(time.RFC3339, *value)
}

// makeNextPageFunc advances `offset` by the number of records returned.
// The last page is the one shorter than `limit`, or the one reaching `totalCount`.
// https://docs.lightfield.app/using-the-api/list-endpoints#pagination
func makeNextPageFunc(requestURL *url.URL) common.NextPageFunc {
	return func(body *ajson.Node) (string, error) {
		records, err := jsonquery.New(body).ArrayRequired("data")
		if err != nil {
			return "", err
		}

		query := requestURL.Query()

		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil {
			return "", err
		}

		offset, err := strconv.Atoi(query.Get("offset"))
		if err != nil {
			return "", err
		}

		nextOffset := offset + len(records)

		if len(records) < limit {
			return "", nil
		}

		totalCount, err := jsonquery.New(body).IntegerOptional("totalCount")
		if err != nil {
			return "", err
		}

		if totalCount != nil && int64(nextOffset) >= *totalCount {
			return "", nil
		}

		nextURL, err := urlbuilder.FromRawURL(requestURL)
		if err != nil {
			return "", err
		}

		nextURL.WithQueryParam("offset", strconv.Itoa(nextOffset))

		return nextURL.String(), nil
	}
}
