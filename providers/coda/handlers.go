package coda

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/amp-labs/connectors/providers/coda/metadata"
)

// Docs write API:
//   - create: POST /docs, body DocCreate, 201 with the created doc.
//     https://coda.io/developers/apis/v1#operation/createDoc
//   - update: PATCH /docs/{docId}, body DocUpdate (title, iconName), 200 with an empty object.
//     https://coda.io/developers/apis/v1#operation/updateDoc
func (c *Connector) buildWriteRequest(ctx context.Context, params common.WriteParams) (*http.Request, error) {
	path, err := metadata.Schemas.FindURLPath(c.ProviderContext.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	method := http.MethodPost
	segments := []string{apiVersion, path}

	if params.IsUpdate() {
		method = http.MethodPatch
		segments = append(segments, params.RecordId)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, segments...)
	if err != nil {
		return nil, err
	}

	record, err := common.RecordDataToMap(params.RecordData)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *Connector) parseWriteResponse(
	_ context.Context,
	params common.WriteParams,
	_ *http.Request,
	response *common.JSONHTTPResponse,
) (*common.WriteResult, error) {
	body, ok := response.Body()
	if !ok {
		return &common.WriteResult{Success: true, RecordId: params.RecordId}, nil
	}

	// Update responds with `{}`, so the id falls back to the record that was updated.
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
