package nooks

import (
	"net/http"
	"net/url"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// recordsFunc extracts the `data` array present on every Nooks list response,
// as map[string]any records (for common.ParseResult / server-side-filtered reads).
func recordsFunc(node *ajson.Node) ([]map[string]any, error) {
	return common.ExtractOptionalRecordsFromPath("data")(node)
}

// nodeRecordsFunc is the ajson.Node variant of recordsFunc, used by
// common.ParseResultFiltered for connector-side (tasks) filtering.
func nodeRecordsFunc(node *ajson.Node) ([]*ajson.Node, error) {
	return jsonquery.New(node).ArrayOptional("data")
}

// makeNextPageFunc reads `links.next` from a Nooks list response and resolves
// it into an absolute URL.
//
// Per docs/openapi_spec.json's PaginationLinks schema, `links.next` may be
// returned as a relative reference (path + query) that "should be resolved
// against the base URL of the request" -- it is not guaranteed to be
// absolute. We resolve it against the URL of the request that produced this
// response, so the token stored in ReadResult.NextPage is always a URL that
// buildReadRequest can GET directly on the next call.
func makeNextPageFunc(request *http.Request) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StrWithDefault("next", "")
		if err != nil || next == "" {
			return "", nil //nolint:nilerr
		}

		ref, err := url.Parse(next)
		if err != nil {
			return "", err
		}

		return request.URL.ResolveReference(ref).String(), nil
	}
}
