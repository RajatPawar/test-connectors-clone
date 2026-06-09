package connectWise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/internal/jsonquery"
)

const (
	pageSizeKey  = "pageSize"
	conditionKey = "conditions"
	clientIDHdr  = "clientId"
	pageSize     = "1000"
	samplePageSz = "1"

	// lastUpdated is the standard ConnectWise audit field usable in `conditions`
	// for incremental reads across the core entities.
	lastUpdatedField = "lastUpdated"
)

// withHeaders applies the headers ConnectWise requires on every request.
// The clientId GUID is mandatory; the version Accept header pins the model.
func (c *Connector) withHeaders(req *http.Request) *http.Request {
	if c.clientId != "" {
		req.Header.Set(clientIDHdr, c.clientId)
	}

	req.Header.Set("Accept", "application/json")

	return req
}

func (c *Connector) buildSingleObjectMetadataRequest(
	ctx context.Context, objectName string,
) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, objectName)
	if err != nil {
		return nil, err
	}

	// Sample a single record to read the field names off of it.
	url.WithQueryParam(pageSizeKey, samplePageSz)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.withHeaders(req), nil
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

	// ConnectWise list responses are a top-level JSON array.
	records, err := jsonquery.New(body).ArrayRequired("")
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("%w: could not find a record to sample fields from", common.ErrMissingExpectedValues)
	}

	firstRecord, err := jsonquery.Convertor.ObjectToMap(records[0])
	if err != nil {
		return nil, err
	}

	for fld := range firstRecord {
		objectMetadata.FieldsMap[fld] = fld // nolint:staticcheck
	}

	return &objectMetadata, nil
}

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	url, err := c.buildReadURL(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.withHeaders(req), nil
}

func (c *Connector) buildReadURL(params common.ReadParams) (*urlbuilder.URL, error) {
	// Navigable pagination returns a fully-formed `next` URL in the Link header.
	if params.NextPage != "" {
		return urlbuilder.New(params.NextPage.String())
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, params.ObjectName)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam(pageSizeKey, pageSize)

	// Incremental read via the `conditions` query parameter. Datetimes are
	// wrapped in square brackets per ConnectWise condition syntax.
	if condition := buildTimeCondition(params); condition != "" {
		url.WithQueryParam(conditionKey, condition)
	}

	return url, nil
}

// buildTimeCondition assembles a ConnectWise `conditions` clause that bounds the
// result set by lastUpdated using the requested Since / Until window.
func buildTimeCondition(params common.ReadParams) string {
	var clauses []string

	if !params.Since.IsZero() {
		clauses = append(clauses, fmt.Sprintf("%s >= [%s]",
			lastUpdatedField, datautils.Time.FormatRFC3339inUTC(params.Since)))
	}

	if !params.Until.IsZero() {
		clauses = append(clauses, fmt.Sprintf("%s < [%s]",
			lastUpdatedField, datautils.Time.FormatRFC3339inUTC(params.Until)))
	}

	switch len(clauses) {
	case 0:
		return ""
	case 1:
		return clauses[0]
	default:
		return clauses[0] + " and " + clauses[1]
	}
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
		nextRecordsURL(response.Headers),
		common.GetMarshaledData,
		params.Fields,
	)
}

func (c *Connector) buildWriteRequest(ctx context.Context, params common.WriteParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, params.ObjectName)
	if err != nil {
		return nil, err
	}

	// POST creates a new record; PUT replaces an existing one with the same
	// flat shape that reads return.
	method := http.MethodPost
	if params.RecordId != "" {
		url.AddPath(params.RecordId)

		method = http.MethodPut
	}

	jsonData, err := json.Marshal(params.RecordData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url.String(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return c.withHeaders(req), nil
}

func (c *Connector) parseWriteResponse(
	ctx context.Context,
	params common.WriteParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.WriteResult, error) {
	body, ok := response.Body()
	if !ok {
		return &common.WriteResult{Success: true}, nil
	}

	recordId, err := jsonquery.New(body).IntegerOptional("id")
	if err != nil {
		return nil, err
	}

	data, err := jsonquery.Convertor.ObjectToMap(body)
	if err != nil {
		return nil, err
	}

	result := &common.WriteResult{
		Success: true,
		Data:    data,
	}

	if recordId != nil {
		result.RecordId = strconv.FormatInt(*recordId, 10)
	}

	return result, nil
}
