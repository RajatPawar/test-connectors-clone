package clarify

import (
	"net/url"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextRecordsURL returns links.next, which is null on the last page.
// Its path and percent-encoded query (page and filters) are kept as they are, but the live API
// returns it as http://api.clarify.ai/..., so the scheme and host are taken from the base URL:
// following it verbatim would send the API key over plain HTTP.
// TODO: the docs show https links; drop the rewrite if Clarify fixes links.next.
// https://developer.clarify.ai/docs/api-basics/pagination#response
func makeNextRecordsURL(baseURL string) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StringOptional("next")
		if err != nil || next == nil {
			return "", err
		}

		base, err := url.Parse(baseURL)
		if err != nil {
			return "", err
		}

		nextURL, err := url.Parse(*next)
		if err != nil {
			return "", err
		}

		nextURL.Scheme = base.Scheme
		nextURL.Host = base.Host

		return nextURL.String(), nil
	}
}

// makeUpdatedAtFilter keeps records whose attributes._updated_at falls within (Since, Until].
// It serves GET /users, which documents no filter parameters. Users are not sorted by update time,
// so every page is read: paging cannot stop early on a timestamp.
func makeUpdatedAtFilter(nextPage common.NextPageFunc) common.RecordsFilterFunc {
	return readhelper.MakeTimeFilterFuncWithZoom(
		readhelper.Unordered,
		readhelper.NewTimeBoundary(readhelper.ExcludeSince()),
		[]string{"attributes"}, fieldUpdatedAt, time.RFC3339,
		nextPage,
	)
}
