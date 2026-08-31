package activecampaigncl

import (
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

const (
	limitParam         = "limit"
	offsetParam        = "offset"
	defaultPageSize    = "100"
	defaultPageSizeInt = 100
)

// incrementalStrategy captures how (or whether) an object supports
// time-based filtering, per CLAUDE.md's incremental-read rule. See
// providers/activecampaigncl/README.md for the full per-object rationale.
type incrementalStrategy int

const (
	// incrementalNone: the object has no updated-at field at all (users), or
	// only a created-at field (tags) which CLAUDE.md forbids using in place
	// of updated-at. Full sync only.
	incrementalNone incrementalStrategy = iota
	// incrementalNative: the object's list endpoint documents native
	// filters[updated_after]/filters[updated_before] query params.
	incrementalNative
	// incrementalClientSide: no query-param filter exists, but the record
	// carries its own updated-at field, so we fetch every page and discard
	// records outside the Since/Until window in parseReadResponse.
	incrementalClientSide
)

// objectSpec describes per-object read behaviour. Every object shares the
// same response envelope shape: {"<objectName>": [...], "meta": {"total": N}}
// (verified against docs/openapi_spec.json for all 8 objects), so no
// per-object response-key mapping is needed — the object name IS the
// response key.
type objectSpec struct {
	incremental incrementalStrategy
	// clientTimeField is the record field used for incrementalClientSide
	// filtering. Unused otherwise.
	clientTimeField string
}

//nolint:gochecknoglobals
var objectSpecs = map[string]objectSpec{
	// Native filters[updated_after]/[updated_before]; response fields cdate
	// (created) / udate (updated).
	objectContacts: {incremental: incrementalNative},
	// Native filters[updated_after]/[updated_before]; response fields cdate
	// (created) / mdate (modified) — NOT udate, unlike contacts.
	objectDeals: {incremental: incrementalNative},
	// No filters[updated_*] query param declared for /accounts, but the
	// record carries camelCase updatedTimestamp.
	objectAccounts: {incremental: incrementalClientSide, clientTimeField: "updatedTimestamp"},
	// No filters[updated_*] query param declared for /campaigns (only
	// filters[seriesid] and orders[sdate]/orders[ldate]), but the record
	// carries mdate (modified date).
	objectCampaigns: {incremental: incrementalClientSide, clientTimeField: "mdate"},
	// /users has no time field of any kind in its response (id, email,
	// links, phone, lastName, username, firstName, signature) — full sync
	// only, nothing to filter on.
	objectUsers: {incremental: incrementalNone},
	// No filters[updated_*] query param declared for /lists, but the record
	// carries udate (updated).
	objectLists: {incremental: incrementalClientSide, clientTimeField: "udate"},
	// /tags only has cdate (created), no updated-at field at all. Per
	// CLAUDE.md ("always filter by updated_at, never created_at"), we do NOT
	// use cdate as a stand-in — full sync only.
	objectTags: {incremental: incrementalNone},
	// No filters[updated_*] query param declared for /dealGroups, but the
	// record carries udate (updated) per the OpenAPI response example.
	objectDealGroups: {incremental: incrementalClientSide, clientTimeField: "udate"},
}

// offsetNextPage implements the shared limit/offset pagination convention
// documented at https://developers.activecampaign.com/reference/pagination:
// the response never echoes the requested offset back, so the next offset is
// derived from the outgoing request's own URL, and meta.total (a sibling of
// the top-level plural collection key, sometimes a JSON string) tells us
// when to stop.
func offsetNextPage(reqURL *urlbuilder.URL) common.NextPageFunc {
	return func(root *ajson.Node) (string, error) {
		meta, err := jsonquery.New(root).ObjectOptional("meta")
		if err != nil {
			return "", err
		}

		if meta == nil {
			return "", nil
		}

		totalStr, err := jsonquery.New(meta).TextWithDefault("total", "")
		if err != nil {
			return "", err
		}

		total, err := strconv.Atoi(totalStr)
		if err != nil {
			// meta.total missing or not numeric: nothing reliable to page against.
			return "", nil //nolint:nilerr
		}

		limit := queryParamIntWithDefault(reqURL, limitParam, defaultPageSizeInt)
		offset := queryParamIntWithDefault(reqURL, offsetParam, 0)
		nextOffset := offset + limit

		if nextOffset >= total {
			return "", nil
		}

		next, err := urlbuilder.New(reqURL.String())
		if err != nil {
			return "", err
		}

		next.WithQueryParam(offsetParam, strconv.Itoa(nextOffset))

		return next.String(), nil
	}
}

func queryParamIntWithDefault(reqURL *urlbuilder.URL, name string, defaultValue int) int {
	raw, ok := reqURL.GetFirstQueryParam(name)
	if !ok {
		return defaultValue
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}

	return value
}
