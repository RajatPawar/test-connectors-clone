package bamboographtrial5

import (
	"strconv"

	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextRecordsURL returns a NextRecordsFunc for page/pageSize-based pagination.
// If ps == 0, the object is not paginated and no next page is ever returned.
func makeNextRecordsURL(responseKey string, ps int, currentPage int) func(*ajson.Node) (string, error) {
	return func(node *ajson.Node) (string, error) {
		if ps == 0 {
			// Object is not paginated.
			return "", nil
		}

		// Determine the count of records on this page.
		var count int
		if responseKey == "" {
			// Root node is the array itself.
			count = node.Size()
		} else {
			records, err := jsonquery.New(node).ArrayOptional(responseKey)
			if err != nil || records == nil {
				return "", nil //nolint:nilerr
			}
			count = len(records)
		}

		if count < ps {
			// Fewer records than page size — we're on the last page.
			return "", nil
		}

		return strconv.Itoa(currentPage + 1), nil
	}
}
