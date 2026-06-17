package tipalti

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

func records() common.RecordsFunc {
	return func(node *ajson.Node) ([]map[string]any, error) {
		items, err := jsonquery.New(node).ArrayOptional("items")
		if err != nil {
			return nil, err
		}

		return jsonquery.Convertor.ArrayToMap(items)
	}
}

func nextPageCursor() common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		hasNext, err := jsonquery.New(node, "pageInfo").BoolOptional("hasNextPage")
		if err != nil {
			return "", err
		}

		if hasNext == nil || !*hasNext {
			return "", nil
		}

		cursor, err := jsonquery.New(node, "pageInfo").StringOptional("nextPageCursor")
		if err != nil {
			return "", err
		}

		if cursor == nil {
			return "", nil
		}

		return *cursor, nil
	}
}
