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
	apiVersion = "v1"

	// Required on every request; requests without it fail with the "version_header" error code.
	// https://docs.lightfield.app/using-the-api/errors#error-codes
	versionHeaderName  = "Lightfield-Version"
	versionHeaderValue = "2026-03-01"

	// The highest `limit` list methods accept.
	// https://docs.lightfield.app/using-the-api/list-endpoints#pagination
	maxPageSize = 25

	responseKeyRecords = "data"
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	path, err := metadata.Catalog.ReadPath(params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, path)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("limit", strconv.Itoa(pageSize(params)))

	if params.NextPage != "" {
		url.WithQueryParam("offset", params.NextPage.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(versionHeaderName, versionHeaderValue)

	return req, nil
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	resp *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResultFiltered(
		params,
		resp,
		common.MakeRecordsFunc(responseKeyRecords),
		makeFilterFunc(makeNextRecordsOffset(request.URL)),
		readhelper.MakeMarshaledDataFuncWithId(flattenRecord, readhelper.NewIdField("id")),
		params.Fields,
	)
}

func pageSize(params common.ReadParams) int {
	return min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)
}
