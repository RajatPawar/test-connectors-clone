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
	connectwiseconn "github.com/amp-labs/connectors/providers/connectwise"
	conn "github.com/amp-labs/connectors/test/connectwise"
	"github.com/amp-labs/connectors/test/utils"
)

func main() {
	// Handle Ctrl-C gracefully.
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	objectFlag := flag.String("object", "", "single object name to read (omit to read all)")
	sinceFlag := flag.Duration("since", 0, "lookback duration for incremental read (e.g. 720h)")
	fieldsFlag := flag.String("fields", "", "comma-separated field names to return")
	outFlag := flag.String("out", "", "output directory; writes <object>.json per object (stdout if omitted)")
	flag.Parse()

	c := conn.GetConnectWiseConnector(ctx)

	var since time.Time
	if *sinceFlag > 0 {
		since = time.Now().Add(-*sinceFlag)
	}

	var fieldList []string
	if *fieldsFlag != "" {
		fieldList = strings.Split(*fieldsFlag, ",")
	}

	objects := objectsToRead(c, *objectFlag)

	for _, obj := range objects {
		if err := readObject(ctx, c, obj, since, fieldList, *outFlag); err != nil {
			slog.Error("read failed", "object", obj, "error", err)
		}
	}
}

func objectsToRead(c *connectwiseconn.Connector, object string) []string {
	if object != "" {
		return []string{object}
	}

	// Return all supported objects.
	return []string{
		"service/tickets",
		"companies",
		"contacts",
		"projects",
		"activities",
		"time/entries",
		"invoices",
		"schedule/entries",
		"opportunities",
	}
}

type captureResult struct {
	Object      string           `json:"object"`
	Since       string           `json:"since,omitempty"`
	Fields      []string         `json:"fields,omitempty"`
	Status      string           `json:"status"`
	Error       string           `json:"error,omitempty"`
	RecordCount int              `json:"record_count"`
	NextPage    string           `json:"next_page,omitempty"`
	Records     []map[string]any `json:"records"`
}

func readObject(
	ctx context.Context,
	c *connectwiseconn.Connector,
	objectName string,
	since time.Time,
	fields []string,
	outDir string,
) error {
	params := connectors.ReadParams{
		ObjectName: objectName,
	}

	if !since.IsZero() {
		params.Since = since
	}

	if len(fields) > 0 {
		params.Fields = connectors.Fields(fields...)
	} else {
		params.Fields = connectors.Fields("id")
	}

	result, err := c.Read(ctx, params)

	capture := captureResult{
		Object:  objectName,
		Records: make([]map[string]any, 0),
	}

	if !since.IsZero() {
		capture.Since = since.Format(time.RFC3339)
	}

	capture.Fields = fields

	if err != nil {
		capture.Status = "error"
		capture.Error = err.Error()
	} else {
		capture.Status = "ok"
		capture.RecordCount = len(result.Data)

		if result.NextPage != "" {
			capture.NextPage = result.NextPage.String()
		}

		for _, row := range result.Data {
			capture.Records = append(capture.Records, row.Raw)
		}
	}

	jsonBytes, jsonErr := json.MarshalIndent(capture, "", "  ")
	if jsonErr != nil {
		return fmt.Errorf("json marshal: %w", jsonErr)
	}

	if outDir != "" {
		// Sanitize object name for filesystem use (replace / with _).
		fileName := strings.ReplaceAll(objectName, "/", "_") + ".json"
		path := filepath.Join(outDir, fileName)

		if writeErr := os.WriteFile(path, jsonBytes, 0o600); writeErr != nil {
			return fmt.Errorf("write file %s: %w", path, writeErr)
		}

		slog.Info("captured", "object", objectName, "path", path, "records", capture.RecordCount)
	} else {
		_, _ = os.Stdout.Write(jsonBytes)
		_, _ = os.Stdout.WriteString("\n")
	}

	return nil
}
