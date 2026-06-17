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
	"github.com/amp-labs/connectors/internal/datautils"
	tipaltitest "github.com/amp-labs/connectors/test/tipalti"
	"github.com/amp-labs/connectors/test/utils"
)

type objectCapture struct {
	Object      string           `json:"object"`
	Since       string           `json:"since,omitempty"`
	Fields      []string         `json:"fields,omitempty"`
	Result      objectResult     `json:"result"`
	DurationMS  int64            `json:"duration_ms"`
}

type objectResult struct {
	Status      string           `json:"status"`
	Error       string           `json:"error,omitempty"`
	RecordCount int64            `json:"record_count"`
	NextPage    string           `json:"next_page,omitempty"`
	Records     []map[string]any `json:"records"`
}

// allObjects lists every readable object supported by the Tipalti connector.
var allObjects = []string{
	"custom-fields",
	"gl-accounts",
	"invoices",
	"payees",
	"payer-entities",
	"payment-terms",
	"payments",
	"tax-codes",
}

func main() {
	objectFlag := flag.String("object", "", "Single object to read (default: all supported objects)")
	sinceFlag := flag.String("since", "", "Time window as Go duration, e.g. 720h")
	fieldsFlag := flag.String("fields", "", "Comma-separated list of fields to request")
	outFlag := flag.String("out", "", "Output directory for capture files (one JSON per object)")
	flag.Parse()

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	conn := tipaltitest.GetConnector(ctx)

	var since time.Time
	if *sinceFlag != "" {
		d, err := time.ParseDuration(*sinceFlag)
		if err != nil {
			slog.Error("invalid -since value", "error", err)
			os.Exit(1)
		}

		since = time.Now().Add(-d)
	}

	fields := connectors.Fields("id")
	if *fieldsFlag != "" {
		parts := strings.Split(*fieldsFlag, ",")
		cleaned := make([]string, 0, len(parts))

		for _, f := range parts {
			if f = strings.TrimSpace(f); f != "" {
				cleaned = append(cleaned, f)
			}
		}

		if len(cleaned) > 0 {
			fields = datautils.NewStringSet(cleaned...)
		}
	}

	objects := allObjects
	if *objectFlag != "" {
		objects = []string{*objectFlag}
	}

	for _, objectName := range objects {
		cap := readObject(ctx, conn, objectName, since, fields)

		if *outFlag != "" {
			if err := os.MkdirAll(*outFlag, 0o750); err != nil {
				slog.Error("failed to create output directory", "error", err)
				os.Exit(1)
			}

			fileName := strings.ReplaceAll(objectName, "/", "_") + ".json"
			filePath := filepath.Join(*outFlag, fileName)

			data, err := json.MarshalIndent(cap, "", "  ")
			if err != nil {
				slog.Error("failed to marshal capture", "object", objectName, "error", err)
				continue
			}

			if err := os.WriteFile(filePath, data, 0o600); err != nil {
				slog.Error("failed to write capture file", "path", filePath, "error", err)
			} else {
				slog.Info("wrote capture", "path", filePath, "records", cap.Result.RecordCount)
			}
		} else {
			data, _ := json.MarshalIndent(cap, "", "  ")
			fmt.Println(string(data))
		}
	}
}

func readObject(
	ctx context.Context,
	conn interface {
		Read(context.Context, common.ReadParams) (*common.ReadResult, error)
	},
	objectName string,
	since time.Time,
	fields datautils.StringSet,
) objectCapture {
	params := common.ReadParams{
		ObjectName: objectName,
		Fields:     fields,
		Since:      since,
	}

	sinceStr := ""
	if !since.IsZero() {
		sinceStr = since.Format(time.RFC3339)
	}

	fieldNames := make([]string, 0, len(fields))
	for f := range fields {
		fieldNames = append(fieldNames, f)
	}

	start := time.Now()
	result, err := conn.Read(ctx, params)
	durationMS := time.Since(start).Milliseconds()

	cap := objectCapture{
		Object:     objectName,
		Since:      sinceStr,
		Fields:     fieldNames,
		DurationMS: durationMS,
	}

	if err != nil {
		cap.Result = objectResult{
			Status:  "error",
			Error:   err.Error(),
			Records: []map[string]any{},
		}

		return cap
	}

	records := make([]map[string]any, 0, len(result.Data))
	for _, row := range result.Data {
		records = append(records, row.Raw)
	}

	nextPage := ""
	if result.NextPage != "" {
		nextPage = result.NextPage.String()
	}

	cap.Result = objectResult{
		Status:      "ok",
		RecordCount: result.Rows,
		NextPage:    nextPage,
		Records:     records,
	}

	return cap
}
