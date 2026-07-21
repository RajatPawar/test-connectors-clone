package sagehr

import (
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// paginationStyle captures the (non-uniform) pagination conventions observed
// across Sage HR endpoints. See providers/sagehr/README.md.
type paginationStyle int

const (
	// paginationNone: the endpoint returns the full `data` array in one shot,
	// no `meta` envelope at all (e.g. leave-management/policies).
	paginationNone paginationStyle = iota
	// paginationPage: `?page=N`, response has `meta.next_page` (int or null).
	// This is the default for core endpoints (teams, employees, positions, ...).
	paginationPage
	// paginationPagePerPage: `?page=N&per_page=M`, same `meta` envelope.
	// Used by recruitment endpoints (default 30, max 100).
	paginationPagePerPage
)

// fanOutSpec describes a child collection that is only reachable per-parent
// (e.g. /employees/{id}/compensations). parentObject must be a key in
// objectSpecs whose pagination/path describe the parent LIST endpoint used to
// discover parent ids.
type fanOutSpec struct {
	parentObject   string
	childSuffix    string
	pagination     paginationStyle
	perPageDefault string
}

type objectSpec struct {
	// path is the URL path (relative to BaseURL) for a top-level object, or —
	// when fanOut is set — the parent's own list path (fanOut.parentObject
	// duplicates this lookup; path is unused directly for fan-out objects).
	path           string
	pagination     paginationStyle
	perPageDefault string
	// dateWindowed is true only for leave-management/requests, whose `from`/
	// `to` window must stay under 65 days (docs: "must be less than 65").
	dateWindowed bool
	fanOut       *fanOutSpec
}

// maxLeaveRequestWindowDays is kept safely under the documented 65-day limit
// for leave-management/requests' from/to window.
const maxLeaveRequestWindowDays = 60

//nolint:gochecknoglobals
var objectSpecs = map[string]objectSpec{
	objectTeams:               {path: "teams", pagination: paginationPage},
	objectEmployees:           {path: "employees", pagination: paginationPage},
	objectPositions:           {path: "positions", pagination: paginationPage},
	objectTerminationReasons:  {path: "termination-reasons", pagination: paginationPage},
	objectTerminatedEmployees: {path: "terminated-employees", pagination: paginationPage},
	objectRecruitmentPositions: {
		path: "recruitment/positions", pagination: paginationPagePerPage, perPageDefault: "30",
	},
	objectLeavePolicies:      {path: "leave-management/policies", pagination: paginationNone},
	objectDocumentCategories: {path: "documents/categories", pagination: paginationNone},
	objectLeaveRequests: {
		path: "leave-management/requests", pagination: paginationPage, dateWindowed: true,
	},
	objectEmployeeCompensations: {
		fanOut: &fanOutSpec{
			parentObject: objectEmployees,
			childSuffix:  "compensations",
			// The docs' example response for this endpoint includes a `meta`
			// block identical to other paginated endpoints, but no `page`
			// query parameter is documented for it (likely a doc omission —
			// every other object uses the same `page` convention). We follow
			// meta.next_page defensively via the shared `page` param; if the
			// endpoint is actually single-page, meta.next_page will simply
			// always be null and this is a no-op.
			pagination: paginationPage,
		},
	},
	objectEmployeeCustomFields: {
		fanOut: &fanOutSpec{
			parentObject: objectEmployees,
			childSuffix:  "custom-fields",
			pagination:   paginationNone,
		},
	},
	objectEmployeeLeaveBalances: {
		fanOut: &fanOutSpec{
			parentObject: objectEmployees,
			childSuffix:  "leave-management/balances",
			pagination:   paginationNone,
		},
	},
	objectRecruitmentApplicants: {
		fanOut: &fanOutSpec{
			parentObject:   objectRecruitmentPositions,
			childSuffix:    "applicants",
			pagination:     paginationPagePerPage,
			perPageDefault: "30",
		},
	},
}

// recordsFunc extracts the `data` array present on every Sage HR list response.
func recordsFunc(node *ajson.Node) ([]map[string]any, error) {
	return common.ExtractOptionalRecordsFromPath("data")(node)
}

func idField() readhelper.IdFieldQuery {
	return readhelper.NewIdField("id")
}

func marshalFunc() common.MarshalFunc {
	return readhelper.MakeGetMarshaledDataWithId(idField())
}

func applyPaginationParams(url *urlbuilder.URL, style paginationStyle, perPageDefault string, params common.ReadParams) {
	switch style {
	case paginationNone:
		return
	case paginationPage:
		url.WithQueryParam("page", "1")
	case paginationPagePerPage:
		url.WithQueryParam("page", "1")
		url.WithQueryParam("per_page", readhelper.PageSizeWithDefaultStr(params, perPageDefault))
	}
}

// applyEmployeeHistoryParams always requests the optional history collections
// on /employees and /employees/{id}. These are boolean-gated in the API
// (absent unless explicitly requested) but cost nothing extra to fetch and
// enrich the record — see CLAUDE.md's field-completeness guidance.
func applyEmployeeHistoryParams(url *urlbuilder.URL) {
	url.WithQueryParam("team_history", "true")
	url.WithQueryParam("employment_status_history", "true")
	url.WithQueryParam("position_history", "true")
}

// defaultLeaveRequestLookbackYears bounds how far back a full sync (Since
// zero — e.g. the very first read of an installation) reaches for
// leave-management/requests. The docs state the endpoint defaults its own
// from/to window to only the CURRENT MONTH when the params are omitted; unlike
// most Sage HR list endpoints (which return everything when unfiltered), that
// default would silently truncate a full backfill to a single month. There is
// no documented way to discover the true earliest leave request, so this is an
// explicit assumption (not derived from the docs) — flagged here and in the
// README for human review.
const defaultLeaveRequestLookbackYears = 5

// applyDateWindow sets the initial from/to window for leave-management/requests.
// Since is used when supplied (incremental sync); otherwise a fixed lookback
// is substituted so a full sync doesn't fall back to the API's own
// current-month-only default (see defaultLeaveRequestLookbackYears).
func applyDateWindow(url *urlbuilder.URL, params common.ReadParams) {
	until := params.Until
	if until.IsZero() {
		until = time.Now().UTC()
	}

	since := params.Since
	if since.IsZero() {
		since = until.AddDate(-defaultLeaveRequestLookbackYears, 0, 0)
	}

	windowEnd := since.AddDate(0, 0, maxLeaveRequestWindowDays-1)
	if windowEnd.After(until) {
		windowEnd = until
	}

	url.WithQueryParam("from", since.Format(time.DateOnly))
	url.WithQueryParam("to", windowEnd.Format(time.DateOnly))
}

func cloneURL(u *urlbuilder.URL) (*urlbuilder.URL, error) {
	return urlbuilder.New(u.String())
}

// metaNextPage follows the standard `meta.next_page` (int|null) convention
// shared by every paginated Sage HR endpoint.
func metaNextPage(reqURL *urlbuilder.URL, style paginationStyle) common.NextPageFunc {
	return func(root *ajson.Node) (string, error) {
		if style == paginationNone || reqURL == nil || root == nil {
			return "", nil
		}

		meta, err := jsonquery.New(root).ObjectOptional("meta")
		if err != nil || meta == nil {
			return "", err
		}

		nextPageNum, err := jsonquery.New(meta).IntegerOptional("next_page")
		if err != nil || nextPageNum == nil {
			return "", err
		}

		next, err := cloneURL(reqURL)
		if err != nil {
			return "", err
		}

		next.WithQueryParam("page", strconv.FormatInt(*nextPageNum, 10))

		return next.String(), nil
	}
}

// dateWindowNextPage wraps metaNextPage for leave-management/requests: once a
// from/to window's pages are exhausted, it advances to the next <65-day
// window (per the docs' constraint) until params.Until (or now) is reached.
// applyDateWindow always sets an explicit from/to (defaulting Since to
// defaultLeaveRequestLookbackYears back when the caller supplied none), so
// this always has a window to chunk through.
func dateWindowNextPage(params common.ReadParams, reqURL *urlbuilder.URL) common.NextPageFunc {
	return func(root *ajson.Node) (string, error) {
		next, err := metaNextPage(reqURL, paginationPage)(root)
		if err != nil || next != "" {
			return next, err
		}

		currentTo, ok := reqURL.GetFirstQueryParam("to")
		if !ok {
			return "", nil
		}

		toDate, err := time.Parse(time.DateOnly, currentTo)
		if err != nil {
			return "", err
		}

		overallUntil := params.Until
		if overallUntil.IsZero() {
			overallUntil = time.Now().UTC()
		}

		nextFrom := toDate.AddDate(0, 0, 1)
		if nextFrom.After(overallUntil) {
			return "", nil
		}

		nextTo := nextFrom.AddDate(0, 0, maxLeaveRequestWindowDays-1)
		if nextTo.After(overallUntil) {
			nextTo = overallUntil
		}

		nextURL, err := cloneURL(reqURL)
		if err != nil {
			return "", err
		}

		nextURL.WithQueryParam("page", "1")
		nextURL.WithQueryParam("from", nextFrom.Format(time.DateOnly))
		nextURL.WithQueryParam("to", nextTo.Format(time.DateOnly))

		return nextURL.String(), nil
	}
}

// parseDateWindowedResponse builds a ReadResult for leave-management/requests
// directly, instead of via common.ParseResult. common.ParseResult forces
// Done=true whenever a page has zero records, which is correct for ordinary
// pagination but wrong here: a single <65-day window can legitimately return
// zero records while earlier or later windows (up to params.Until, or the
// present) still have data. Done must be driven solely by whether
// dateWindowNextPage has another window left to walk.
func parseDateWindowedResponse(
	resp *common.JSONHTTPResponse, params common.ReadParams, reqURL *urlbuilder.URL,
) (*common.ReadResult, error) {
	body, ok := resp.Body()
	if !ok {
		return nil, common.ErrEmptyJSONHTTPResponse
	}

	records, err := recordsFunc(body)
	if err != nil {
		return nil, err
	}

	rows, err := marshalFunc()(records, params.Fields.List())
	if err != nil {
		return nil, err
	}

	nextPage, err := dateWindowNextPage(params, reqURL)(body)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		rows = make([]common.ReadResultRow, 0)
	}

	return &common.ReadResult{
		Rows:     int64(len(rows)),
		Data:     rows,
		NextPage: common.NextPageToken(nextPage),
		Done:     nextPage == "",
	}, nil
}

// extractParentIDs pulls the integer `id` field off each parent record node.
func extractParentIDs(nodes []*ajson.Node) []string {
	ids := make([]string, 0, len(nodes))

	for _, node := range nodes {
		id, err := jsonquery.New(node).IntegerOptional("id")
		if err != nil || id == nil {
			continue
		}

		ids = append(ids, strconv.FormatInt(*id, 10))
	}

	return ids
}
