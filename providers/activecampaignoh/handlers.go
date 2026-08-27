package activecampaignoh

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	if err := params.ValidateParams(true); err != nil {
		return nil, err
	}

	if params.NextPage != "" {
		return http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, "api", "3", params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("limit", strconv.Itoa(pageSize))
	url.WithQueryParam("offset", "0")

	if nativeIncrementalObjects.Has(params.ObjectName) {
		if !params.Since.IsZero() {
			url.WithQueryParam("filters[updated_after]", datautils.Time.FormatRFC3339inUTC(params.Since))
		}

		if !params.Until.IsZero() {
			url.WithQueryParam("filters[updated_before]", datautils.Time.FormatRFC3339inUTC(params.Until))
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
	reqURL, err := urlbuilder.FromRawURL(request.URL)
	if err != nil {
		return nil, err
	}

	nextPage := makeNextPageFunc(reqURL)

	if nativeIncrementalObjects.Has(params.ObjectName) {
		// The provider already applied filters[updated_after]/[updated_before];
		// no connector-side filtering needed.
		return common.ParseResult(
			response,
			common.ExtractOptionalRecordsFromPath(recordsPath(params.ObjectName)),
			nextPage,
			readhelper.MakeGetMarshaledDataWithId(idField()),
			params.Fields,
		)
	}

	return common.ParseResultFiltered(
		params,
		response,
		common.MakeRecordsFunc(recordsPath(params.ObjectName)),
		makeRecordsFilterFunc(params.ObjectName, nextPage),
		readhelper.MakeMarshaledDataFuncWithId(nil, idField()),
		params.Fields,
	)
}

// buildSingleObjectMetadataRequest and parseSingleObjectMetadataResponse are
// the fallback schema source (Priority 3: response sampling) for objects
// absent from the embedded schemas.json — currently "contacts" and
// "accounts". See connector.go.
func (c *Connector) buildSingleObjectMetadataRequest(ctx context.Context, objectName string) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, "api", "3", objectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("limit", "1")

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

func (c *Connector) parseSingleObjectMetadataResponse(
	ctx context.Context,
	objectName string,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ObjectMetadata, error) {
	body, ok := response.Body()
	if !ok {
		return nil, common.ErrFailedToUnmarshalBody
	}

	records, err := common.ExtractOptionalRecordsFromPath(recordsPath(objectName))(body)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("%w: could not find a record to sample fields from", common.ErrMissingExpectedValues)
	}

	fields := make(common.FieldsMetadata)

	for field, value := range records[0] {
		fields[field] = common.FieldMetadata{
			DisplayName:  field,
			ValueType:    common.InferValueTypeFromData(value),
			ProviderType: "", // Not exposed by response sampling; no describe endpoint for this object.
		}
	}

	return common.NewObjectMetadata(objectName, fields), nil
}
