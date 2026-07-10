// Command read performs live reads against the BambooHR API using credentials from the
// standard credscanning JSON file.
//
// Usage:
//
//	# One object, printed to stdout:
//	go run ./test/bamboohr/read -object employees -since 720h -fields employeeId,firstName,lastName
//
//	# Capture mode — every supported object, one JSON file per object:
//	go run ./test/bamboohr/read -out /path/to/dir
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers/bamboohr/metadata"
	testBambooHR "github.com/amp-labs/connectors/test/bamboohr"
	"github.com/amp-labs/connectors/test/utils"
)

//nolint:gochecknoglobals
var defaultFields = map[string][]string{
	"employees":           {"employeeId", "firstName", "lastName", "status"},
	"employees/directory": {"id", "firstName", "lastName"},
	"custom-reports":      {"id", "name"},
	"time_off/requests":   {"id", "employeeId", "start", "end", "status"},
}

// capture is the full JSON envelope written for one object.
type capture struct {
	Object     string                     `json:"object"`
	Since      string                     `json:"since,omitempty"`
	Fields     []string                   `json:"fields"`
	HTTPCalls  []testBambooHR.Interaction `json:"http_calls"`
	Result     captureResult              `json:"result"`
	DurationMs int64                      `json:"duration_ms"`
}

type captureResult struct {
	Status      string           `json:"status"`
	Error       string           `json:"error,omitempty"`
	RecordCount int              `json:"record_count"`
	NextPage    string           `json:"next_page,omitempty"`
	Records     []map[string]any `json:"records,omitempty"`
}

func main() {
	objectFlag := flag.String("object", "", "single object to read; if omitted, every supported object is read")
	sinceFlag := flag.Duration("since", 0, "how far back to read, e.g. 720h (30 days); omitted means no Since filter")
	fieldsFlag := flag.String("fields", "", "comma-separated list of fields to request; overrides the built-in defaults for every object read")
	outFlag := flag.String("out", "", "directory to write one capture JSON file per object; if omitted, prints to stdout")
	flag.Parse()

	utils.SetupLogging()

	ctx := context.Background()

	objects := []string{*objectFlag}
	if *objectFlag == "" {
		objects = metadata.Schemas.ObjectNames().GetList(common.ModuleRoot)
	}

	var explicitFields []string
	if *fieldsFlag != "" {
		explicitFields = strings.Split(*fieldsFlag, ",")
	}

	if *outFlag != "" {
		if err := os.MkdirAll(*outFlag, 0o755); err != nil { //nolint:mnd
			utils.Fail("error creating output directory", "error", err)
		}
	}

	for _, object := range objects {
		fields := explicitFields
		if len(fields) == 0 {
			fields = defaultFieldsFor(object)
		}

		capt := readObject(ctx, object, fields, *sinceFlag)

		if *outFlag == "" {
			utils.DumpJSON(capt, os.Stdout)

			continue
		}

		writeCapture(*outFlag, object, capt)
	}
}

func defaultFieldsFor(object string) []string {
	if fields, ok := defaultFields[object]; ok {
		return fields
	}

	// Every employee-table object shares the same minimal, always-present field set.
	return []string{"id", "employeeId"}
}

func readObject(ctx context.Context, object string, fields []string, since time.Duration) capture {
	var interactions []testBambooHR.Interaction

	conn := testBambooHR.GetBambooHRConnector(ctx, func(interaction testBambooHR.Interaction) {
		interactions = append(interactions, interaction)
	})

	params := common.ReadParams{
		ObjectName: object,
		Fields:     connectors.Fields(fields...),
	}

	sinceLabel := ""
	if since > 0 {
		params.Since = time.Now().Add(-since)
		sinceLabel = params.Since.Format(time.RFC3339)
	}

	slog.Info("reading object", "object", object, "fields", fields, "since", sinceLabel)

	start := time.Now()
	res, err := conn.Read(ctx, params)
	duration := time.Since(start)

	capt := capture{
		Object:     object,
		Since:      sinceLabel,
		Fields:     fields,
		HTTPCalls:  interactions,
		DurationMs: duration.Milliseconds(),
	}

	if err != nil {
		slog.Error("error reading object", "object", object, "error", err)

		capt.Result = captureResult{Status: "error", Error: err.Error()}

		return capt
	}

	records := make([]map[string]any, 0, len(res.Data))
	for _, row := range res.Data {
		records = append(records, row.Raw)
	}

	capt.Result = captureResult{
		Status:      "ok",
		RecordCount: int(res.Rows),
		NextPage:    res.NextPage.String(),
		Records:     records,
	}

	return capt
}

func writeCapture(outDir, object string, capt capture) {
	// Table object names never contain path separators, but "employees/directory" and
	// "time_off/requests" do — flatten them into a filesystem-safe file name.
	fileName := strings.ReplaceAll(object, "/", "_") + ".json"

	data, err := json.MarshalIndent(capt, "", "  ")
	if err != nil {
		utils.Fail("error marshaling capture", "object", object, "error", err)
	}

	path := filepath.Join(outDir, fileName)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		utils.Fail("error writing capture file", "object", object, "path", path, "error", err)
	}

	fmt.Fprintf(os.Stdout, "wrote %s\n", path) //nolint:forbidigo
}
