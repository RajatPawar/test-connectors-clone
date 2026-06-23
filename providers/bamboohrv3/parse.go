package bamboohrv3

import (
	"net/url"
	"strconv"

	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

func makeNextPage(objectName string) func(*ajson.Node) (string, error) {
	switch objectName {
	case objectNameEmployees:
		return nextPageForEmployees
	case objectNameApplications:
		return nextPageForApplications
	case objectNameSchedules:
		return nextPageForSchedules
	default:
		return func(_ *ajson.Node) (string, error) { return "", nil }
	}
}

// nextPageForEmployees extracts the cursor from meta.page.nextCursor.
func nextPageForEmployees(node *ajson.Node) (string, error) {
	cursor, err := jsonquery.New(node, "meta", "page").StringOptional("nextCursor")
	if err != nil || cursor == nil || *cursor == "" {
		return "", nil //nolint:nilerr
	}

	return *cursor, nil
}

// nextPageForApplications extracts the next page number from nextPageUrl.
func nextPageForApplications(node *ajson.Node) (string, error) {
	done, err := jsonquery.New(node).BoolWithDefault("paginationComplete", false)
	if err != nil || done {
		return "", nil //nolint:nilerr
	}

	nextURL, err := jsonquery.New(node).StringOptional("nextPageUrl")
	if err != nil || nextURL == nil || *nextURL == "" {
		return "", nil //nolint:nilerr
	}

	parsed, parseErr := url.Parse(*nextURL)
	if parseErr != nil {
		return "", nil
	}

	pageStr := parsed.Query().Get("page")
	if pageStr == "" {
		return "", nil
	}

	return pageStr, nil
}

// nextPageForSchedules computes the next page from meta.page, meta.pageSize, and meta.totalItems.
func nextPageForSchedules(node *ajson.Node) (string, error) {
	total, err := jsonquery.New(node, "meta").IntegerWithDefault("totalItems", 0)
	if err != nil {
		return "", nil //nolint:nilerr
	}

	page, err := jsonquery.New(node, "meta").IntegerWithDefault("page", 0)
	if err != nil || page == 0 {
		return "", nil //nolint:nilerr
	}

	pageSize, err := jsonquery.New(node, "meta").IntegerWithDefault("pageSize", schedulesPageSize)
	if err != nil {
		return "", nil //nolint:nilerr
	}

	if page*pageSize >= total {
		return "", nil
	}

	return strconv.FormatInt(page+1, 10), nil
}
