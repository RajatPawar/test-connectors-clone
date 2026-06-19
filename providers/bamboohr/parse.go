package bamboohr

import (
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextRecordsURL returns the next page token using a heuristic: if the
// number of records returned equals defaultPageSize, there may be more pages.
// BambooHR does not consistently include pagination metadata in responses.
// TODO: verify response shape per object and use server-side pagination signals.
func makeNextRecordsURL(params common.ReadParams) common.NextPageFunc {
	pageSize, _ := strconv.Atoi(defaultPageSize) //nolint:errcheck // constant is valid

	return func(node *ajson.Node) (string, error) {
		// Determine the current page number from the NextPage token.
		currentPage := 0
		if params.NextPage != "" {
			if p, err := strconv.Atoi(params.NextPage.String()); err == nil {
				currentPage = p
			}
		}

		// Count returned records to detect last page.
		// If fewer records than pageSize were returned, this is the last page.
		count, err := countRecords(node)
		if err != nil || count < pageSize {
			return "", nil
		}

		return strconv.Itoa(currentPage + 1), nil
	}
}

// countRecords counts records in the response — handles both root-array
// responses (responseKey=="") and keyed responses.
func countRecords(node *ajson.Node) (int, error) {
	if node.IsArray() {
		arr, err := node.GetArray()
		if err != nil {
			return 0, err
		}

		return len(arr), nil
	}

	// Try to find any array at the root or one level deep.
	obj, err := jsonquery.Convertor.ObjectToMap(node)
	if err != nil {
		return 0, nil
	}

	// Find the first array value in the response object.
	for _, v := range obj {
		if arr, ok := v.([]any); ok {
			return len(arr), nil
		}
	}

	return 0, nil
}
