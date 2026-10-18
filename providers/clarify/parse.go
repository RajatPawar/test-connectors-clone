package clarify

import (
	"net/url"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// nextPage returns links.next, which is null on the last page.
// https://developer.clarify.ai/docs/api-basics/pagination#response
//
// The API returns links.next with an http:// scheme (seen live:
// "http://api.clarify.ai/v1/workspaces/.../resources?page%5Boffset%5D=1&page%5Blimit%5D=1"),
// so the link's scheme and host are replaced by the catalog BaseURL's to keep the API key off plain HTTP.
// Path and query are kept verbatim, as the docs ask.
func (c *Connector) nextPage(node *ajson.Node) (string, error) {
	next, err := jsonquery.New(node, "links").StringOptional("next")
	if err != nil || next == nil {
		return "", err
	}

	nextURL, err := url.Parse(*next)
	if err != nil {
		return "", err
	}

	baseURL, err := url.Parse(c.ProviderInfo().BaseURL)
	if err != nil {
		return "", err
	}

	nextURL.Scheme = baseURL.Scheme
	nextURL.Host = baseURL.Host

	return nextURL.String(), nil
}

// makeUpdatedAtFilter applies Since (exclusive) and Until (inclusive) to users on the connector side,
// because GET /users documents no filter parameter. No ordering by update time is documented,
// so every page is read.
// https://developer.clarify.ai/docs/api-reference/users/getUsers
func makeUpdatedAtFilter(nextPage common.NextPageFunc) common.RecordsFilterFunc {
	return readhelper.MakeTimeFilterFuncWithZoom(
		readhelper.Unordered,
		readhelper.NewTimeBoundary(readhelper.ExcludeSince()),
		[]string{"attributes"}, "_updated_at", time.RFC3339,
		nextPage,
	)
}
