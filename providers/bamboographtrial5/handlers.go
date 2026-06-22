package bamboographtrial5

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/providers/bamboographtrial5/metadata"
)

const (
	defaultPageSize    = 100
	locationPageSize   = 500
	schedulesPageSize  = 100
	queryPageKey       = "page"
	queryPageSizeKey   = "pageSize"
	queryFilterKey     = "filter"
)

// pageSizeForObject returns the page size for paginated objects.
func pageSizeForObject(objectName string) int {
	switch objectName {
	case "api/v1/hris/org/locations":
		return locationPageSize
	case "schedules":
		return schedulesPageSize
	default:
		return 0 // not paginated
	}
}

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	path, err := metadata.Schemas.LookupURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	switch params.ObjectName {
	case "schedules":
		page := "1"
		if params.NextPage != "" {
			page = params.NextPage.String()
		}
		url.WithQueryParam(queryPageKey, page)
		url.WithQueryParam(queryPageSizeKey, strconv.Itoa(schedulesPageSize))
		if !params.Since.IsZero() {
			// schedules OData filter supports updatedAt for incremental reads.
			url.WithQueryParam(queryFilterKey, fmt.Sprintf(
				"updatedAt ge '%s'", datautils.Time.FormatRFC3339inUTC(params.Since),
			))
		}

	case "api/v1/hris/org/locations":
		// locations uses 0-indexed pages.
		page := "0"
		if params.NextPage != "" {
			page = params.NextPage.String()
		}
		url.WithQueryParam(queryPageKey, page)
		url.WithQueryParam(queryPageSizeKey, strconv.Itoa(locationPageSize))
		if !params.Since.IsZero() {
			// TODO: locations OData filter does not expose updatedAt; createdAt is used instead.
			// This means updates to existing locations will not be captured incrementally.
			url.WithQueryParam(queryFilterKey, fmt.Sprintf(
				"createdAt ge '%s'", datautils.Time.FormatRFC3339inUTC(params.Since),
			))
		}

	case "whos_out":
		// whos_out is a date-window query. Use params.Since as start when provided.
		if !params.Since.IsZero() {
			url.WithQueryParam("start", params.Since.Format("2006-01-02"))
		}
		if !params.Until.IsZero() {
			url.WithQueryParam("end", params.Until.Format("2006-01-02"))
		}
	}

	// All BambooHR endpoints require Accept: application/json.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	responseKey := metadata.Schemas.LookupArrayFieldName(c.Module(), params.ObjectName)
	ps := pageSizeForObject(params.ObjectName)

	// Determine current page for paginated objects so the next-page function can compute the next page.
	currentPage := 1
	if params.NextPage != "" {
		n, err := strconv.Atoi(params.NextPage.String())
		if err == nil {
			currentPage = n
		}
	}

	return common.ParseResult(
		response,
		common.ExtractRecordsFromPath(responseKey),
		makeNextRecordsURL(responseKey, ps, currentPage),
		readhelper.MakeGetMarshaledDataWithId(readhelper.NewIdField("id")),
		params.Fields,
	)
}
