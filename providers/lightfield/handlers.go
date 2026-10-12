package lightfield

import (
	"context"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers/lightfield/internal/metadata"
)

const (
	apiPathVersion = "v1"

	// Every request must carry this header; requests may error without it.
	// https://docs.lightfield.app/using-the-api/api-keys
	apiVersionHeader = "Lightfield-Version"
	apiVersion       = "2026-03-01"

	// List endpoints return at most 25 records per page.
	// https://docs.lightfield.app/using-the-api/list-endpoints#pagination
	maxPageSize = 25
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	url, err := c.objectURL(params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("limit", strconv.Itoa(pageSize(params)))

	if params.NextPage != "" {
		url.WithQueryParam("offset", params.NextPage.String())
	}

	return newGetRequest(ctx, url)
}

// parseReadResponse filters by Since/Until on the connector side.
// List endpoints are sorted by createdAt and have no documented filter on updatedAt,
// so every page is read and records are kept by their updatedAt.
func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResultFiltered(
		params,
		response,
		common.MakeRecordsFunc("data"),
		makeUpdatedTimeFilter(makeNextPageFunc(request)),
		readhelper.MakeMarshaledDataFuncWithId(flattenRecord, readhelper.NewIdField("id")),
		params.Fields,
	)
}

func (c *Connector) objectURL(objectName string, paths ...string) (*urlbuilder.URL, error) {
	path, err := metadata.Catalog.ReadPath(objectName)
	if err != nil {
		return nil, err
	}

	return urlbuilder.New(c.ProviderInfo().BaseURL, append([]string{apiPathVersion, path}, paths...)...)
}

func newGetRequest(ctx context.Context, url *urlbuilder.URL) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set(apiVersionHeader, apiVersion)

	return request, nil
}

func pageSize(params common.ReadParams) int {
	return min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)
}
