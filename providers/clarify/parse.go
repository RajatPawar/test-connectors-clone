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
// https://developer.clarify.ai/docs/api-basics/pagination#response
func makeNextRecordsURL(baseURL string) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StringOptional("next")
		if err != nil || next == nil {
			return "", err
		}

		return onBaseURL(baseURL, *next)
	}
}

// onBaseURL keeps a links.next URL's path and percent-encoded query (page, cursor, filters) but
// takes the scheme and host from the base URL. The live API returns links.next as
// http://api.clarify.ai/..., and following it verbatim would send the API key over plain HTTP.
// Every links.next the connector follows (records and schemas) goes through here.
// TODO: the docs show https links; drop the rewrite if Clarify fixes links.next.
func onBaseURL(baseURL, link string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	linkURL, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	linkURL.Scheme = base.Scheme
	linkURL.Host = base.Host

	return linkURL.String(), nil
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
