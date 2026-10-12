package lightfield

import (
	"net/url"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextRecordsOffset returns the offset of the next page.
// There are no more records when a page holds fewer records than the requested limit,
// or when the offset reaches totalCount.
// https://docs.lightfield.app/using-the-api/list-endpoints#pagination
func makeNextRecordsOffset(requestURL *url.URL) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		query := jsonquery.New(node)

		records, err := query.ArrayRequired(responseKeyRecords)
		if err != nil {
			return "", err
		}

		totalCount, err := query.IntegerOptional("totalCount")
		if err != nil {
			return "", err
		}

		limit, err := intQueryParam(requestURL, "limit")
		if err != nil {
			return "", err
		}

		offset, err := intQueryParam(requestURL, "offset")
		if err != nil {
			return "", err
		}

		nextOffset := offset + len(records)

		if len(records) < limit || (totalCount != nil && int64(nextOffset) >= *totalCount) {
			return "", nil
		}

		return strconv.Itoa(nextOffset), nil
	}
}

func intQueryParam(requestURL *url.URL, name string) (int, error) {
	value := requestURL.Query().Get(name)
	if value == "" {
		return 0, nil
	}

	return strconv.Atoi(value)
}

// makeFilterFunc keeps records modified within the Since/Until window.
//
// List methods offer no updated-at filter, and records come sorted by createdAt (newest first),
// so every page is read and filtered connector-side. A record's modification time is its updatedAt,
// or its createdAt when updatedAt is null (the docs describe updatedAt as nullable).
// https://docs.lightfield.app/api/resources/account/methods/list
//
// TODO: list methods filter DATETIME fields with greaterThan/lessThan, but the docs do not say whether
// updatedAt is filterable that way. If Lightfield confirms it, filter provider-side instead.
// https://docs.lightfield.app/using-the-api/list-endpoints#filtering
func makeFilterFunc(nextPageFunc common.NextPageFunc) common.RecordsFilterFunc {
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
			modifiedAt, err := recordModifiedAt(record)
			if err != nil {
				return nil, "", err
			}

			if boundary.Contains(params, modifiedAt) {
				filtered = append(filtered, record)
			}
		}

		return filtered, next, nil
	}
}

func recordModifiedAt(record *ajson.Node) (time.Time, error) {
	query := jsonquery.New(record)

	timestamp, err := query.StringOptional("updatedAt")
	if err != nil {
		return time.Time{}, err
	}

	if timestamp == nil || *timestamp == "" {
		createdAt, err := query.StringRequired("createdAt")
		if err != nil {
			return time.Time{}, err
		}

		timestamp = &createdAt
	}

	return time.Parse(time.RFC3339, *timestamp)
}

// flattenRecord lifts each entry of "fields" (its value) and "relationships" (its related IDs)
// to the top level, next to id, createdAt, updatedAt, httpLink and externalId.
// Field keys are kept as the API returns them: "$name" for system fields, bare slugs for custom ones.
// On a key collision the first source wins: top-level properties, then fields, then relationships.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#field-keys
func flattenRecord(node *ajson.Node) (map[string]any, error) {
	record, err := jsonquery.Convertor.ObjectToMap(node)
	if err != nil {
		return nil, err
	}

	fields, _ := record["fields"].(map[string]any)
	relationships, _ := record["relationships"].(map[string]any)

	delete(record, "fields")
	delete(record, "relationships")

	for key, field := range fields {
		if entry, ok := field.(map[string]any); ok {
			setIfAbsent(record, key, entry["value"])
		}
	}

	for key, relationship := range relationships {
		if entry, ok := relationship.(map[string]any); ok {
			setIfAbsent(record, key, entry["values"])
		}
	}

	return record, nil
}

func setIfAbsent(record map[string]any, key string, value any) {
	if _, exists := record[key]; !exists {
		record[key] = value
	}
}
