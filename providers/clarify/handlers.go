package clarify

import (
	"context"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

const (
	apiVersion = "v1"

	// List endpoints default to 500 records and return 422 above 500.
	// https://developer.clarify.ai/docs/api-basics/pagination
	maxPageSize = 500

	// Since is exclusive and Until inclusive, as in ReadParams.
	// The operator is a second bracket; its space must be sent as %20.
	// https://developer.clarify.ai/docs/api-basics/filtering#filter-with-an-operator
	updatedSinceParam = "filter[_updated_at][Is after]"
	updatedUntilParam = "filter[_updated_at][Is on or before]"
)

// The query encoder writes spaces as "+"; the docs send operator names as `Greater%20than`.
var queryEncodingExceptions = map[string]string{ // nolint:gochecknoglobals
	"+": "%20",
}

// buildReadRequest lists records with page[limit]/page[offset] pagination and follows links.next verbatim,
// as the docs ask: "Pass them through unchanged rather than rebuilding them."
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

	// GET /users documents only page and sortOrder, so user is filtered in parseReadResponse.
	// https://developer.clarify.ai/docs/api-reference/users/getUsers
	if params.ObjectName != objectNameUser {
		if !params.Since.IsZero() {
			readURL.WithQueryParam(updatedSinceParam, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Since))
		}

		if !params.Until.IsZero() {
			readURL.WithQueryParam(updatedUntilParam, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Until))
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
	filter := readhelper.MakeIdentityFilterFunc(nextPage)
	if params.ObjectName == objectNameUser {
		filter = filterByUpdatedAt
	}

	return common.ParseResultFiltered(params, response,
		common.MakeRecordsFunc("data"),
		filter,
		readhelper.MakeMarshaledDataFuncWithId(common.FlattenNestedFields("attributes"), readhelper.NewIdField("id")),
		params.Fields,
	)
}
