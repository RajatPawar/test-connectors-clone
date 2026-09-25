package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/coda"
	"github.com/amp-labs/connectors/test/utils"
)

// Creates one test doc in the connected Coda account and prints the WriteResult.
// The API token owner must be a Doc Maker in the workspace (Coda "Free and Paid Workspaces" docs).
func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()

	utils.SetupLogging()

	conn := coda.GetCodaConnector(ctx)

	res, err := conn.Write(ctx, common.WriteParams{
		ObjectName: "docs",
		RecordData: map[string]any{
			"title": fmt.Sprintf("connie-test-doc-%d", time.Now().Unix()),
		},
	})
	if err != nil {
		utils.Fail("error creating doc", "error", err)
	}

	slog.Info("created doc", "recordId", res.RecordId)
	utils.DumpJSON(res, os.Stdout)
}
