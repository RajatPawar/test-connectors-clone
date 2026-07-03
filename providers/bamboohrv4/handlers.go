package bamboohrv4

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

// objectPaths maps an Ampersand object name to its BambooHR endpoint path,
// relative to the company base URL.
var objectPaths = map[string]string{ // nolint:gochecknoglobals
	objectNameEmployees:        "api/v1/employees",
	objectNameJobs:             "api/v1/applicant_tracking/jobs",
	objectNameApplications:     "api/v1/applicant_tracking/applications",
	objectNameRequests:         "api/v1/time_off/requests",
	objectNameSchedules:        "api/v1/scheduling/schedules",
	objectNameTimesheetEntries: "api/v1/time_tracking/timesheet_entries",
}

const dateOnlyLayout = "2006-01-02"

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	// Employees, applications, and schedules paginate via a fully-qualified
	// "next" link embedded in the response body — just follow it as-is.
	if params.NextPage != "" {
		url, err := urlbuilder.New(params.NextPage.String())
		if err != nil {
			return nil, err
		}

		return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	}

	path, ok := objectPaths[params.ObjectName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", common.ErrObjectNotSupported, params.ObjectName)
	}

	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	switch params.ObjectName {
	case objectNameEmployees:
		// Cursor pagination. GET /api/v1/employees has no updated-since filter of
		// its own — BambooHR instead offers a separate change-tracking endpoint
		// (GET /api/v1/employees/changed) that returns only changed employee IDs.
		// TODO: wire up /api/v1/employees/changed as a pre-filter once the
		// framework has a clean way to do a 2-step (ids -> batch fetch) read for
		// a single object; out of scope for this first pass.
		url.WithQueryParam("page[limit]", readhelper.PageSizeWithDefaultStr(params, "250"))

	case objectNameJobs:
		// No pagination and no time-scoping query parameters are documented for
		// this endpoint; it always returns the full list of non-deleted job
		// openings in one response.

	case objectNameApplications:
		// Only a "created after" filter is documented (newSince), not an
		// updated-since one. Used as the closest available incremental signal.
		if !params.Since.IsZero() {
			url.WithQueryParam("newSince", params.Since.UTC().Format("2006-01-02 15:04:05"))
		}

	case objectNameRequests:
		// start/end are both required by the API and bound a date-overlap
		// window (not an updated-since filter); Since/Until are mapped onto them.
		start, end := readDateWindow(params)
		url.WithQueryParam("start", start)
		url.WithQueryParam("end", end)

	case objectNameSchedules:
		url.WithQueryParam("pageSize", readhelper.PageSizeWithDefaultStr(params, "100"))

		if filter := scheduleUpdatedFilter(params); filter != "" {
			url.WithQueryParam("filter", filter)
		}

	case objectNameTimesheetEntries:
		// start/end are required and must fall within the last 365 days; they
		// scope entries by their own date, not by an updated-since timestamp.
		start, end := readDateWindow(params)
		url.WithQueryParam("start", start)
		url.WithQueryParam("end", end)
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
}

// readDateWindow maps ReadParams.Since/Until onto a YYYY-MM-DD date window,
// defaulting to the last 365 days when unset (BambooHR's own maximum lookback
// for timesheet entries; reused for requests for consistency).
func readDateWindow(params common.ReadParams) (start, end string) {
	until := params.Until
	if until.IsZero() {
		until = time.Now()
	}

	since := params.Since
	if since.IsZero() {
		since = until.AddDate(-1, 0, 0)
	}

	return since.Format(dateOnlyLayout), until.Format(dateOnlyLayout)
}

// scheduleUpdatedFilter builds an OData filter expression against the
// "updatedAt" field, the only incremental signal documented for this endpoint.
// TODO: the docs only show quoted string literal examples (e.g. name eq
// 'Default Schedule'); no example with a datetime literal is given, so it's
// unconfirmed whether "updatedAt ge '<RFC3339>'" (quoted) is the expected form
// vs. an unquoted ISO 8601 literal. Verify against a live capture.
func scheduleUpdatedFilter(params common.ReadParams) string {
	filter := ""

	if !params.Since.IsZero() {
		filter = fmt.Sprintf("updatedAt ge '%s'", datautils.Time.FormatRFC3339inUTC(params.Since))
	}

	if !params.Until.IsZero() {
		clause := fmt.Sprintf("updatedAt le '%s'", datautils.Time.FormatRFC3339inUTC(params.Until))
		if filter == "" {
			filter = clause
		} else {
			filter = filter + " and " + clause
		}
	}

	return filter
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	return common.ParseResult(
		response,
		records(params.ObjectName),
		nextRecordsURL(params.ObjectName),
		readhelper.MakeGetMarshaledDataWithId(idFieldForObject(params.ObjectName)),
		params.Fields,
	)
}

func idFieldForObject(objectName string) readhelper.IdFieldQuery {
	if objectName == objectNameEmployees {
		return readhelper.NewIdField("employeeId")
	}

	return readhelper.NewIdField("id")
}
