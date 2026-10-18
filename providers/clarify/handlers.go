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

	// List endpoints default to 500 records per page in the API reference and reject
	// a `page[limit]` above 500 with a 422.
	// https://developer.clarify.ai/docs/api-basics/pagination
	maxPageSize = 500

	// Datetime filter operators, `filter[{field}][{operator}]`. Operator names are case-sensitive
	// and contain spaces, which the docs send as %20. They match ReadParams: Since is exclusive
	// and Until inclusive.
	// https://developer.clarify.ai/docs/api-basics/filtering#operators-by-field-type
	updatedSinceParam = "filter[_updated_at][Is after]"
	updatedUntilParam = "filter[_updated_at][Is on or before]"
)

// buildReadRequest lists one page of records. The next page is the response's `links.next`,
// which the docs say to fetch verbatim: it is percent-encoded and carries the filters.
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

	// /users documents no filter parameter; it is filtered in parseReadResponse.
	if resourceObjects.Has(params.ObjectName) {
		// TODO: The docs show datetime operators only with a date value (`[Is after]=2024-01-01`).
		// Timestamps carry milliseconds (`2026-02-01T17:45:00.000Z`), so the bounds are sent at
		// that precision; whether the API compares at it rather than by day is not documented.
		if !params.Since.IsZero() {
			readURL.WithQueryParam(updatedSinceParam, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Since))
		}

		if !params.Until.IsZero() {
			readURL.WithQueryParam(updatedUntilParam, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Until))
		}
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
	filter := readhelper.MakeIdentityFilterFunc(getNextPageURL)
	if !resourceObjects.Has(params.ObjectName) {
		filter = makeUpdatedAtFilter()
	}

	return common.ParseResultFiltered(params, response,
		common.MakeRecordsFunc("data"),
		filter,
		readhelper.MakeMarshaledDataFuncWithId(common.FlattenNestedFields("attributes"), readhelper.NewIdField("id")),
		params.Fields,
	)
}
