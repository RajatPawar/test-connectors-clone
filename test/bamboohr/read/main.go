package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	bamboohrconn "github.com/amp-labs/connectors/providers/bamboohr"
	testconn "github.com/amp-labs/connectors/test/bamboohr"
	"github.com/amp-labs/connectors/test/utils"
)

// supportedObjects is the canonical list of objects this connector exposes.
var supportedObjects = []string{
	"api/v1/employees",
	"webhooks",
	"fields",
	"api/v1/meta/timezones",
	"custom-reports",
	"api/v1_2/datasets",
}

func main() {
	var (
		objectFlag = flag.String("object", "", "Single object to read (e.g. webhooks). If empty, all objects are read.")
		sinceFlag  = flag.Duration("since", 0, "Incremental read window (Go duration, e.g. 720h). Not supported by BambooHR list endpoints — kept for API compatibility.")
		fieldsFlag = flag.String("fields", "", "Comma-separated list of fields to request. Defaults to id for each object.")
		outFlag    = flag.String("out", "", "Output directory for capture mode. Each object writes <object>.json. If empty, prints to stdout.")
	)

	flag.Parse()

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	conn := testconn.GetBambooHRConnector(ctx)

	objects := supportedObjects
	if *objectFlag != "" {
		objects = []string{*objectFlag}
	}

	var fields []string
	if *fieldsFlag != "" {
		fields = strings.Split(*fieldsFlag, ",")
	}

	var since time.Time
	if *sinceFlag > 0 {
		since = time.Now().Add(-*sinceFlag)
	}

	_ = since // BambooHR list endpoints do not support since filter

	for _, obj := range objects {
		result, err := readObject(ctx, conn, obj, fields)
		if err != nil {
			slog.Error("read failed", "object", obj, "error", err)
			continue
		}

		if *outFlag != "" {
			if err := writeCapture(*outFlag, obj, result); err != nil {
				slog.Error("write capture failed", "object", obj, "error", err)
			}
		} else {
			utils.DumpJSON(result, os.Stdout)
		}
	}
}

func readObject(ctx context.Context, conn *bamboohrconn.Connector, objectName string, fields []string) (*common.ReadResult, error) {
	fieldSet := connectors.Fields("id")
	if len(fields) > 0 {
		fieldSet = connectors.Fields(fields...)
	}

	return conn.Read(ctx, common.ReadParams{
		ObjectName: objectName,
		Fields:     fieldSet,
	})
}

type captureEntry struct {
	Object      string                 `json:"object"`
	RecordCount int64                  `json:"record_count"`
	Records     []common.ReadResultRow `json:"records"`
	NextPage    string                 `json:"next_page,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

func writeCapture(outDir, objectName string, result *common.ReadResult) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Replace slashes with underscores in filename to avoid nested directories.
	safeName := strings.ReplaceAll(objectName, "/", "_")
	filePath := filepath.Join(outDir, safeName+".json")

	entry := captureEntry{
		Object:      objectName,
		RecordCount: result.Rows,
		Records:     result.Data,
		NextPage:    result.NextPage.String(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	slog.Info("captured", "object", objectName, "records", entry.RecordCount, "file", filePath)

	return nil
}
