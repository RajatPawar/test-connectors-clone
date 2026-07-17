package sagehr

import (
	"context"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

const (
	dataField = "data"
	pageQuery = "page"
	fromQuery = "from"
	toQuery   = "to"

	// dateFormat matches the `YYYY-MM-DD` shape used by leave-management/requests'
	// `from`/`to` filters, per the API reference examples (e.g. "2018-05-20").
	dateFormat = "2006-01-02"

	// objectLeaveManagementRequests is the only object documented to support
	// time-based filtering (via `from`/`to`, see docs/openapi_spec.json).
	// The API additionally requires the `to`-`from` window to be under 65 days;
	// the connector does not chunk longer requested windows automatically.
	objectLeaveManagementRequests = "leave-management/requests"
)

// readResponse mirrors the `{"data": [...], "meta": {...}}` envelope shared by
// every Sage HR list endpoint in docs/openapi_spec.json. `meta` is omitted by
// endpoints that don't paginate (e.g. documents/categories).
type readResponse struct {
	Data []map[string]any `json:"data"`
	Meta map[string]any   `json:"meta"`
}

func (c *Connector) buildSingleObjectMetadataRequest(ctx context.Context, objectName string) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, objectName)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

func (c *Connector) parseSingleObjectMetadataResponse(
	ctx context.Context,
	objectName string,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ObjectMetadata, error) {
	objectMetadata := common.ObjectMetadata{
		FieldsMap:   make(map[string]string),
		DisplayName: naming.CapitalizeFirstLetterEveryWord(objectName),
	}

	resp, err := common.UnmarshalJSON[readResponse](response)
	if err != nil {
		return nil, common.ErrFailedToUnmarshalBody
	}

	if len(resp.Data) == 0 {
		return nil, common.ErrMissingExpectedValues
	}

	for fld := range resp.Data[0] {
		// TODO fix deprecated
		objectMetadata.FieldsMap[fld] = fld // nolint:staticcheck
	}

	return &objectMetadata, nil
}

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, params.ObjectName)
	if err != nil {
		return nil, err
	}

	if params.NextPage != "" {
		url.WithQueryParam(pageQuery, params.NextPage.String())
	}

	if params.ObjectName == objectLeaveManagementRequests {
		if !params.Since.IsZero() {
			url.WithQueryParam(fromQuery, params.Since.Format(dateFormat))
		}

		if !params.Until.IsZero() {
			url.WithQueryParam(toQuery, params.Until.Format(dateFormat))
		}
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResult(
		response,
		common.ExtractRecordsFromPath(dataField),
		nextRecordsURL,
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}
