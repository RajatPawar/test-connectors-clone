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

	// Most list endpoints accept page[limit] up to 500 and return 422 above that.
	// https://developer.clarify.ai/docs/api-basics/pagination
	maxPageSize = 500

	// Datetime filter operators, as `filter[{field}][{operator}]`. Operator names are case-sensitive.
	// `Is after` is exclusive and `Is on or before` inclusive, matching ReadParams Since and Until.
	// https://developer.clarify.ai/docs/api-basics/filtering#operators-by-field-type
	updatedSinceParam = "filter[" + fieldUpdatedAt + "][Is after]"
	updatedUntilParam = "filter[" + fieldUpdatedAt + "][Is on or before]"

	fieldUpdatedAt = "_updated_at"
)

// buildReadRequest lists records with page[limit]/page[offset] and then follows links.next verbatim,
// as the docs ask: link URLs are percent-encoded and carry the filters over.
// https://developer.clarify.ai/docs/api-basics/pagination
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	readURL, err := c.readURL(params)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, readURL, nil)
}

func (c *Connector) readURL(params common.ReadParams) (string, error) {
	if params.NextPage != "" {
		return params.NextPage.String(), nil
	}

	readURL, err := c.workspaceURL(objectPath(params.ObjectName)...)
	if err != nil {
		return "", err
	}

	pageSize := min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)
	readURL.WithQueryParam("page[limit]", strconv.Itoa(pageSize))
	readURL.WithQueryParam("page[offset]", "0")

	// The resources endpoints filter on any field, including the system field _updated_at,
	// which is stored with millisecond precision.
	// GET /users documents no filter, so users are filtered in parseReadResponse.
	if resourceObjects.Has(params.ObjectName) {
		if !params.Since.IsZero() {
			readURL.WithQueryParam(updatedSinceParam, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Since))
		}

		if !params.Until.IsZero() {
			readURL.WithQueryParam(updatedUntilParam, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Until))
		}
	}

	// Operator names contain spaces, which the docs encode as %20.
	readURL.AddEncodingExceptions(map[string]string{"+": "%20"})

	return readURL.String(), nil
}

func (c *Connector) workspaceURL(path ...string) (*urlbuilder.URL, error) {
	return urlbuilder.New(c.ProviderInfo().BaseURL, append([]string{apiVersion, "workspaces", c.Workspace}, path...)...)
}

// objectPath is the path under /v1/workspaces/{workspace} that lists an object's records.
// https://developer.clarify.ai/docs/api-reference/resources/getResources
// https://developer.clarify.ai/docs/api-reference/users/getUsers
func objectPath(objectName string) []string {
	if objectName == objectNameUsers {
		return []string{"users"}
	}

	return []string{"objects", objectName, "resources"}
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	filter := readhelper.MakeIdentityFilterFunc(getNextRecordsURL)
	if !resourceObjects.Has(params.ObjectName) {
		filter = makeUpdatedAtFilter(getNextRecordsURL)
	}

	return common.ParseResultFiltered(params, response,
		common.MakeRecordsFunc("data"),
		filter,
		readhelper.MakeMarshaledDataFuncWithId(common.FlattenNestedFields("attributes"), readhelper.NewIdField("id")),
		params.Fields,
	)
}
