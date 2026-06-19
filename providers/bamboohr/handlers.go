package bamboohr

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers/bamboohr/metadata"
)

const (
	defaultPageSize = "500"
	pageSizeKey     = "pageSize"
	pageKey         = "page"
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	path, err := metadata.Schemas.LookupURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(pageSizeKey, defaultPageSize)

	if params.NextPage != "" {
		url.WithQueryParam(pageKey, params.NextPage.String())
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	responseKey := metadata.Schemas.LookupArrayFieldName(c.Module(), params.ObjectName)

	return common.ParseResult(
		response,
		common.ExtractRecordsFromPath(responseKey),
		makeNextRecordsURL(params),
		common.GetMarshaledData,
		params.Fields,
	)
}
