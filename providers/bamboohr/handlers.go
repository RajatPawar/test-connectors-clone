package bamboohr

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

const (
	defaultEmployeesPageSize = "250" // API default; max 2500.
	defaultReportsPageSize   = "500" // API default; max 1000.
	timeOffDefaultWindow     = 365 * 24 * time.Hour
)

// changedEmployeesResponse is the "Get Changed Employee IDs" response shape.
// https://documentation.bamboohr.com/reference (operationId: get-changed-employee-ids)
type changedEmployeesResponse struct {
	Latest    string `json:"latest"`
	Employees map[string]struct {
		ID          string `json:"id"`
		Action      string `json:"action"` // "Inserted", "Updated", or "Deleted"
		LastChanged string `json:"lastChanged"`
	} `json:"employees"`
}

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	// Employees and custom-reports pagination continuations carry the full next-page URL
	// (see nextPageFromLinksHref / nextPageFromPaginationNextPage in parse.go) — every other
	// object is single-page, so params.NextPage is always empty for them.
	if params.NextPage != "" {
		return http.NewRequestWithContext(ctx, http.MethodGet, params.NextPage.String(), nil)
	}

	switch {
	case params.ObjectName == objectNameEmployees:
		return c.buildEmployeesReadRequest(ctx, params)
	case params.ObjectName == objectNameEmployeeDirectory:
		return c.buildDirectoryReadRequest(ctx, params)
	case params.ObjectName == objectNameCustomReports:
		return c.buildCustomReportsReadRequest(ctx, params)
	case params.ObjectName == objectNameTimeOffRequests:
		return c.buildTimeOffRequestsReadRequest(ctx, params)
	case isTableObject(params.ObjectName):
		return c.buildTableReadRequest(ctx, params)
	default:
		return nil, fmt.Errorf("%w: %v", common.ErrObjectNotSupported, params.ObjectName)
	}
}

// buildEmployeesReadRequest handles GET /api/v1/employees. When params.Since is set, it first
// resolves the set of changed employee IDs via "Get Changed Employee IDs" (GET
// /api/v1/employees/changed), then filters the main List Employees call down to just those IDs —
// List Employees itself has no last-changed filter (confirmed against
// GetEmployeesFilterRequestObject in the OpenAPI spec), so this two-step lookup is how
// incremental sync is achieved for this object.
func (c *Connector) buildEmployeesReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, "api", "v1", "employees")
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("page[limit]", readhelper.PageSizeWithDefaultStr(params, defaultEmployeesPageSize))

	if len(params.Fields) > 0 {
		url.WithQueryParam("fields", strings.Join(params.Fields.List(), ","))
	}

	if !params.Since.IsZero() {
		ids, err := c.fetchChangedEmployeeIDs(ctx, params.Since)
		if err != nil {
			return nil, err
		}

		if len(ids) == 0 {
			// No employees changed in this window. Use a filter that is guaranteed to match
			// nothing rather than skipping the request, so callers get a normal empty result.
			ids = []string{"0"}
		}

		url.WithQueryParam("filter[ids]", strings.Join(ids, ","))
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// fetchChangedEmployeeIDs calls GET /api/v1/employees/changed?since=<since> and returns the IDs
// of employees that were Inserted or Updated (Deleted employees can no longer be fetched via
// List Employees, so they are excluded).
func (c *Connector) fetchChangedEmployeeIDs(ctx context.Context, since time.Time) ([]string, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, "api", "v1", "employees", "changed")
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("since", since.UTC().Format(time.RFC3339))

	resp, err := c.JSONHTTPClient().Get(ctx, url.String())
	if err != nil {
		return nil, err
	}

	changed, err := common.UnmarshalJSON[changedEmployeesResponse](resp)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(changed.Employees))

	for id, change := range changed.Employees {
		if change.Action == "Deleted" {
			continue
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// buildDirectoryReadRequest handles GET /api/v1/employees/directory. This endpoint has no
// pagination and no time-based filter, so Since/Until are not applied; this is a documented
// limitation (see README.md).
func (c *Connector) buildDirectoryReadRequest(ctx context.Context, _ common.ReadParams) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, "api", "v1", "employees", "directory")
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// buildCustomReportsReadRequest handles GET /api/v1/custom-reports (List Reports). The endpoint
// only exposes report id/name, not a last-modified timestamp, so Since/Until are not applied;
// this is a documented limitation (see README.md).
func (c *Connector) buildCustomReportsReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) { //nolint:lll
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, "api", "v1", "custom-reports")
	if err != nil {
		return nil, err
	}

	url.WithQueryParam("page", "1")
	url.WithQueryParam("page_size", readhelper.PageSizeWithDefaultStr(params, defaultReportsPageSize))

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// buildTimeOffRequestsReadRequest handles GET /api/v1/time_off/requests. Unlike most objects,
// `start` and `end` are REQUIRED by BambooHR, so Since/Until are given a default window
// (1 year back / 1 year forward) when the caller omits them.
func (c *Connector) buildTimeOffRequestsReadRequest(
	ctx context.Context, params common.ReadParams,
) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, "api", "v1", "time_off", "requests")
	if err != nil {
		return nil, err
	}

	since := params.Since
	if since.IsZero() {
		since = time.Now().Add(-timeOffDefaultWindow)
	}

	until := params.Until
	if until.IsZero() {
		until = time.Now().Add(timeOffDefaultWindow)
	}

	url.WithQueryParam("start", since.Format(time.DateOnly))
	url.WithQueryParam("end", until.Format(time.DateOnly))

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// buildTableReadRequest handles the employee-table objects (jobInfo, compensation, ...). Without
// Since it reads every row for every employee via the "all" sentinel
// (GET /api/v1/employees/all/tables/{table}). With Since it reads only rows for employees changed
// since that time via "Get Changed Employee Table Data"
// (GET /api/v1/employees/changed/tables/{table}?since=...) — see changedTableRecords in parse.go
// for how that differently-shaped response is flattened back to rows.
func (c *Connector) buildTableReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	var url *urlbuilder.URL

	var err error

	if !params.Since.IsZero() {
		url, err = urlbuilder.New(c.ProviderInfo().BaseURL, "api", "v1", "employees", "changed", "tables", params.ObjectName) //nolint:lll
		if err != nil {
			return nil, err
		}

		url.WithQueryParam("since", params.Since.UTC().Format(time.RFC3339))
	} else {
		url, err = urlbuilder.New(c.ProviderInfo().BaseURL, "api", "v1", "employees", "all", "tables", params.ObjectName)
		if err != nil {
			return nil, err
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
	usingChangedTablesEndpoint := isTableObject(params.ObjectName) && !params.Since.IsZero()

	return common.ParseResult(
		response,
		recordsFunc(params.ObjectName, usingChangedTablesEndpoint),
		nextPageFunc(params.ObjectName),
		readhelper.MakeGetMarshaledDataWithId(idFieldForObject(params.ObjectName)),
		params.Fields,
	)
}
