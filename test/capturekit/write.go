package capturekit

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/test/utils"
)

// MainWrite is the write shim's entry point. `samples` maps each object the connector writes to
// the record it creates in the live account:
//
//	func main() {
//		capturekit.MainWrite(providers.Acme, map[string]map[string]any{
//			"contacts": {"email": "connie-test@example.com", "name": "Connie Test"},
//		})
//	}
//
// Each object gets ONE create (no RecordId), captured like a read: every request/response, the
// connector's WriteResult, and the created record's id (WriteResult.RecordId) so the record can be
// found — and later deleted — by exactly that id. Nothing is updated or deleted here.
//
// A capture is written as write__<object>.json with "object": "write:<object>", so it sits in the
// same round as the reads without colliding with a read of the same object.
func MainWrite(provider providers.Provider, samples map[string]map[string]any) {
	objectFlag := flag.String("object", "", "single object to write; default: every object in the shim")
	dataFlag := flag.String("data", "", "JSON record to create, overriding the shim's sample (needs -object)")
	outFlag := flag.String("out", "", "dir to write one capture JSON per object; if omitted, prints to stdout")
	flag.Parse()

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	if *objectFlag != "" {
		sample := samples[*objectFlag]
		if *dataFlag != "" {
			sample = map[string]any{}
			if err := json.Unmarshal([]byte(*dataFlag), &sample); err != nil {
				utils.Fail("-data is not a JSON object", "error", err)
			}
		}
		samples = map[string]map[string]any{*objectFlag: sample}
	}

	if *outFlag != "" {
		if err := os.MkdirAll(*outFlag, 0o755); err != nil {
			utils.Fail("error creating output directory", "error", err)
		}
	}

	for object, sample := range samples {
		name := "write:" + object
		file := "write__" + strings.ReplaceAll(object, "/", "_")

		capture := newCapturingClient()
		conn, err := newConnector(ctx, provider, func(real common.AuthenticatedHTTPClient) common.AuthenticatedHTTPClient {
			capture.inner = real
			return capture
		})
		if err != nil {
			emitWrite(*outFlag, file, &writeCapture{Object: name, Result: writeResult{Status: "error", Error: err.Error()}})
			continue
		}

		wc, ok := conn.(connectors.WriteConnector)
		if !ok {
			emitWrite(*outFlag, file, &writeCapture{Object: name, Result: writeResult{Status: "error", Error: "connector does not support Write"}})
			continue
		}

		emitWrite(*outFlag, file, writeOne(ctx, wc, object, sample, capture))
	}
}

type writeCapture struct {
	Object       string           `json:"object"`
	RecordData   map[string]any   `json:"record_data"`
	Request      *requestCapture  `json:"request"`
	Response     *responseCapture `json:"response"`
	Result       writeResult      `json:"result"`
	Interactions []interaction    `json:"interactions"`
	DurationMs   int64            `json:"duration_ms"`
}

type writeResult struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	// RecordId is the id of the record this write CREATED — the only record a cleanup may delete.
	RecordId string         `json:"record_id,omitempty"`
	Success  bool           `json:"success"`
	Errors   []any          `json:"errors,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
	// RecordCount mirrors a read's field so the journal and the ingest count a write the same way.
	RecordCount int `json:"record_count"`
}

func writeOne(
	ctx context.Context, conn connectors.WriteConnector, object string, sample map[string]any,
	capture *capturingClient,
) *writeCapture {
	start := time.Now()
	result, err := conn.Write(ctx, common.WriteParams{ObjectName: object, RecordData: sample})
	interactions := capture.drain()

	record := &writeCapture{
		Object:       "write:" + object,
		RecordData:   sample,
		Interactions: interactions,
		DurationMs:   time.Since(start).Milliseconds(),
	}
	if len(interactions) > 0 {
		last := interactions[len(interactions)-1]
		record.Request = &last.Request
		record.Response = &last.Response
	}

	switch {
	case err != nil:
		record.Result = writeResult{Status: "error", Error: err.Error()}
	case result == nil:
		record.Result = writeResult{Status: "error", Error: "Write returned no result and no error"}
	case !result.Success:
		record.Result = writeResult{
			Status: "error", Error: "Write reported Success=false",
			RecordId: result.RecordId, Errors: result.Errors, Data: result.Data,
		}
	default:
		record.Result = writeResult{
			Status: "ok", Success: true, RecordId: result.RecordId,
			Errors: result.Errors, Data: result.Data, RecordCount: 1,
		}
	}
	return record
}

func emitWrite(outDir, file string, record *writeCapture) {
	if outDir == "" {
		printJSON(record)
		return
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		utils.Fail("error marshalling JSON", "error", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, file+".json"), data, 0o644); err != nil { //nolint:gosec
		utils.Fail("error writing capture file", "error", err, "object", record.Object)
	}
}
