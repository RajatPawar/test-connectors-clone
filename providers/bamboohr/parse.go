package bamboohr

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// idFieldForObject returns the field within each record that holds the record's unique ID.
func idFieldForObject(objectName string) readhelper.IdFieldQuery {
	if objectName == objectNameEmployees {
		// List Employees keys each record by "employeeId", not "id".
		return readhelper.NewIdField("employeeId")
	}

	// employees/directory, custom-reports, time_off/requests, and every employee table
	// key their records by "id".
	return readhelper.NewIdField("id")
}

// recordsFunc picks the record-extraction function for the given object. usingChangedTablesEndpoint
// must be true only when the request went to the "Get Changed Employee Table Data" endpoint
// (an employee table object read with Since set), since that endpoint uses a different envelope
// than every other object.
func recordsFunc(objectName string, usingChangedTablesEndpoint bool) common.RecordsFunc {
	switch {
	case usingChangedTablesEndpoint:
		return changedTableRecords
	case objectName == objectNameEmployees:
		return common.ExtractRecordsFromPath("data")
	case objectName == objectNameEmployeeDirectory:
		return common.ExtractRecordsFromPath("employees")
	case objectName == objectNameCustomReports:
		return common.ExtractRecordsFromPath("reports")
	default:
		// time_off/requests and the "all employees" table read both return a bare
		// top-level JSON array.
		return common.ExtractRecordsFromPath("")
	}
}

// changedTableRecords flattens the "Get Changed Employee Table Data" response shape —
//
//	{"table": "...", "employees": {"<employeeId>": {"rows": [...], "lastChanged": "..."}}}
//
// into a flat list of rows, attaching employeeId and lastChanged to each row so callers see
// the same row shape as a full-table read.
func changedTableRecords(node *ajson.Node) ([]map[string]any, error) {
	employeesNode, err := jsonquery.New(node).ObjectOptional("employees")
	if err != nil {
		return nil, err
	}

	records := make([]map[string]any, 0)

	if employeesNode == nil {
		return records, nil
	}

	employeesMap, err := jsonquery.Convertor.ObjectToMap(employeesNode)
	if err != nil {
		return nil, err
	}

	for employeeID, value := range employeesMap {
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}

		lastChanged := entry["lastChanged"]

		rows, _ := entry["rows"].([]any) //nolint:errcheck

		for _, row := range rows {
			rowMap, ok := row.(map[string]any)
			if !ok {
				continue
			}

			rowMap["employeeId"] = employeeID
			rowMap["lastChanged"] = lastChanged
			records = append(records, rowMap)
		}
	}

	return records, nil
}

// nextPageFunc picks the pagination-token extractor for the given object. Every table object
// and time_off/requests return their full result set in a single response (no next-page token).
func nextPageFunc(objectName string) common.NextPageFunc {
	switch objectName {
	case objectNameEmployees:
		return nextPageFromLinksHref
	case objectNameCustomReports:
		return nextPageFromPaginationNextPage
	default:
		return noNextPage
	}
}

// nextPageFromLinksHref reads the full next-page URL from `_links.next.href`, as returned by
// List Employees.
func nextPageFromLinksHref(node *ajson.Node) (string, error) {
	return jsonquery.New(node, "_links", "next").StrWithDefault("href", "")
}

// nextPageFromPaginationNextPage reads the full next-page URL from `pagination.next_page`, as
// returned by List Reports.
func nextPageFromPaginationNextPage(node *ajson.Node) (string, error) {
	return jsonquery.New(node, "pagination").StrWithDefault("next_page", "")
}

func noNextPage(*ajson.Node) (string, error) {
	return "", nil
}
