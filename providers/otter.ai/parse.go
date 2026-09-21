package otterai

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// records returns a RecordsFunc for the object's response shape. List
// endpoints (conversations, channels) wrap their array under a top-level
// "data" field. GET /workspace returns a single object under "data" instead
// of an array — it is wrapped here as a one-element slice so it flows through
// the same marshal path as list objects.
func records(objectName string) common.RecordsFunc {
	if objectName == objectNameWorkspace {
		return func(node *ajson.Node) ([]map[string]any, error) {
			obj, err := jsonquery.New(node).ObjectOptional("data")
			if err != nil {
				return nil, err
			}

			if obj == nil {
				return []map[string]any{}, nil
			}

			record, err := jsonquery.Convertor.ObjectToMap(obj)
			if err != nil {
				return nil, err
			}

			return []map[string]any{record}, nil
		}
	}

	return func(node *ajson.Node) ([]map[string]any, error) {
		arr, err := jsonquery.New(node).ArrayOptional("data")
		if err != nil {
			return nil, err
		}

		return jsonquery.Convertor.ArrayToMap(arr)
	}
}

// nextRecordsURL reads the cursor-pagination fields documented for list
// endpoints: meta.has_more (bool) and meta.next_cursor (string). Only
// GET /conversations documents this pairing; GET /channels and GET /workspace
// examples show only meta.retrieved_at, so has_more/next_cursor are simply
// absent there and this returns "" (single page), which is the correct,
// docs-grounded default for those objects.
func nextRecordsURL() common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		meta, err := jsonquery.New(node).ObjectOptional("meta")
		if err != nil || meta == nil {
			return "", err
		}

		hasMore, err := jsonquery.New(meta).BoolOptional("has_more")
		if err != nil {
			return "", err
		}

		if hasMore == nil || !*hasMore {
			return "", nil
		}

		nextCursor, err := jsonquery.New(meta).StringOptional("next_cursor")
		if err != nil || nextCursor == nil {
			return "", nil
		}

		return *nextCursor, nil
	}
}
