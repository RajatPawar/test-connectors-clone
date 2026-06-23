package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/providers/bamboohrv3"
	connTest "github.com/amp-labs/connectors/test/bamboohrv3"
	testUtils "github.com/amp-labs/connectors/test/utils"
)

func main() {
	objectFlag := flag.String("object", "", "Object name to read (omit to read all supported objects)")
	sinceFlag := flag.Duration("since", 0, "How far back to read (e.g. 720h). Omit for no time filter.")
	fieldsFlag := flag.String("fields", "", "Comma-separated list of fields to read")
	outFlag := flag.String("out", "", "Output directory for capture mode (one JSON file per object)")
	flag.Parse()

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	testUtils.SetupLogging()

	transport := &loggingRoundTripper{inner: http.DefaultTransport}
	httpClient := &http.Client{Transport: transport}
	conn := connTest.GetBambooHRConnector(ctx, httpClient)

	var targets []string
	if *objectFlag != "" {
		targets = []string{*objectFlag}
	} else {
		targets = bamboohrv3.SupportedObjects
	}

	var fields []string
	if *fieldsFlag != "" {
		fields = strings.Split(*fieldsFlag, ",")
	}

	for _, obj := range targets {
		transport.reset()

		entry := readObject(ctx, conn, obj, *sinceFlag, fields, transport)

		data, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling capture for %s: %v\n", obj, err)
			continue
		}

		if *outFlag != "" {
			// Capture mode: write to <out>/<object>.json (replace / with __)
			filename := strings.ReplaceAll(obj, "/", "__") + ".json"
			path := filepath.Join(*outFlag, filename)

			if err := os.WriteFile(path, data, 0o600); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
			} else {
				fmt.Printf("wrote %s (%d records)\n", path, entry.Result.RecordCount)
			}
		} else {
			os.Stdout.Write(data) //nolint:errcheck
			os.Stdout.WriteString("\n")
		}
	}
}

func readObject(
	ctx context.Context,
	conn *bamboohrv3.Connector,
	objectName string,
	since time.Duration,
	fields []string,
	transport *loggingRoundTripper,
) captureEntry {
	effectiveFields := fields
	if len(effectiveFields) == 0 {
		effectiveFields = defaultFieldsFor(objectName)
	}

	var sinceTime time.Time
	if since > 0 {
		sinceTime = time.Now().Add(-since)
	}

	params := common.ReadParams{
		ObjectName: objectName,
		Fields:     datautils.NewSet(effectiveFields...),
		Since:      sinceTime,
	}

	start := time.Now()
	result, err := conn.Read(ctx, params)
	durationMs := time.Since(start).Milliseconds()

	entry := captureEntry{
		Object:     objectName,
		Fields:     effectiveFields,
		DurationMs: durationMs,
	}

	if since > 0 {
		entry.Since = sinceTime.Format(time.RFC3339)
	}

	// Attach captured HTTP interaction (first/last request made during this read).
	if cap := transport.last(); cap != nil {
		entry.Request = cap.Req
		entry.Response = cap.Resp
	}

	if err != nil {
		entry.Result = captureResult{
			Status: "error",
			Error:  err.Error(),
		}
		return entry
	}

	records := make([]interface{}, len(result.Data))
	for i, row := range result.Data {
		records[i] = row.Raw
	}

	nextPage := ""
	if result.NextPage != "" {
		nextPage = result.NextPage.String()
	}

	entry.Result = captureResult{
		Status:      "ok",
		RecordCount: len(result.Data),
		NextPage:    nextPage,
		Records:     records,
	}

	return entry
}

// defaultFieldsFor returns safe id-like fields to request when none are specified.
func defaultFieldsFor(objectName string) []string {
	switch objectName {
	case "api/v1/employees":
		return []string{"employeeId", "firstName", "lastName"}
	case "applications":
		return []string{"id", "applicantId", "status"}
	case "jobs":
		return []string{"id", "title", "status"}
	case "requests":
		return []string{"id", "type", "status"}
	case "timesheet_entries":
		return []string{"id", "type", "hours"}
	case "schedules":
		return []string{"id", "name", "timezone"}
	default:
		return []string{"id"}
	}
}

// captureEntry is the full HTTP interaction + parsed result for one object read.
type captureEntry struct {
	Object     string         `json:"object"`
	Since      string         `json:"since,omitempty"`
	Fields     []string       `json:"fields"`
	Request    *captureReq    `json:"request"`
	Response   *captureResp   `json:"response"`
	Result     captureResult  `json:"result"`
	DurationMs int64          `json:"duration_ms"`
}

type captureReq struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Query   map[string][]string `json:"query"`
	Body    interface{}         `json:"body"`
}

type captureResp struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body"`
}

type captureResult struct {
	Status      string        `json:"status"`
	Error       string        `json:"error,omitempty"`
	RecordCount int           `json:"record_count"`
	NextPage    string        `json:"next_page,omitempty"`
	Records     []interface{} `json:"records,omitempty"`
}

// loggingRoundTripper captures the last HTTP request/response pair.
type loggingRoundTripper struct {
	inner   http.RoundTripper
	entries []*httpCapture
}

type httpCapture struct {
	Req  *captureReq
	Resp *captureResp
}

func (t *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cap := &httpCapture{}

	// Read and restore request body.
	var reqBodyBytes []byte
	if req.Body != nil {
		reqBodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	}

	cap.Req = &captureReq{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: map[string][]string(req.Header),
		Query:   map[string][]string(req.URL.Query()),
	}
	if len(reqBodyBytes) > 0 {
		var bodyJSON interface{}
		if json.Unmarshal(reqBodyBytes, &bodyJSON) == nil {
			cap.Req.Body = bodyJSON
		} else {
			cap.Req.Body = string(reqBodyBytes)
		}
	}

	resp, err := t.inner.RoundTrip(req)

	if err != nil {
		t.entries = append(t.entries, cap)
		return resp, err
	}

	// Read and restore response body.
	var respBodyBytes []byte
	if resp.Body != nil {
		respBodyBytes, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))
	}

	cap.Resp = &captureResp{
		Status:  resp.StatusCode,
		Headers: map[string][]string(resp.Header),
		Body:    json.RawMessage(respBodyBytes),
	}

	t.entries = append(t.entries, cap)

	return resp, nil
}

func (t *loggingRoundTripper) reset() {
	t.entries = nil
}

// last returns the last captured HTTP interaction (most recent request/response).
func (t *loggingRoundTripper) last() *httpCapture {
	if len(t.entries) == 0 {
		return nil
	}

	return t.entries[len(t.entries)-1]
}
