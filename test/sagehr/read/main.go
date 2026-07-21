// Command read is a flag-driven live-read tool for the Sage HR connector.
//
// Single object, printed to stdout:
//
//	go run ./test/sagehr/read -object teams -fields id,name
//
// Every supported object, captured (full HTTP interaction + parsed result) to
// one JSON file per object:
//
//	go run ./test/sagehr/read -out /path/to/dir
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
	"sync"
	"syscall"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers/sagehr"
	connTest "github.com/amp-labs/connectors/test/sagehr"
	"github.com/amp-labs/connectors/test/utils"
)

// defaultFields is used when -fields is omitted. Most objects have a
// meaningful "id"; the two fan-out objects below have no natural id field in
// the provider's response (see providers/sagehr/README.md), so a couple of
// other safe, always-present fields are used instead.
//
//nolint:gochecknoglobals
var defaultFields = map[string][]string{
	"employees/compensations":             {"amount", "start_date"},
	"employees/leave-management/balances": {"policy_id", "used", "available"},
}

func main() {
	objectFlag := flag.String("object", "", "single object to read; if omitted, every supported object is read")
	sinceFlag := flag.Duration("since", 0, "how far back to read, e.g. 720h (optional)")
	fieldsFlag := flag.String("fields", "", "comma-separated list of fields (optional)")
	outFlag := flag.String("out", "", "directory to write one capture JSON file per object; if omitted, prints to stdout")
	flag.Parse()

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	objects := sagehr.SupportedReadObjects()
	if *objectFlag != "" {
		objects = []string{*objectFlag}
	}

	var since time.Time
	if *sinceFlag > 0 {
		since = time.Now().UTC().Add(-*sinceFlag)
	}

	var explicitFields []string
	if *fieldsFlag != "" {
		explicitFields = strings.Split(*fieldsFlag, ",")
	}

	if *outFlag != "" {
		if err := os.MkdirAll(*outFlag, 0o755); err != nil {
			utils.Fail("error creating output directory", "error", err)
		}
	}

	for _, object := range objects {
		fields := explicitFields
		if len(fields) == 0 {
			fields = fieldsFor(object)
		}

		capture := newCapturingClient()
		conn := connTest.GetConnectorWithClientWrapper(ctx, func(real common.AuthenticatedHTTPClient) common.AuthenticatedHTTPClient {
			capture.inner = real

			return capture
		})

		record := runOne(ctx, conn, object, since, fields, capture)

		if *outFlag == "" {
			printJSON(record)

			continue
		}

		writeRecord(*outFlag, object, record)
	}
}

func fieldsFor(object string) []string {
	if fields, ok := defaultFields[object]; ok {
		return fields
	}

	return []string{"id"}
}

type captureRecord struct {
	Object   string           `json:"object"`
	Since    string           `json:"since"`
	Fields   []string         `json:"fields"`
	Request  *requestCapture  `json:"request"`
	Response *responseCapture `json:"response"`
	Result   resultCapture    `json:"result"`
	// Interactions holds every HTTP call made while reading this object.
	// Most objects make exactly one (matching Request/Response above); the
	// fan-out objects (employees/compensations, employees/custom-fields,
	// employees/leave-management/balances, recruitment/positions/applicants)
	// make one parent-list call plus one call per parent id.
	Interactions []interaction `json:"interactions"`
	DurationMs   int64         `json:"duration_ms"`
}

type resultCapture struct {
	Status      string           `json:"status"`
	Error       string           `json:"error,omitempty"`
	RecordCount int              `json:"record_count"`
	NextPage    string           `json:"next_page,omitempty"`
	Records     []map[string]any `json:"records"`
}

func runOne(
	ctx context.Context, conn connectors.ReadConnector, object string, since time.Time, fields []string,
	capture *capturingClient,
) *captureRecord {
	start := time.Now()

	result, err := conn.Read(ctx, common.ReadParams{
		ObjectName: object,
		Fields:     connectors.Fields(fields...),
		Since:      since,
	})

	duration := time.Since(start)
	interactions := capture.drain()

	record := &captureRecord{
		Object:       object,
		Fields:       fields,
		Interactions: interactions,
		DurationMs:   duration.Milliseconds(),
	}

	if !since.IsZero() {
		record.Since = since.Format(time.RFC3339)
	}

	// For fan-out objects (e.g. employees/leave-management/balances) there are
	// multiple interactions: a parent-list call followed by one call per
	// parent id. The LAST interaction is the one whose response actually
	// produced Result.Records, so top-level Request/Response must reflect
	// that call, not the first (parent-list) one — otherwise a reviewer
	// comparing top-level request/response against result.records sees a
	// spurious mismatch (parent object's fields vs. child object's fields).
	if len(interactions) > 0 {
		last := interactions[len(interactions)-1]
		record.Request = &last.Request
		record.Response = &last.Response
	}

	if err != nil {
		record.Result = resultCapture{Status: "error", Error: err.Error()}

		return record
	}

	records := make([]map[string]any, 0, len(result.Data))
	for _, row := range result.Data {
		records = append(records, row.Raw)
	}

	record.Result = resultCapture{
		Status:      "ok",
		RecordCount: len(records),
		NextPage:    result.NextPage.String(),
		Records:     records,
	}

	return record
}

func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		utils.Fail("error marshalling JSON", "error", err)
	}

	fmt.Println(string(data)) //nolint:forbidigo
}

func writeRecord(outDir, object string, record *captureRecord) {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		utils.Fail("error marshalling JSON", "error", err)
	}

	// Object names contain "/" (e.g. "employees/compensations"); flatten to a
	// safe filename rather than creating nested directories.
	fileName := strings.ReplaceAll(object, "/", "_") + ".json"

	if err := os.WriteFile(filepath.Join(outDir, fileName), data, 0o644); err != nil { //nolint:gosec
		utils.Fail("error writing capture file", "error", err, "object", object)
	}
}

// --- HTTP capture plumbing -------------------------------------------------

type requestCapture struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Query   map[string][]string `json:"query"`
	Body    json.RawMessage     `json:"body"`
}

type responseCapture struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body"`
}

type interaction struct {
	Request  requestCapture  `json:"request"`
	Response responseCapture `json:"response"`
}

// capturingClient wraps an authenticated HTTP client and records the full
// request/response of every call it makes, so capture mode can persist the
// gold data (see CLAUDE.md's Testing section) rather than just the parsed
// result.
type capturingClient struct {
	inner        common.AuthenticatedHTTPClient
	mu           sync.Mutex
	interactions []interaction
}

func newCapturingClient() *capturingClient {
	return &capturingClient{}
}

func (c *capturingClient) Do(req *http.Request) (*http.Response, error) {
	reqCap := requestCapture{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: map[string][]string(req.Header.Clone()),
		Query:   map[string][]string(req.URL.Query()),
	}

	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		if len(bodyBytes) > 0 {
			reqCap.Body = json.RawMessage(bodyBytes)
		}
	}

	resp, err := c.inner.Do(req)
	if err != nil {
		c.record(interaction{Request: reqCap})

		return resp, err
	}

	bodyBytes, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	respCap := responseCapture{
		Status:  resp.StatusCode,
		Headers: map[string][]string(resp.Header.Clone()),
		Body:    json.RawMessage(bodyBytes),
	}

	c.record(interaction{Request: reqCap, Response: respCap})

	return resp, readErr
}

func (c *capturingClient) CloseIdleConnections() {
	if c.inner != nil {
		c.inner.CloseIdleConnections()
	}
}

func (c *capturingClient) record(i interaction) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.interactions = append(c.interactions, i)
}

func (c *capturingClient) drain() []interaction {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := c.interactions
	c.interactions = nil

	return out
}
