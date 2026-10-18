package clarify

import (
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// getNextPageURL returns `links.next`, which is null on the last page.
// https://developer.clarify.ai/docs/api-basics/pagination#response
func getNextPageURL(node *ajson.Node) (string, error) {
	next, err := jsonquery.New(node, "links").StringOptional("next")
	if err != nil || next == nil {
		return "", err
	}

	return *next, nil
}

// makeUpdatedAtFilter keeps users whose `_updated_at` is within (Since, Until].
// GET /users documents only page and sortOrder parameters, so every page is read.
// Without sortOrder records come back ordered by `_id`, so paging cannot stop early.
func makeUpdatedAtFilter() common.RecordsFilterFunc {
	return readhelper.MakeTimeFilterFuncWithZoom(
		readhelper.Unordered,
		readhelper.NewTimeBoundary(readhelper.ExcludeSince()),
		[]string{"attributes"}, "_updated_at", time.RFC3339,
		getNextPageURL,
	)
}
