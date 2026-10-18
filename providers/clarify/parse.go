package clarify

import (
	"net/url"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextPageFunc returns `links.next`, which is null on the last page.
// https://developer.clarify.ai/docs/api-basics/pagination#response
// The API builds the link with an http:// scheme (seen live:
// "http://api.clarify.ai/v1/workspaces/.../resources?page%5Boffset%5D=1&page%5Blimit%5D=1"),
// so the scheme and host are taken from the request just made, keeping the API key off plain HTTP.
// Path and query are kept verbatim, as the docs ask.
func makeNextPageFunc(requestURL *url.URL) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StringOptional("next")
		if err != nil || next == nil || *next == "" {
			return "", err
		}

		nextURL, err := url.Parse(*next)
		if err != nil {
			return "", err
		}

		nextURL.Scheme = requestURL.Scheme
		nextURL.Host = requestURL.Host

		return nextURL.String(), nil
	}
}

// makeUpdatedAtFilter keeps records whose `_updated_at` is within (Since, Until].
// For person, company, deal, meeting and task it trims the whole-second window the request asked
// for (see readURL); /users documents no filter at all, so every user is read and filtered here.
// Records come back ordered by `_id`, not by time, so paging cannot stop early.
func makeUpdatedAtFilter(nextPage common.NextPageFunc) common.RecordsFilterFunc {
	return readhelper.MakeTimeFilterFuncWithZoom(
		readhelper.Unordered,
		readhelper.NewTimeBoundary(readhelper.ExcludeSince()),
		[]string{"attributes"}, "_updated_at", time.RFC3339,
		nextPage,
	)
}
