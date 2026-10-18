package clarify

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

const (
	apiVersion = "v1"

	// List endpoints default to 500 records and return 422 above 500.
	// https://developer.clarify.ai/docs/api-basics/pagination
	maxPageSize = 500

	// The operator is a second bracket; its space must be sent as %20.
	// https://developer.clarify.ai/docs/api-basics/filtering#filter-with-an-operator
	updatedSinceParam = "filter[_updated_at][Is after]"
	updatedUntilParam = "filter[_updated_at][Is on or before]"

	// The _updated_at filter compares whole days, not instants. Live, on a person updated at
	// 2025-08-27T18:36:13.720Z: `Is after 2025-08-27T18:36:13.721Z` still returned it, and
	// `Is on or before 2025-08-27T18:36:13.721Z` did not, while `Is on or before 2025-08-28T00:00:00.000Z` did.
	// The day boundary's time zone is not documented. So the provider is sent a window widened by two days on
	// each side, as dates like the docs' own example (`filter[_created_at][Is after]=2024-01-01`), and
	// parseReadResponse trims it to the exact (Since, Until].
	// TODO: Ask Clarify whether date-time filters on _updated_at can compare instants; then drop the slack.
	filterDateLayout = "2006-01-02"
	filterSlack      = 48 * time.Hour
)

// The query encoder writes spaces as "+"; the docs send operator names as `Greater%20than`.
var queryEncodingExceptions = map[string]string{ // nolint:gochecknoglobals
	"+": "%20",
}

// buildReadRequest lists records with page[limit]/page[offset] pagination and follows links.next,
// whose origin nextPage pins to the BaseURL.
// https://developer.clarify.ai/docs/api-basics/pagination
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		return http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
	}

	readURL, err := c.readURL(params)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, readURL.String(), nil)
}

func (c *Connector) readURL(params common.ReadParams) (*urlbuilder.URL, error) {
	readURL, err := c.workspaceURL(readPath(params.ObjectName)...)
	if err != nil {
		return nil, err
	}

	readURL.WithQueryParam("page[limit]",
		strconv.Itoa(min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)))

	// GET /users documents only page and sortOrder, so users are filtered in parseReadResponse alone.
	// https://developer.clarify.ai/docs/api-reference/users/getUsers
	if params.ObjectName != objectNameUser {
		if !params.Since.IsZero() {
			readURL.WithQueryParam(updatedSinceParam,
				params.Since.UTC().Add(-filterSlack).Format(filterDateLayout))
		}

		if !params.Until.IsZero() {
			readURL.WithQueryParam(updatedUntilParam,
				params.Until.UTC().Add(filterSlack).Format(filterDateLayout))
		}
	}

	readURL.AddEncodingExceptions(queryEncodingExceptions)

	return readURL, nil
}

// readPath is relative to /v1/workspaces/{workspace}.
// https://developer.clarify.ai/docs/api-reference/resources/getResources
// https://developer.clarify.ai/docs/api-reference/users/getUsers
func readPath(objectName string) []string {
	if objectName == objectNameUser {
		return []string{"users"}
	}

	return []string{"objects", objectName, "resources"}
}

func (c *Connector) workspaceURL(path ...string) (*urlbuilder.URL, error) {
	return urlbuilder.New(c.ProviderInfo().BaseURL, append([]string{apiVersion, "workspaces", c.Workspace}, path...)...)
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResultFiltered(params, response,
		common.MakeRecordsFunc("data"),
		makeUpdatedAtFilter(c.nextPage),
		readhelper.MakeMarshaledDataFuncWithId(common.FlattenNestedFields("attributes"), readhelper.NewIdField("id")),
		params.Fields,
	)
}
