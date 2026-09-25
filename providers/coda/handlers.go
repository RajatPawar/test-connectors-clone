package coda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/amp-labs/connectors/providers/coda/metadata"
)

const apiVersion = "v1"

// buildWriteRequest creates a doc with POST /v1/docs and updates one with PATCH /v1/docs/{docId}.
// RecordData is sent as-is: create accepts DocCreate (title, folderId, timezone, sourceDoc,
// initialPage) and update accepts DocUpdate (title, iconName).
func (c *Connector) buildWriteRequest(ctx context.Context, params common.WriteParams) (*http.Request, error) {
	path, err := metadata.Schemas.FindURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	method := http.MethodPost
	parts := []string{apiVersion, path}

	if params.IsUpdate() {
		method = http.MethodPatch
		parts = append(parts, params.RecordId)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, parts...)
	if err != nil {
		return nil, err
	}

	record, err := params.GetRecord()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// parseWriteResponse reads the created Doc (201) or the empty DocUpdateResult (200).
// The create response also carries a requestId for the async mutation; it is not the record id.
func (c *Connector) parseWriteResponse(
	ctx context.Context,
	params common.WriteParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.WriteResult, error) {
	body, ok := response.Body()
	if !ok {
		return &common.WriteResult{
			Success:  true,
			RecordId: params.RecordId,
		}, nil
	}

	recordID, err := jsonquery.New(body).TextWithDefault("id", params.RecordId)
	if err != nil {
		return nil, err
	}

	data, err := jsonquery.Convertor.ObjectToMap(body)
	if err != nil {
		return nil, err
	}

	return &common.WriteResult{
		Success:  true,
		RecordId: recordID,
		Data:     data,
	}, nil
}
