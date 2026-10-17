package lightfield

import (
	"context"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

const (
	apiVersion = "v1"

	// Every request must carry this header; a missing or invalid one returns 400 `version_header`.
	// https://docs.lightfield.app/using-the-api/errors#error-codes
	versionHeader = "Lightfield-Version"
	versionValue  = "2026-03-01"

	// https://docs.lightfield.app/using-the-api/list-endpoints#pagination
	maxPageSize = 25
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		return newGetRequest(ctx, params.NextPage.String())
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("limit", strconv.Itoa(min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)))
	url.WithQueryParam("offset", "0")

	return newGetRequest(ctx, url.String())
}

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
		makeFilterFunc(request.URL),
		readhelper.MakeMarshaledDataFuncWithId(flattenRecord, readhelper.NewIdField("id")),
		params.Fields,
	)
}

func newGetRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(versionHeader, versionValue)

	return req, nil
}
