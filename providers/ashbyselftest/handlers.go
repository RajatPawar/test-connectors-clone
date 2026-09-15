package ashbyselftest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers/ashbyselftest/metadata"
)

const (
	pageSizeKey     = "cursor"
	pageLimitKey    = "limit"
	defaultPageSize = "100"
)

// buildReadRequest builds a POST request against the object's `.list` RPC
// endpoint. Every Ashby list endpoint is a POST with a JSON body — even
// though it "reads" data — per https://developers.ashbyhq.com/docs/introduction.
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	path, err := metadata.Schemas.LookupURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(buildRequestBody(params))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json; version=1")

	return req, nil
}

// buildRequestBody constructs the JSON body for an object's `.list` call.
//
// Ashby has no updated_at/updated_since query filter on any list endpoint —
// only an opaque syncToken (which requires persisting provider-issued state
// across syncs, not exposed by ReadParams) and, on some objects, a
// createdAfter (creation-time) bound that would miss updates to older
// records. Per CLAUDE.md ("always filter by updated_at, never created_at"),
// Since/Until are therefore applied client-side against each record's own
// updatedAt field in parse.go, and are intentionally NOT sent as request
// params here. See README.md and connie_notes.md for the full rationale.
func buildRequestBody(params common.ReadParams) map[string]any {
	body := make(map[string]any)

	if !supportsCursorPagination.Has(params.ObjectName) {
		// jobPosting.list has no cursor/limit params at all — it always
		// returns every posting in a single response. Ask for unpublished
		// (draft) postings too, so sync coverage matches job.list's Draft
		// handling instead of silently omitting drafts.
		if params.ObjectName == objectJobPosting {
			body["includeUnpublishedJobPostings"] = true
		}

		return body
	}

	body[pageLimitKey] = readhelper.PageSizeWithDefaultStr(params, defaultPageSize)

	if params.NextPage != "" {
		body[pageSizeKey] = params.NextPage.String()
	}

	return body
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	responseKey := metadata.Schemas.LookupArrayFieldName(c.Module(), params.ObjectName)

	return common.ParseResultFiltered(
		params,
		response,
		common.MakeRecordsFunc(responseKey),
		filterByUpdatedAt(params.ObjectName),
		readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField("id")),
		params.Fields,
	)
}
