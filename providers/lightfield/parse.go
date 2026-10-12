package lightfield

import (
	"net/http"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

const (
	keyFields        = "fields"
	keyRelationships = "relationships"
)

// makeNextPageFunc returns the next offset. A page shorter than the requested limit is the last one.
// https://docs.lightfield.app/using-the-api/list-endpoints#pagination
func makeNextPageFunc(request *http.Request) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		records, err := jsonquery.New(node).ArrayRequired("data")
		if err != nil {
			return "", err
		}

		query := request.URL.Query()

		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil {
			return "", err
		}

		if len(records) < limit {
			return "", nil
		}

		offset := 0
		if value := query.Get("offset"); value != "" {
			offset, err = strconv.Atoi(value)
			if err != nil {
				return "", err
			}
		}

		return strconv.Itoa(offset + len(records)), nil
	}
}

// makeUpdatedTimeFilter keeps records whose last update falls within Since/Until.
// updatedAt is null for records that were never updated, then createdAt is used.
// Records are sorted by createdAt, not updatedAt, so pagination always continues.
func makeUpdatedTimeFilter(nextPageFunc common.NextPageFunc) common.RecordsFilterFunc {
	boundary := readhelper.NewTimeBoundary()

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
			timestamp, err := lastModified(record)
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

func lastModified(record *ajson.Node) (time.Time, error) {
	query := jsonquery.New(record)

	value, err := query.StringOptional("updatedAt")
	if err != nil {
		return time.Time{}, err
	}

	if value == nil {
		value, err = query.StringOptional("createdAt")
		if err != nil {
			return time.Time{}, err
		}
	}

	if value == nil {
		return time.Time{}, common.ErrMissingExpectedValues
	}

	return time.Parse(time.RFC3339, *value)
}

// flattenRecord lifts each typed field to the top level as its plain value
// ({"fields": {"$name": {"value": "Acme", "valueType": "TEXT"}}} becomes {"$name": "Acme"})
// and each relationship as its list of related ids.
// Top-level properties win over fields, and fields win over relationships with the same key;
// ListObjectMetadata applies the same precedence.
func flattenRecord(node *ajson.Node) (map[string]any, error) {
	record, err := jsonquery.Convertor.ObjectToMap(node)
	if err != nil {
		return nil, err
	}

	fields, _ := record[keyFields].(map[string]any)
	relationships, _ := record[keyRelationships].(map[string]any)

	delete(record, keyFields)
	delete(record, keyRelationships)

	liftNested(record, fields, "value")
	liftNested(record, relationships, "values")

	return record, nil
}

func liftNested(record map[string]any, entries map[string]any, valueKey string) {
	for key, entry := range entries {
		if _, taken := record[key]; taken {
			continue
		}

		if object, ok := entry.(map[string]any); ok {
			record[key] = object[valueKey]
		}
	}
}
