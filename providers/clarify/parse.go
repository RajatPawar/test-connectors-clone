package clarify

import (
	"time"

	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// nextPage returns links.next, which is null on the last page.
// https://developer.clarify.ai/docs/api-basics/pagination#response
func nextPage(node *ajson.Node) (string, error) {
	next, err := jsonquery.New(node, "links").StringOptional("next")
	if err != nil || next == nil {
		return "", err
	}

	return *next, nil
}

// filterByUpdatedAt applies Since (exclusive) and Until (inclusive) to users on the connector side,
// because GET /users has no filter parameter. No ordering by update time is documented, so every page is read.
// https://developer.clarify.ai/docs/api-reference/users/getUsers
var filterByUpdatedAt = readhelper.MakeTimeFilterFuncWithZoom( // nolint:gochecknoglobals
	readhelper.Unordered,
	readhelper.NewTimeBoundary(readhelper.ExcludeSince()),
	[]string{"attributes"}, "_updated_at", time.RFC3339,
	nextPage,
)
