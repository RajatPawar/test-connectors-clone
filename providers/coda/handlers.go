package coda

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
)

// keyColumnsField is a reserved key inside RecordData. When present on an insert,
// its value (a list of column IDs/names) is lifted out of the cells and sent as the
// top-level `keyColumns` of the request, turning the insert into an upsert.
// See: "Insert/upsert rows" in the Coda (Superhuman Docs) API.
const keyColumnsField = "keyColumns"

// buildWriteRequest writes a row into a Coda table.
//
// The object name carries the doc + table context and is the rows collection path:
//
//	docs/{docId}/tables/{tableIdOrName}/rows
//
//   - Insert/upsert (no RecordId): POST .../rows with body {"rows":[{"cells":[...]}]}
//     (plus a top-level "keyColumns" when RecordData supplies one — that makes it an upsert).
//   - Update (RecordId is the rowIdOrName): PUT .../rows/{rowIdOrName} with body {"row":{"cells":[...]}}.
//
// RecordData is a flat column->value map; each entry becomes a Coda cell
// {"column": <id/name>, "value": <value>}. Column IDs are preferred over names
// (names are fragile), per the provider docs.
func (c *Connector) buildWriteRequest(ctx context.Context, params common.WriteParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, params.ObjectName)
	if err != nil {
		return nil, err
	}

	data, err := common.RecordDataToMap(params.RecordData)
	if err != nil {
		return nil, err
	}

	method := http.MethodPost

	var body any

	if params.RecordId != "" {
		// Update a single existing row.
		url.AddPath(params.RecordId)

		method = http.MethodPut
		body = map[string]any{
			"row": map[string]any{
				"cells": buildCells(data),
			},
		}
	} else {
		// Insert or upsert one row.
		insert := map[string]any{
			"rows": []map[string]any{
				{"cells": buildCells(data)},
			},
		}

		if keyColumns, ok := data[keyColumnsField]; ok {
			insert[keyColumnsField] = keyColumns
		}

		body = insert
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, method, url.String(), bytes.NewReader(jsonData))
}

// buildCells converts a flat column->value map into Coda's cell list. Keys are
// sorted so the output is deterministic.
func buildCells(data map[string]any) []map[string]any {
	keys := make([]string, 0, len(data))

	for key := range data {
		if key == keyColumnsField {
			continue
		}

		keys = append(keys, key)
	}

	sort.Strings(keys)

	cells := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		cells = append(cells, map[string]any{
			"column": key,
			"value":  data[key],
		})
	}

	return cells
}

// parseWriteResponse handles Coda's asynchronous mutation response. Row writes
// return HTTP 202 with a `requestId` (the edit is queued, not yet applied); an
// insert also returns `addedRowIds`. The requestId is preserved in Data so the
// caller can poll GET /mutationStatus/{requestId} to confirm completion.
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

	data, err := jsonquery.Convertor.ObjectToMap(body)
	if err != nil {
		return nil, err
	}

	recordID := params.RecordId
	if recordID == "" {
		// Insert/upsert: the API returns the IDs of the rows it added.
		if ids, err := jsonquery.New(body).ArrayOptional("addedRowIds"); err == nil && len(ids) > 0 {
			if id, err := ids[0].GetString(); err == nil {
				recordID = id
			}
		}
	}

	return &common.WriteResult{
		Success:  true,
		RecordId: recordID,
		Data:     data,
	}, nil
}
