package nooks

import (
	"net/http"
	"net/url"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

const responseDataKey = "data"

// records extracts the "data" array present on every Nooks list response:
// {"data": [...], "links": {"next", "prev", "first"}}.
func records() common.NodeRecordsFunc {
	return common.MakeRecordsFunc(responseDataKey)
}

// nextRecordsURL reads links.next from the response body. docs/openapi_spec.json's
// PaginationLinks schema documents this as "returned as relative references
// (path + query) that should be resolved against the base URL of the request" --
// even though the spec's own worked examples show absolute URLs. Resolving via
// url.ResolveReference handles both cases correctly.
func nextRecordsURL(request *http.Request) common.NextPageFunc {
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

// makeFilterFunc returns the RecordsFilterFunc used by parseReadResponse.
//
// Objects with a native filter[updatedAt] query param (everything except
// tasks/callDispositions) already come back pre-filtered by the provider, so
// pagination just follows links.next unconditionally.
//
// tasks and callDispositions have no filter[updatedAt] parameter in the spec
// (see objectsWithoutNativeTimeFilter in handlers.go), so Since/Until are
// applied connector-side against each record's updatedAt field. The spec makes
// no claim about list ordering for these two objects, so Unordered is used:
// every page is walked and filtered, pagination never short-circuits early.
func makeFilterFunc(params common.ReadParams, nextPageFunc common.NextPageFunc) common.RecordsFilterFunc {
	if !objectsWithoutNativeTimeFilter.Has(params.ObjectName) {
		return readhelper.MakeIdentityFilterFunc(nextPageFunc)
	}

	if params.Since.IsZero() && params.Until.IsZero() {
		return readhelper.MakeIdentityFilterFunc(nextPageFunc)
	}

	return readhelper.MakeTimeFilterFunc(
		readhelper.Unordered,
		readhelper.NewTimeBoundary(),
		updatedAtField,
		time.RFC3339,
		nextPageFunc,
	)
}
