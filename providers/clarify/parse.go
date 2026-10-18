package clarify

import (
	"fmt"
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

// makeUpdatedAtFilter keeps records whose attributes._updated_at falls within (Since, Until].
// The resources endpoints are pre-filtered by whole days (see filterSlack) and GET /users not at all,
// so the exact window is applied here. Records are not ordered by update time, so every page is read.
// A record without _updated_at is placed at its _created_at.
func makeUpdatedAtFilter(nextPage common.NextPageFunc) common.RecordsFilterFunc {
	boundary := readhelper.NewTimeBoundary(readhelper.ExcludeSince())

	return func(params common.ReadParams, body *ajson.Node, records []*ajson.Node) ([]*ajson.Node, string, error) {
		next, err := nextPage(body)
		if err != nil {
			return nil, "", err
		}

		if params.Since.IsZero() && params.Until.IsZero() {
			return records, next, nil
		}

		filtered := make([]*ajson.Node, 0, len(records))

		for _, record := range records {
			timestamp, err := lastModified(record)
			if err != nil {
				return nil, "", err
			}

			if boundary.Contains(params, timestamp) {
				filtered = append(filtered, record)
			}
		}

		return filtered, next, nil
	}
}

func lastModified(record *ajson.Node) (time.Time, error) {
	attributes := jsonquery.New(record, "attributes")

	for _, key := range []string{"_updated_at", "_created_at"} {
		value, err := attributes.StringOptional(key)
		if err != nil {
			return time.Time{}, err
		}

		if value != nil {
			return time.Parse(time.RFC3339, *value)
		}
	}

	return time.Time{}, fmt.Errorf("%w: _updated_at and _created_at", common.ErrMissingExpectedValues)
}
