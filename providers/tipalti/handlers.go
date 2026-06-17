package tipalti

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/internal/jsonquery"
)

// payees is the only object that supports time-based incremental filtering.
const payeesObject = "payees"

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiBase, apiVersion, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("pageSize", pageSize)

	if params.NextPage != "" {
		url.WithQueryParam("pageCursor", params.NextPage.String())
	}

	// Only payees supports lastChangeDateTimeUTC filtering.
	if params.ObjectName == payeesObject && !params.Since.IsZero() {
		filter := fmt.Sprintf(`lastChangeDateTimeUTC >= "%s"`, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Since))
		if !params.Until.IsZero() {
			filter += fmt.Sprintf(` && lastChangeDateTimeUTC <= "%s"`, datautils.Time.FormatRFC3339inUTCWithMilliseconds(params.Until))
		}

		url.WithQueryParam("filter", filter)
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
		records(),
		nextPageCursor(),
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}

func (c *Connector) buildSingleObjectMetadataRequest(ctx context.Context, objectName string) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiBase, apiVersion, objectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("pageSize", "1")

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

func (c *Connector) parseSingleObjectMetadataResponse(
	ctx context.Context,
	objectName string,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ObjectMetadata, error) {
	fields := make(common.FieldsMetadata)

	objectMetadata := common.ObjectMetadata{
		Fields:      fields,
		DisplayName: naming.CapitalizeFirstLetterEveryWord(objectName),
	}

	body, ok := response.Body()
	if !ok {
		return nil, common.ErrFailedToUnmarshalBody
	}

	items, err := jsonquery.New(body).ArrayOptional("items")
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return &objectMetadata, nil
	}

	firstRecord, err := jsonquery.Convertor.ObjectToMap(items[0])
	if err != nil {
		return nil, err
	}

	for field := range firstRecord {
		fields.AddFieldWithDisplayOnly(field, field)
	}

	return &objectMetadata, nil
}
