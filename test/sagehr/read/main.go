package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	sagehrtest "github.com/amp-labs/connectors/test/sagehr"
	"github.com/amp-labs/connectors/test/utils"
)

// objectSpec pairs a Sage HR object name (as declared in providers/sagehr/supports.go)
// with a couple of safe, always-present fields to sample when -fields isn't given.
type objectSpec struct {
	Name   string
	Fields []string
}

var allObjects = []objectSpec{ //nolint:gochecknoglobals
	{"teams", []string{"id", "name"}},
	{"employees", []string{"id", "first_name", "last_name", "email"}},
	{"positions", []string{"id", "title", "code"}},
	{"termination-reasons", []string{"id", "name", "type"}},
	{"documents/categories", []string{"id", "name"}},
	{"terminated-employees", []string{"id", "first_name", "last_name"}},
	{"onboarding/categories", []string{"id", "title"}},
	{"recruitment/positions", []string{"id", "title", "status"}},
	{"offboarding/categories", []string{"id", "title"}},
	{"leave-management/policies", []string{"id", "name"}},
	{"leave-management/requests", []string{"id", "status", "employee_id"}},
	{"leave-management/out-of-office-today", []string{"id", "employee_id", "start_date"}},
}

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	objectName := flag.String("object", "", "a single object to read; if omitted, every supported object is read")
	since := flag.Duration("since", 0, "how far back to set ReadParams.Since, e.g. 720h")
	fieldsFlag := flag.String("fields", "", "comma-separated list of fields to request")
	outDir := flag.String("out", "", "directory to write one capture JSON file per object; if omitted, prints to stdout")
	flag.Parse()

	transport := &capturingTransport{base: http.DefaultTransport}
	conn := sagehrtest.GetSageHRConnectorWithTransport(ctx, transport)

	var sinceTime time.Time
	if *since > 0 {
		sinceTime = time.Now().Add(-*since)
	}

	targets := allObjects
	if *objectName != "" {
		targets = []objectSpec{{Name: *objectName, Fields: defaultFieldsFor(*objectName)}}
	}

	for _, target := range targets {
		fields := target.Fields
		if *fieldsFlag != "" {
			fields = strings.Split(*fieldsFlag, ",")
		}

		capture := runRead(ctx, conn, transport, target.Name, sinceTime, fields)

		if *outDir == "" {
			printCapture(capture)

			continue
		}

		if err := writeCapture(*outDir, capture); err != nil {
			slog.Error("failed to write capture", "object", target.Name, "error", err)
		}
	}
}

func defaultFieldsFor(objectName string) []string {
	for _, o := range allObjects {
		if o.Name == objectName {
			return o.Fields
		}
	}

	// Unknown object (not in our documented list) — fall back to "id" alone.
	return []string{"id"}
}

// capturedRequest/capturedResponse/resultSummary/captureFile mirror the capture
// format described in CLAUDE.md so captures can be replayed/diffed later.
type capturedRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Query   map[string][]string `json:"query"`
	Body    any                 `json:"body"`
}

type capturedResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body"`
}

type resultSummary struct {
	Status      string           `json:"status"`
	Error       string           `json:"error,omitempty"`
	RecordCount int              `json:"record_count"`
	NextPage    string           `json:"next_page,omitempty"`
	Records     []map[string]any `json:"records,omitempty"`
}

type captureFile struct {
	Object     string           `json:"object"`
	Since      string           `json:"since,omitempty"`
	Fields     []string         `json:"fields"`
	Request    capturedRequest  `json:"request"`
	Response   capturedResponse `json:"response"`
	Result     resultSummary    `json:"result"`
	DurationMs int64            `json:"duration_ms"`
}

func runRead(
	ctx context.Context,
	conn connectors.ReadConnector,
	transport *capturingTransport,
	objectName string,
	since time.Time,
	fields []string,
) captureFile {
	transport.reset()

	params := common.ReadParams{
		ObjectName: objectName,
		Fields:     connectors.Fields(fields...),
		Since:      since,
	}

	start := time.Now()
	result, err := conn.Read(ctx, params)
	duration := time.Since(start)

	capture := captureFile{
		Object:     objectName,
		Fields:     fields,
		DurationMs: duration.Milliseconds(),
	}

	if !since.IsZero() {
		capture.Since = since.Format(time.RFC3339)
	}

	if interaction := transport.take(); interaction != nil {
		capture.Request = interaction.Request
		capture.Response = interaction.Response
	}

	if err != nil {
		capture.Result = resultSummary{Status: "error", Error: err.Error()}

		return capture
	}

	records := make([]map[string]any, 0, len(result.Data))

	for _, row := range result.Data {
		records = append(records, row.Raw)
	}

	capture.Result = resultSummary{
		Status:      "ok",
		RecordCount: len(result.Data),
		NextPage:    result.NextPage.String(),
		Records:     records,
	}

	return capture
}

func printCapture(capture captureFile) {
	data, err := json.MarshalIndent(capture, "", "  ")
	if err != nil {
		slog.Error("failed to marshal capture", "object", capture.Object, "error", err)

		return
	}

	fmt.Println(string(data)) //nolint:forbidigo
}

func writeCapture(outDir string, capture captureFile) error {
	path := filepath.Join(outDir, capture.Object+".json")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data, err := json.MarshalIndent(capture, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling capture: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("writing capture file: %w", err)
	}

	slog.Info("wrote capture", "object", capture.Object, "path", path, "records", capture.Result.RecordCount)

	return nil
}

// interaction is the single HTTP request/response pair captured for one Read
// call. Every supported Sage HR read issues exactly one HTTP request per page,
// so capturing the most recent round trip is sufficient.
type interaction struct {
	Request  capturedRequest
	Response capturedResponse
}

type capturingTransport struct {
	base http.RoundTripper
	mu   sync.Mutex
	last *interaction
}

func (t *capturingTransport) reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.last = nil
}

func (t *capturingTransport) take() *interaction {
	t.mu.Lock()
	defer t.mu.Unlock()

	last := t.last
	t.last = nil

	return last
}

func (t *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) { //nolint:varnamelen
	var reqBody any

	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}

		req.Body = io.NopCloser(bytes.NewReader(data))

		if len(data) > 0 {
			_ = json.Unmarshal(data, &reqBody)
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err //nolint:wrapcheck
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respData))

	t.mu.Lock()
	t.last = &interaction{
		Request: capturedRequest{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: req.Header,
			Query:   req.URL.Query(),
			Body:    reqBody,
		},
		Response: capturedResponse{
			Status:  resp.StatusCode,
			Headers: resp.Header,
			Body:    respData,
		},
	}
	t.mu.Unlock()

	return resp, nil
}
