package otterai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
)

// idField is used for every object: every documented object (conversations,
// channels, channel members, workspace) exposes a flat top-level "id".
var idField = readhelper.NewIdField("id") // nolint:gochecknoglobals

func (c *Connector) buildSingleObjectMetadataRequest(ctx context.Context, objectName string) (*http.Request, error) {
	switch objectName {
	case objectNameConversations:
		url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectNameConversations)
		if err != nil {
			return nil, err
		}

		// limit is documented for GET /conversations only (min 1, max 100).
		url.WithQueryParam("limit", "1")

		return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	case objectNameChannels:
		return c.buildChannelsListRequest(ctx)
	case objectNameChannelsMembers:
		return c.buildFirstChannelMembersRequest(ctx)
	case objectNameWorkspace:
		return c.buildWorkspaceRequest(ctx)
	default:
		return nil, common.ErrOperationNotSupportedForObject
	}
}

func (c *Connector) parseSingleObjectMetadataResponse(
	ctx context.Context,
	objectName string,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ObjectMetadata, error) {
	objectMetadata := common.ObjectMetadata{
		Fields:      make(map[string]common.FieldMetadata),
		DisplayName: naming.CapitalizeFirstLetterEveryWord(objectName),
	}

	body, ok := response.Body()
	if !ok {
		return nil, common.ErrMissingExpectedValues
	}

	var (
		firstRecord map[string]any
		err         error
	)

	if objectName == objectNameWorkspace {
		// GET /workspace wraps a single object under "data", not an array.
		obj, objErr := jsonquery.New(body).ObjectRequired("data")
		if objErr != nil {
			return nil, objErr
		}

		firstRecord, err = jsonquery.Convertor.ObjectToMap(obj)
	} else {
		records, arrErr := jsonquery.New(body).ArrayOptional("data")
		if arrErr != nil {
			return nil, arrErr
		}

		if len(records) == 0 {
			return nil, fmt.Errorf("%w: could not find a record to sample fields from", common.ErrMissingExpectedValues)
		}

		firstRecord, err = jsonquery.Convertor.ObjectToMap(records[0])
	}

	if err != nil {
		return nil, err
	}

	for field, value := range firstRecord {
		objectMetadata.Fields[field] = common.FieldMetadata{
			DisplayName:  field,
			ValueType:    inferValueTypeFromData(value),
			ProviderType: "",
			Values:       nil,
		}
	}

	return &objectMetadata, nil
}

func inferValueTypeFromData(value any) common.ValueType {
	switch value.(type) {
	case string:
		return common.ValueTypeString
	case float64:
		return common.ValueTypeFloat
	case bool:
		return common.ValueTypeBoolean
	default:
		return common.ValueTypeOther
	}
}

// buildChannelsListRequest builds GET /v1/channels. No pagination or filter
// query params are documented for this endpoint, so none are sent.
func (c *Connector) buildChannelsListRequest(ctx context.Context) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectNameChannels)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// buildWorkspaceRequest builds GET /v1/workspace.
func (c *Connector) buildWorkspaceRequest(ctx context.Context) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectNameWorkspace)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// buildFirstChannelMembersRequest fetches the channel list and builds a
// members request for the first channel found, for metadata sampling only.
func (c *Connector) buildFirstChannelMembersRequest(ctx context.Context) (*http.Request, error) {
	channelIDs, err := c.fetchChannelIDs(ctx)
	if err != nil {
		return nil, err
	}

	if len(channelIDs) == 0 {
		return nil, fmt.Errorf("%w: no channels available to sample channel members from", common.ErrMissingExpectedValues)
	}

	return c.buildChannelMembersRequest(ctx, channelIDs[0])
}

func (c *Connector) buildChannelMembersRequest(ctx context.Context, channelID string) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectNameChannels, channelID, "members")
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// fetchChannelIDs issues an auxiliary GET /v1/channels call (outside the
// normal Read request/response cycle) and returns every channel's id. Used to
// fan out over GET /v1/channels/{id}/members, which has no bulk/list-all form.
func (c *Connector) fetchChannelIDs(ctx context.Context) ([]string, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectNameChannels)
	if err != nil {
		return nil, err
	}

	resp, err := c.JSONHTTPClient().Get(ctx, url.String())
	if err != nil {
		return nil, err
	}

	body, ok := resp.Body()
	if !ok {
		return nil, nil
	}

	channels, err := jsonquery.New(body).ArrayOptional("data")
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(channels))

	for _, channel := range channels {
		id, err := jsonquery.New(channel).StringOptional("id")
		if err != nil || id == nil {
			continue
		}

		ids = append(ids, *id)
	}

	return ids, nil
}

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	switch params.ObjectName {
	case objectNameConversations:
		return c.buildConversationsReadRequest(ctx, params)
	case objectNameChannels:
		return c.buildChannelsListRequest(ctx)
	case objectNameChannelsMembers:
		// The fan-out over every channel's members happens in
		// parseReadResponse; this request only fetches the channel list.
		return c.buildChannelsListRequest(ctx)
	case objectNameWorkspace:
		return c.buildWorkspaceRequest(ctx)
	default:
		return nil, common.ErrOperationNotSupportedForObject
	}
}

// buildConversationsReadRequest builds GET /v1/conversations.
//
// TODO: Otter's API does not document an updated_at/updated_since filter for
// any object (conversations only exposes `created_at`, a creation timestamp).
// Per the docs, Otter recommends workspace webhooks for real-time change
// delivery instead of polling. ReadParams.Since/Until are intentionally not
// applied here to avoid silently missing updates to existing conversations
// under a fake "incremental" filter — see README.md.
func (c *Connector) buildConversationsReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectNameConversations)
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("limit", readhelper.PageSizeWithDefaultStr(params, "100"))

	if params.NextPage != "" {
		url.WithQueryParam("cursor", params.NextPage.String())
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	if params.ObjectName == objectNameChannelsMembers {
		return c.parseChannelsMembersResponse(ctx, params, response)
	}

	return common.ParseResult(
		response,
		records(params.ObjectName),
		nextRecordsURL(),
		readhelper.MakeGetMarshaledDataWithId(idField),
		params.Fields,
	)
}

// parseChannelsMembersResponse turns the just-fetched channel list into a
// full member roster: GET /v1/channels/{id}/members has no bulk/list-all
// form, so every channel is queried individually and the rosters are
// concatenated. Neither GET /channels nor GET /channels/{id}/members document
// pagination, so this always completes in a single Read() call (Done: true).
func (c *Connector) parseChannelsMembersResponse(
	ctx context.Context,
	params common.ReadParams,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	body, ok := response.Body()
	if !ok {
		return &common.ReadResult{Data: []common.ReadResultRow{}, Done: true}, nil
	}

	channels, err := jsonquery.New(body).ArrayOptional("data")
	if err != nil {
		return nil, err
	}

	var allRecords []map[string]any

	for _, channel := range channels {
		channelID, err := jsonquery.New(channel).StringOptional("id")
		if err != nil || channelID == nil {
			continue
		}

		members, err := c.fetchChannelMembers(ctx, *channelID)
		if err != nil {
			return nil, err
		}

		allRecords = append(allRecords, members...)
	}

	marshaled, err := readhelper.MakeGetMarshaledDataWithId(idField)(allRecords, params.Fields.List())
	if err != nil {
		return nil, err
	}

	return &common.ReadResult{
		Rows:     int64(len(marshaled)),
		Data:     marshaled,
		NextPage: "",
		Done:     true,
	}, nil
}

func (c *Connector) fetchChannelMembers(ctx context.Context, channelID string) ([]map[string]any, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, restAPIVersion, objectNameChannels, channelID, "members")
	if err != nil {
		return nil, err
	}

	resp, err := c.JSONHTTPClient().Get(ctx, url.String())
	if err != nil {
		return nil, err
	}

	body, ok := resp.Body()
	if !ok {
		return nil, nil
	}

	members, err := jsonquery.New(body).ArrayOptional("data")
	if err != nil {
		return nil, err
	}

	return jsonquery.Convertor.ArrayToMap(members)
}
