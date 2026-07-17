package sagehr

import (
	"strconv"

	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

const (
	metaField     = "meta"
	nextPageField = "next_page"
)

// nextRecordsURL reads Sage HR's `meta.next_page` (an integer page number, or
// null on the last page). Endpoints that don't paginate (e.g. documents/categories)
// omit `meta` entirely, in which case this reports no next page.
func nextRecordsURL(node *ajson.Node) (string, error) {
	meta, err := jsonquery.New(node).ObjectOptional(metaField)
	if err != nil {
		return "", err
	}

	nextPage, err := jsonquery.New(meta).IntegerOptional(nextPageField)
	if err != nil {
		return "", err
	}

	if nextPage == nil {
		return "", nil
	}

	return strconv.FormatInt(*nextPage, 10), nil
}
