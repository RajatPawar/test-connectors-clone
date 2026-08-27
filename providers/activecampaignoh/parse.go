package activecampaignoh

import (
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// pageSize is the max page size documented for every v3 list endpoint.
// https://developers.activecampaign.com/reference/pagination
const pageSize = 100

// incrementalTimestampField maps an object to the response field used to
// advance an incremental (connector-side filtered) read. Only objects that
// (a) have no filters[updated_after]/[updated_before] query parameter and
// (b) DO expose a timestamp field in their records are listed here.
//
// contacts and deals are absent from this map: they support native
// filters[updated_after]/[updated_before] query parameters (see
// buildReadRequest), so no connector-side filtering is needed for them.
//
// tags and users are absent too, but for the opposite reason: neither the
// query parameters NOR a usable updated-at field exist for those objects
// (see connie_context feasibility verdicts c1e54af7 / 51ea10bf), so they are
// full-read-only with no filtering at all.
//
// nolint:gochecknoglobals
var incrementalTimestampField = map[string]string{
	objectAccounts:  "updatedTimestamp", // camelCase; accounts have no filters[updated_*] param.
	objectLists:     "udate",
	objectCampaigns: "mdate",
	objectDealTasks: "udate",
}

// nativeIncrementalObjects support filters[updated_after]/filters[updated_before]
// query parameters natively, confirmed in docs/openapi_spec.json.
// nolint:gochecknoglobals
var nativeIncrementalObjects = datautils.NewSet(objectContacts, objectDeals)

// recordsPath returns the JSON key holding the record array for an object.
// Every v3 list response nests its rows under a top-level plural key
// identical to the object/URL-path name (e.g. {"deals": [...], "meta": {...}}).
func recordsPath(objectName string) string {
	return objectName
}

func idField() readhelper.IdFieldQuery {
	return readhelper.NewIdField("id")
}

// makeNextPageFunc paginates via offset += limit until offset >= meta.total.
// There is no next-page cursor/link; meta.total's JSON type is inconsistent
// across endpoints (int in prose examples, string in the OpenAPI schema), so
// it's read defensively as text. reqURL is the URL of the request that
// produced this response; offset/limit are read back off of it so pagination
// keeps whatever filters/orders were set on the very first request.
func makeNextPageFunc(reqURL *urlbuilder.URL) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		totalText, err := jsonquery.New(node, "meta").TextWithDefault("total", "")
		if err != nil {
			return "", err
		}

		if totalText == "" {
			return "", nil
		}

		total, err := strconv.Atoi(totalText)
		if err != nil {
			// Malformed/unexpected meta.total: stop rather than loop forever.
			return "", nil // nolint:nilerr
		}

		offset := 0
		if offsetStr, ok := reqURL.GetFirstQueryParam("offset"); ok {
			offset, _ = strconv.Atoi(offsetStr)
		}

		nextOffset := offset + pageSize
		if nextOffset >= total {
			return "", nil
		}

		nextURL, err := urlbuilder.New(reqURL.String())
		if err != nil {
			return "", err
		}

		nextURL.WithQueryParam("offset", strconv.Itoa(nextOffset))

		return nextURL.String(), nil
	}
}

// makeRecordsFilterFunc returns the connector-side time filter for objects
// with no native filters[updated_*] query parameter but that do expose an
// updated-at field. Records are NOT guaranteed sorted by that field (none of
// these list endpoints document an orders[] option for their timestamp
// field), so Unordered is used: every page is fetched and filtered, with no
// early-stop optimization.
func makeRecordsFilterFunc(objectName string, nextPage common.NextPageFunc) common.RecordsFilterFunc {
	timestampField, ok := incrementalTimestampField[objectName]
	if !ok {
		return readhelper.MakeIdentityFilterFunc(nextPage)
	}

	return readhelper.MakeTimeFilterFunc(
		readhelper.Unordered,
		readhelper.NewTimeBoundary(),
		timestampField,
		time.RFC3339,
		nextPage,
	)
}
