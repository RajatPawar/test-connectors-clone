package orb

import (
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// nextPageCursor returns `pagination_metadata.next_cursor` while `has_more` is true.
// https://docs.withorb.com/api-reference/pagination#paginated-api-responses
func nextPageCursor(node *ajson.Node) (string, error) {
	pagination, err := jsonquery.New(node).ObjectOptional("pagination_metadata")
	if err != nil || pagination == nil {
		return "", err
	}

	hasMore, err := jsonquery.New(pagination).BoolWithDefault("has_more", false)
	if err != nil || !hasMore {
		return "", err
	}

	return jsonquery.New(pagination).StrWithDefault("next_cursor", "")
}
