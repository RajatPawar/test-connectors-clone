package clarify

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

const (
	apiVersion = "v1"

	// List endpoints default to 500 records per page in the API reference and reject
	// a `page[limit]` above 500 with a 422.
	// https://developer.clarify.ai/docs/api-basics/pagination
	maxPageSize = 500

	// Datetime filter operators, `filter[{field}][{operator}]`. Operator names are case-sensitive
	// and contain spaces, which the docs send as %20.
	// https://developer.clarify.ai/docs/api-basics/filtering#operators-by-field-type
	updatedAfterParam  = "filter[_updated_at][Is after]"
	updatedBeforeParam = "filter[_updated_at][Is before]"
)

// buildReadRequest lists one page of records. The next page is the response's `links.next`,
// which carries the filters and is fetched as is (see makeNextPageFunc).
// https://developer.clarify.ai/docs/api-basics/pagination#response
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
	readURL, err := c.objectURL(params.ObjectName)
	if err != nil {
		return nil, err
	}

	readURL.AddEncodingExceptions(map[string]string{"+": "%20"})
	readURL.WithQueryParam("page[limit]",
		strconv.Itoa(min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)))

	// /users documents no filter parameter.
	if !resourceObjects.Has(params.ObjectName) {
		return readURL, nil
	}

	// `_updated_at` carries milliseconds, but the filter compares whole seconds: live, a 1 ms window
	// around a record's `_updated_at` matched nothing while a 2 s window matched it. The request
	// therefore asks for a window widened to whole seconds, one second each side, and
	// makeUpdatedAtFilter trims it to (Since, Until].
	if !params.Since.IsZero() {
		after := params.Since.Truncate(time.Second).Add(-time.Second)
		readURL.WithQueryParam(updatedAfterParam, datautils.Time.FormatRFC3339inUTC(after))
	}

	if !params.Until.IsZero() {
		before := params.Until.Truncate(time.Second).Add(time.Second)
		readURL.WithQueryParam(updatedBeforeParam, datautils.Time.FormatRFC3339inUTC(before))
	}

	return readURL, nil
}

// objectURL is the collection endpoint of an object:
// person, company, deal, meeting, task: /v1/workspaces/{workspace}/objects/{object}/resources
// https://developer.clarify.ai/docs/api-reference/resources/getResources
// users: /v1/workspaces/{workspace}/users
// https://developer.clarify.ai/docs/api-reference/users/getUsers
func (c *Connector) objectURL(objectName string) (*urlbuilder.URL, error) {
	if objectName == objectNameUsers {
		return urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, "workspaces", c.Workspace, objectNameUsers)
	}

	return urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, "workspaces", c.Workspace,
		"objects", objectName, "resources")
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResultFiltered(params, response,
		common.MakeRecordsFunc("data"),
		makeUpdatedAtFilter(makeNextPageFunc(request.URL)),
		readhelper.MakeMarshaledDataFuncWithId(common.FlattenNestedFields("attributes"), readhelper.NewIdField("id")),
		params.Fields,
	)
}
