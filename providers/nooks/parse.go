package nooks

import (
	"net/url"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// records extracts the `data` array present on every Nooks list response.
func records(node *ajson.Node) ([]map[string]any, error) {
	return common.ExtractRecordsFromPath("data")(node)
}

// recordNodes is the ajson.Node-preserving counterpart of records, used by the
// connector-side-filtered read path (common.ParseResultFiltered).
func recordNodes(node *ajson.Node) ([]*ajson.Node, error) {
	return common.MakeRecordsFunc("data")(node)
}

// nextPageFunc reads `links.next`. The Nooks spec documents this as a
// relative reference (path + query) that must be resolved against the
// request's base URL, even though the spec's own worked examples show a full
// absolute URL for it. Resolving via url.ResolveReference is correct for both
// cases: it is a no-op when the value is already absolute.
func nextPageFunc(requestURL *url.URL) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StrWithDefault("next", "")
		if err != nil || next == "" {
			return "", err
		}

		parsed, err := url.Parse(next)
		if err != nil {
			return "", err
		}

		return requestURL.ResolveReference(parsed).String(), nil
	}
}
