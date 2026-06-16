package connectwise

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/internal/httpkit"
	"github.com/amp-labs/connectors/providers/connectwise/metadata"
	"github.com/spyzhov/ajson"
)

const (
	pageSizeParam = "pageSize"
	pageSize      = "1000"
	conditionsKey = "conditions"
	// lastUpdated is the conditions field for time-based incremental reads.
	// Format per docs: lastUpdated>[2016-08-20T18:04:26Z]
	lastUpdated = "lastUpdated"
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if params.NextPage != "" {
		// Link header supplies the full next-page URL; use it directly.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Add("clientId", c.clientId)

		return req, nil
	}

	urlPath, err := metadata.Schemas.LookupURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, urlPath)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(pageSizeParam, pageSize)

	if !params.Since.IsZero() {
		// ConnectWise conditions require datetime values wrapped in square brackets.
		condition := fmt.Sprintf("%s>[%s]", lastUpdated, datautils.Time.FormatRFC3339inUTC(params.Since))
		url.WithQueryParam(conditionsKey, condition)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("clientId", c.clientId)

	return req, nil
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResult(
		response,
		getRecords,
		makeNextRecordsURL(response),
		common.GetMarshaledData,
		params.Fields,
	)
}

func makeNextRecordsURL(resp *common.JSONHTTPResponse) common.NextPageFunc {
	return func(_ *ajson.Node) (string, error) {
		return httpkit.HeaderLink(resp, "next"), nil
	}
}
