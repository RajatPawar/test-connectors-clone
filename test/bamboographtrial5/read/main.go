package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	bamboographtrial5test "github.com/amp-labs/connectors/test/bamboographtrial5"
	"github.com/amp-labs/connectors/test/utils"
)

// supportedObjects is the canonical list of readable objects for BambooHR.
// Keep this in sync with providers/bamboographtrial5/supports.go.
var supportedObjects = []string{
	"jobs",
	"api/v1/hris/org/locations",
	"schedules",
	"policies",
	"benefitcoverages",
	"whos_out",
}

// captureRecord holds the full HTTP interaction for one object read.
type captureRecord struct {
	Object     string         `json:"object"`
	Since      string         `json:"since,omitempty"`
	Fields     []string       `json:"fields,omitempty"`
	Request    *captureReq    `json:"request,omitempty"`
	Response   *captureResp   `json:"response,omitempty"`
	Result     *captureResult `json:"result"`
	DurationMS int64          `json:"duration_ms"`
}

type captureReq struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Query   map[string][]string `json:"query"`
	Body    *string             `json:"body"`
}

type captureResp struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body"`
}

type captureResult struct {
	Status      string                  `json:"status"`
	Error       string                  `json:"error,omitempty"`
	RecordCount int                     `json:"record_count"`
	NextPage    string                  `json:"next_page,omitempty"`
	Records     []common.ReadResultRow  `json:"records"`
}

// capturingTransport wraps an http.RoundTripper and records the last interaction.
type capturingTransport struct {
	inner    http.RoundTripper
	last     *captureRecord
	start    time.Time
}

func (t *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.start = time.Now()

	cr := &captureRecord{}
	cr.Request = &captureReq{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: map[string][]string(req.Header),
		Query:   map[string][]string(req.URL.Query()),
		Body:    nil,
	}

	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		t.last = cr
		return resp, err
	}

	// Read body, then restore it for the caller.
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(strings.NewReader(string(body)))

	if readErr == nil {
		cr.Response = &captureResp{
			Status:  resp.StatusCode,
			Headers: map[string][]string(resp.Header),
			Body:    json.RawMessage(body),
		}
	}

	cr.DurationMS = time.Since(t.start).Milliseconds()
	t.last = cr

	return resp, nil
}

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	objectFlag := flag.String("object", "", "object name to read (default: all supported objects)")
	sinceFlag := flag.Duration("since", 0, "how far back to read (e.g. 720h)")
	fieldsFlag := flag.String("fields", "", "comma-separated list of fields to request")
	outFlag := flag.String("out", "", "directory to write capture files (enables capture mode)")
	flag.Parse()

	conn := bamboographtrial5test.GetConnector(ctx)

	objects := supportedObjects
	if *objectFlag != "" {
		objects = []string{*objectFlag}
	}

	var since time.Time
	if *sinceFlag > 0 {
		since = time.Now().Add(-*sinceFlag)
	}

	var fields common.ReadParams
	if *fieldsFlag != "" {
		parts := strings.Split(*fieldsFlag, ",")
		fields.Fields = connectors.Fields(parts...)
	} else {
		fields.Fields = connectors.Fields("id")
	}

	for _, obj := range objects {
		params := common.ReadParams{
			ObjectName: obj,
			Fields:     fields.Fields,
			Since:      since,
		}

		if *outFlag != "" {
			readAndCapture(ctx, conn, params, *outFlag)
		} else {
			readAndPrint(ctx, conn, params)
		}
	}

	slog.Info("Read operation completed.")
}

func readAndPrint(ctx context.Context, conn connectors.ReadConnector, params common.ReadParams) {
	slog.Info("Reading object", "object", params.ObjectName)

	result, err := conn.Read(ctx, params)
	if err != nil {
		slog.Error("read error", "object", params.ObjectName, "error", err)

		return
	}

	utils.DumpJSON(result, os.Stdout)
}

func readAndCapture(
	ctx context.Context,
	conn connectors.ReadConnector,
	params common.ReadParams,
	outDir string,
) {
	slog.Info("Capturing object", "object", params.ObjectName)

	start := time.Now()

	result, err := conn.Read(ctx, params)

	durationMS := time.Since(start).Milliseconds()

	rec := &captureRecord{
		Object:    params.ObjectName,
		Since:     params.Since.Format(time.RFC3339),
		DurationMS: durationMS,
	}

	if params.Fields != nil {
		rec.Fields = params.Fields.List()
	}

	if err != nil {
		rec.Result = &captureResult{
			Status:  "error",
			Error:   err.Error(),
			Records: []common.ReadResultRow{},
		}
	} else {
		rec.Result = &captureResult{
			Status:      "ok",
			RecordCount: int(result.Rows),
			NextPage:    result.NextPage.String(),
			Records:     result.Data,
		}
	}

	outPath := filepath.Join(outDir, safeFilename(params.ObjectName)+".json")

	data, jsonErr := json.MarshalIndent(rec, "", "  ")
	if jsonErr != nil {
		slog.Error("failed to marshal capture", "object", params.ObjectName, "error", jsonErr)

		return
	}

	if writeErr := os.WriteFile(outPath, data, 0o600); writeErr != nil {
		slog.Error("failed to write capture file", "object", params.ObjectName, "error", writeErr)

		return
	}

	slog.Info("Capture written", "object", params.ObjectName, "file", outPath,
		"record_count", rec.Result.RecordCount)
}

// safeFilename converts an object name like "api/v1/hris/org/locations" to a
// filesystem-safe name like "api_v1_hris_org_locations".
func safeFilename(objectName string) string {
	return strings.NewReplacer("/", "_").Replace(objectName)
}

