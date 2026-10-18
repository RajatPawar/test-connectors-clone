package clarify

import (
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// getNextRecordsURL returns links.next, which is null on the last page.
// https://developer.clarify.ai/docs/api-basics/pagination#response
func getNextRecordsURL(node *ajson.Node) (string, error) {
	next, err := jsonquery.New(node, "links").StringOptional("next")
	if err != nil || next == nil {
		return "", err
	}

	return *next, nil
}

// makeUpdatedAtFilter keeps records whose attributes._updated_at falls within (Since, Until].
// It serves GET /users, which documents no filter parameters. Users come back ordered by _id,
// so every page is read: paging cannot stop early on a timestamp.
func makeUpdatedAtFilter(nextPage common.NextPageFunc) common.RecordsFilterFunc {
	return readhelper.MakeTimeFilterFuncWithZoom(
		readhelper.Unordered,
		readhelper.NewTimeBoundary(readhelper.ExcludeSince()),
		[]string{"attributes"}, fieldUpdatedAt, time.RFC3339,
		nextPage,
	)
}
