package bamboohrv4

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// responseRecordsPath is the JSON path to the array of records within each
// object's response body. An empty string means the response body itself is
// the array (no envelope).
var responseRecordsPath = map[string]string{ // nolint:gochecknoglobals
	objectNameEmployees:        "data",
	objectNameJobs:             "",
	objectNameApplications:     "applications",
	objectNameRequests:         "",
	objectNameSchedules:        "data",
	objectNameTimesheetEntries: "",
}

func records(objectName string) common.RecordsFunc {
	return common.ExtractRecordsFromPath(responseRecordsPath[objectName])
}

func nextRecordsURL(objectName string) common.NextPageFunc {
	switch objectName {
	case objectNameApplications:
		// {"nextPageUrl": "https://...", "applications": [...]}
		return func(node *ajson.Node) (string, error) {
			return jsonquery.New(node).StrWithDefault("nextPageUrl", "")
		}
	case objectNameEmployees, objectNameSchedules:
		// {"data": [...], "_links": {"next": {"href": "https://..."} | null}}
		return func(node *ajson.Node) (string, error) {
			return nextLinkHref(node)
		}
	default:
		// Single-page responses: jobs, requests, timesheet_entries.
		return func(node *ajson.Node) (string, error) {
			return "", nil
		}
	}
}

func nextLinkHref(node *ajson.Node) (string, error) {
	links, err := jsonquery.New(node).ObjectOptional("_links")
	if err != nil || links == nil {
		return "", err
	}

	next, err := jsonquery.New(links).ObjectOptional("next")
	if err != nil || next == nil {
		return "", err
	}

	return jsonquery.New(next).StrWithDefault("href", "")
}
