package orb

import (
	"net/url"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextPageFunc follows Orb's cursor pagination: while pagination_metadata.has_more is true,
// next_cursor is passed back as the `cursor` query parameter of the same request.
// https://docs.withorb.com/api-reference/pagination
func makeNextPageFunc(requestURL *url.URL) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		pagination := jsonquery.New(node, "pagination_metadata")

		hasMore, err := pagination.BoolWithDefault("has_more", false)
		if err != nil {
			return "", err
		}

		if !hasMore {
			return "", nil
		}

		cursor, err := pagination.StringOptional("next_cursor")
		if err != nil || cursor == nil || *cursor == "" {
			return "", err
		}

		nextURL, err := urlbuilder.FromRawURL(requestURL)
		if err != nil {
			return "", err
		}

		nextURL.WithQueryParam("cursor", *cursor)

		return nextURL.String(), nil
	}
}
