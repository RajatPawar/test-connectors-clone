package connectwise

import (
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// getRecords extracts the records from a ConnectWise list response.
// All list endpoints return a top-level JSON array with no wrapper key.
func getRecords(node *ajson.Node) ([]map[string]any, error) {
	arr, err := jsonquery.New(node).ArrayRequired("")
	if err != nil {
		return nil, err
	}

	return jsonquery.Convertor.ArrayToMap(arr)
}
