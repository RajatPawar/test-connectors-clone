package activecampaign

import (
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

const metaField = "meta"

// recordsFromNode extracts the collection nested under the top-level plural key
// named after the object (e.g. {"contacts": [...], "meta": {...}}).
func recordsFromNode(objectName string) common.RecordsFunc {
	return func(node *ajson.Node) ([]map[string]any, error) {
		records, err := jsonquery.New(node).ArrayOptional(objectName)
		if err != nil {
			return nil, err
		}

		return jsonquery.Convertor.ArrayToMap(records)
	}
}

// nextRecordsURL drives zero-based offset pagination using meta.total, which is
// a sibling of the collection key. meta.total may be a JSON string or int, so it
// is parsed defensively.
func nextRecordsURL(currentOffset int) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		meta, err := jsonquery.New(node).ObjectOptional(metaField)
		if err != nil {
			return "", err
		}

		if meta == nil {
			return "", nil
		}

		totalStr, err := jsonquery.New(meta).TextWithDefault("total", "0")
		if err != nil {
			return "", err
		}

		total, err := strconv.Atoi(totalStr)
		if err != nil {
			return "", nil //nolint:nilerr // unparseable total means stop paginating
		}

		nextOffset := currentOffset + pageSize
		if nextOffset < total {
			return strconv.Itoa(nextOffset), nil
		}

		return "", nil
	}
}
