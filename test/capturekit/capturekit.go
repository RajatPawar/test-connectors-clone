// Package capturekit is the GENERIC live-read/capture engine shared by every
// connector's test/<name>/read shim. The shim supplies only the provider id and
// its object list:
//
//	func main() { capturekit.Main(providers.SageHR, sagehr.SupportedReadObjects()) }
//
// Everything error-prone — building the auth client from the provider's auth
// type, and passing the WHOLE creds `metadata` map into ConnectorParams so
// templated vars like {{.workspace}} resolve — lives here, once, so a connector
// can never mis-wire its own credentials. Objects/fields/times are flags; the
// tool never validates an object, it just reads what it's handed and captures
// the request/response/result faithfully (errors verbatim).
package capturekit

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

	"golang.org/x/oauth2"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/common/substitutions/catalogreplacer"
	"github.com/amp-labs/connectors/connector"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/test/utils"
)

// Main is the shim entry point. `objects` is the connector's SupportedReadObjects();
// -object overrides it for a single read.
func Main(provider providers.Provider, objects []string) {
	objectFlag := flag.String("object", "", "single object to read; default: every object passed by the shim")
	sinceFlag := flag.Duration("since", 0, "how far back to read, e.g. 720h (optional)")
	fieldsFlag := flag.String("fields", "", "comma-separated fields (optional; discovered from metadata if omitted)")
	outFlag := flag.String("out", "", "dir to write one capture JSON per object; if omitted, prints to stdout")
	flag.Parse()

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

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
		capture := newCapturingClient()
		conn, err := newConnector(ctx, provider, func(real common.AuthenticatedHTTPClient) common.AuthenticatedHTTPClient {
			capture.inner = real
			return capture
		})
		if err != nil {
			// Faithful: connector construction failed (bad registration/creds) — record it.
			record := &captureRecord{Object: object, Result: resultCapture{Status: "error", Error: err.Error()}}
			emit(*outFlag, object, record)
			continue
		}

		rc, ok := conn.(connectors.ReadConnector)
		if !ok {
			emit(*outFlag, object, &captureRecord{Object: object, Result: resultCapture{Status: "error", Error: "connector does not support Read"}})
			continue
		}

		fields := explicitFields
		if len(fields) == 0 {
			fields = discoverFields(ctx, conn, object)
		}

		record := runOne(ctx, rc, object, since, fields, capture)
		emit(*outFlag, object, record)
	}
}

func emit(outDir, object string, record *captureRecord) {
	if outDir == "" {
		printJSON(record)
		return
	}
	writeRecord(outDir, object, record)
}

// newConnector builds ANY connector generically: pick the client from the
// provider's auth type, and pass the whole creds metadata map so templated vars
// (workspace/region/…) resolve without per-connector field registration.
func newConnector(
	ctx context.Context, provider providers.Provider,
	wrap func(common.AuthenticatedHTTPClient) common.AuthenticatedHTTPClient,
) (connectors.Connector, error) {
	info, err := providers.ReadInfo(provider, catalogreplacer.CustomCatalogVariable{
		Plan: catalogreplacer.SubstitutionPlan{From: catalogreplacer.VariableWorkspace, To: ""},
	})
	if err != nil {
		return nil, fmt.Errorf("ReadInfo(%s): %w", provider, err)
	}

	path := credscanning.LoadPath(provider)
	meta, workspace := readMetadata(path)

	var client common.AuthenticatedHTTPClient
	switch info.AuthType {
	case providers.Oauth2:
		reader := utils.MustCreateProvCredJSON(path, true)
		client = utils.NewOauth2Client(ctx, reader, func(r *credscanning.ProviderCredentials) *oauth2.Config {
			cfg := &oauth2.Config{
				ClientID:     r.Get(credscanning.Fields.ClientId),
				ClientSecret: r.Get(credscanning.Fields.ClientSecret),
			}
			if info.Oauth2Opts != nil {
				cfg.Endpoint = oauth2.Endpoint{
					AuthURL:  info.Oauth2Opts.AuthURL,
					TokenURL: info.Oauth2Opts.TokenURL,
				}
			}
			return cfg
		})
	case providers.Basic:
		reader := utils.MustCreateProvCredJSON(path, false)
		c, berr := common.NewBasicAuthHTTPClient(ctx,
			reader.Get(credscanning.Fields.Username), reader.Get(credscanning.Fields.Password))
		if berr != nil {
			return nil, fmt.Errorf("basic-auth client: %w", berr)
		}
		client = c
	default: // ApiKey (and anything that authenticates via an API key header/query)
		reader := utils.MustCreateProvCredJSON(path, false)
		client = utils.NewAPIKeyClient(ctx, reader, provider)
	}

	if wrap != nil {
		client = wrap(client)
	}

	return connector.New(provider, common.ConnectorParams{
		AuthenticatedClient: client,
		Metadata:            meta,
		Workspace:           workspace,
	})
}

// readMetadata reads the whole `metadata` object from the creds file we write
// (never per-provider field registration). Returns the map + workspace (some
// connectors read ConnectorParams.Workspace directly, others read Metadata).
func readMetadata(path string) (map[string]string, string) {
	meta := map[string]string{}
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return meta, ""
	}
	var raw map[string]any
	if json.Unmarshal(data, &raw) != nil {
		return meta, ""
	}
	if m, ok := raw["metadata"].(map[string]any); ok {
		for k, v := range m {
			meta[k] = fmt.Sprint(v)
		}
	}
	return meta, meta["workspace"]
}

// discoverFields asks the connector for the object's fields (so we don't have to
// be told). Falls back to "id" — a bad guess just yields an honest read error.
func discoverFields(ctx context.Context, conn connectors.Connector, object string) []string {
	mc, ok := conn.(connectors.ObjectMetadataConnector)
	if !ok {
		return []string{"id"}
	}
	md, err := mc.ListObjectMetadata(ctx, []string{object})
	if err != nil || md == nil {
		return []string{"id"}
	}
	om, ok := md.Result[object]
	if !ok {
		return []string{"id"}
	}
	fields := make([]string, 0)
	for name := range om.Fields {
		fields = append(fields, name)
	}
	if len(fields) == 0 {
		for name := range om.FieldsMap {
			fields = append(fields, name)
		}
	}
	if len(fields) == 0 {
		return []string{"id"}
	}
	return fields
}

// ---- capture record + plumbing (identical to the old per-connector binary) ----

type captureRecord struct {
	Object       string           `json:"object"`
	Since        string           `json:"since"`
	Fields       []string         `json:"fields"`
	Request      *requestCapture  `json:"request"`
	Response     *responseCapture `json:"response"`
	Result       resultCapture    `json:"result"`
	Interactions []interaction    `json:"interactions"`
	DurationMs   int64            `json:"duration_ms"`
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
	if len(interactions) > 0 {
		record.Request = &interactions[0].Request
		record.Response = &interactions[0].Response
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
	fileName := strings.ReplaceAll(object, "/", "_") + ".json"
	if err := os.WriteFile(filepath.Join(outDir, fileName), data, 0o644); err != nil { //nolint:gosec
		utils.Fail("error writing capture file", "error", err, "object", object)
	}
}

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

type capturingClient struct {
	inner        common.AuthenticatedHTTPClient
	mu           sync.Mutex
	interactions []interaction
}

func newCapturingClient() *capturingClient { return &capturingClient{} }

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
	c.record(interaction{Request: reqCap, Response: responseCapture{
		Status:  resp.StatusCode,
		Headers: map[string][]string(resp.Header.Clone()),
		Body:    json.RawMessage(bodyBytes),
	}})
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
