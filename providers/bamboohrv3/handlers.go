package bamboohrv3

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers/bamboohrv3/metadata"
)

const (
	dateLayout       = "2006-01-02"
	employeesPageSize = 250
	schedulesPageSize = int64(100)
)

func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	path, err := metadata.Schemas.LookupURLPath(c.Module(), params.ObjectName)
	if err != nil {
		return nil, err
	}

	u, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	switch params.ObjectName {
	case objectNameEmployees:
		u.WithQueryParam("page[limit]", strconv.Itoa(employeesPageSize))
		if params.NextPage != "" {
			u.WithQueryParam("page[after]", params.NextPage.String())
		}
	case objectNameApplications:
		if params.NextPage != "" {
			u.WithQueryParam("page", params.NextPage.String())
		}
	case objectNameRequests:
		start, end := dateWindowFor(params)
		u.WithQueryParam("start", start)
		u.WithQueryParam("end", end)
	case objectNameTimesheetEntries:
		start, end := timesheetDateWindowFor(params)
		u.WithQueryParam("start", start)
		u.WithQueryParam("end", end)
	case objectNameSchedules:
		u.WithQueryParam("pageSize", strconv.FormatInt(schedulesPageSize, 10))
		if params.NextPage != "" {
			u.WithQueryParam("page", params.NextPage.String())
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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

	return common.ParseResult(
		response,
		common.ExtractRecordsFromPath(responseKey),
		makeNextPage(params.ObjectName),
		readhelper.MakeGetMarshaledDataWithId(idFieldFor(params.ObjectName)),
		params.Fields,
	)
}

// idFieldFor returns the ID field query for a given object.
// Employees use "employeeId" (a string); all other objects use "id".
func idFieldFor(objectName string) readhelper.IdFieldQuery {
	if objectName == objectNameEmployees {
		return readhelper.NewIdField("employeeId")
	}

	return readhelper.NewIdField("id")
}

// dateWindowFor returns YYYY-MM-DD start/end for date-bounded endpoints.
// Uses Since/Until when set; defaults to the last year when not set.
func dateWindowFor(params common.ReadParams) (start, end string) {
	now := time.Now()

	if params.Until.IsZero() {
		end = now.Format(dateLayout)
	} else {
		end = params.Until.Format(dateLayout)
	}

	if params.Since.IsZero() {
		start = now.AddDate(-1, 0, 0).Format(dateLayout)
	} else {
		start = params.Since.Format(dateLayout)
	}

	return start, end
}

// timesheetDateWindowFor returns YYYY-MM-DD start/end for timesheet entries.
// BambooHR enforces a maximum 365-day window; this clips the range if needed.
// Both dates must fall within the last 365 days from today.
func timesheetDateWindowFor(params common.ReadParams) (start, end string) {
	now := time.Now()
	maxStart := now.AddDate(0, 0, -364)

	var startDate, endDate time.Time

	if params.Until.IsZero() {
		endDate = now
	} else {
		endDate = params.Until
	}

	if params.Since.IsZero() {
		startDate = maxStart
	} else {
		startDate = params.Since
	}

	// Enforce max 365-day range by clamping startDate forward.
	if endDate.Sub(startDate).Hours() > 364*24 {
		startDate = endDate.AddDate(0, 0, -364)
	}

	return startDate.Format(dateLayout), endDate.Format(dateLayout)
}
