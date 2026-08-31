package nooks

import (
	"net/http"
	"net/url"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// dataKey is the top-level envelope key every Nooks list endpoint uses:
// {"data": [...], "links": {...}}.
const dataKey = "data"

// recordsFunc extracts the `data` array shared by every Nooks list response.
//
//nolint:gochecknoglobals
var recordsFunc = common.ExtractRecordsFromPath(dataKey)

// nodeRecordsFunc is the ajson.Node-returning variant of recordsFunc, used by
// objects that need connector-side time filtering (see handlers.go).
//
//nolint:gochecknoglobals
var nodeRecordsFunc = common.MakeRecordsFunc(dataKey)

// nextPageFunc reads links.next and resolves it against the request URL.
//
// The PaginationLinks schema documents links.next as "returned as relative
// references (path + query) that should be resolved against the base URL of
// the request", but at least one response example in the spec shows a full
// absolute URL instead. url.ResolveReference handles both cases correctly.
func nextPageFunc(request *http.Request) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StrWithDefault("next", "")
		if err != nil || next == "" {
			return "", err
		}

		parsed, err := url.Parse(next)
		if err != nil {
			return "", err
		}

		return request.URL.ResolveReference(parsed).String(), nil
	}
}
