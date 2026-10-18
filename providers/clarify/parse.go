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
func makeNextPageFunc(requestURL *url.URL) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StringOptional("next")
		if err != nil || next == nil || *next == "" {
			return "", err
		}

		return sameOriginLink(*next, requestURL)
	}
}

// sameOriginLink moves a `links.next` URL onto the scheme and host of the request that returned it.
// The API builds these links with an http:// scheme (seen live:
// "http://api.clarify.ai/v1/workspaces/.../resources?page%5Boffset%5D=1&page%5Blimit%5D=1"),
// and following them as given would send the API key over plain HTTP.
// Path and query are kept verbatim, as the docs ask.
func sameOriginLink(link string, requestURL *url.URL) (string, error) {
	linkURL, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	linkURL.Scheme = requestURL.Scheme
	linkURL.Host = requestURL.Host

	return linkURL.String(), nil
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
