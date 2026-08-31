package activecampaign

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

const (
	limitQuery       = "limit"
	offsetQuery      = "offset"
	pageSize         = 100
	metadataPageSize = "1"
)

// objectsWithUpdatedFilter are the objects that support the v3
// filters[updated_after]/filters[updated_before] incremental filters.
var objectsWithUpdatedFilter = map[string]bool{ //nolint:gochecknoglobals
	"contacts": true,
	"deals":    true,
}

func (c *Connector) buildSingleObjectMetadataRequest(
	ctx context.Context,
	objectName string,
) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, objectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(limitQuery, metadataPageSize)

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

	body, ok := response.Body()
	if !ok {
		return nil, common.ErrEmptyJSONHTTPResponse
	}

	records, err := recordsFromNode(objectName)(body)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("%w: could not find a record to sample fields from", common.ErrMissingExpectedValues)
	}

	for field := range records[0] {
		objectMetadata.FieldsMap[field] = field //nolint:staticcheck
	}

	return &objectMetadata, nil
}

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(limitQuery, strconv.Itoa(pageSize))

	if params.NextPage != "" {
		url.WithQueryParam(offsetQuery, params.NextPage.String())
	}

	if objectsWithUpdatedFilter[params.ObjectName] {
		if !params.Since.IsZero() {
			url.WithQueryParam("filters[updated_after]", params.Since.Format(time.RFC3339))
		}

		if !params.Until.IsZero() {
			url.WithQueryParam("filters[updated_before]", params.Until.Format(time.RFC3339))
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
	// v3 offset pagination is stateless; the current offset is not echoed in the
	// response, so derive it from the request URL to compute the next page.
	currentOffset := 0
	if raw := request.URL.Query().Get(offsetQuery); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			currentOffset = parsed
		}
	}

	return common.ParseResult(
		response,
		recordsFromNode(params.ObjectName),
		nextRecordsURL(currentOffset),
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}
