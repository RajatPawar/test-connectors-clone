package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/datautils"
	tc "github.com/amp-labs/connectors/test/otter.ai"
	"github.com/amp-labs/connectors/test/utils"
)

// Each read object → the fields to request for it, taken from ./docs/.
var objects = map[string][]string{
	"conversations":    {"id", "title", "url", "created_at", "abstract_summary", "conf_join_url"},
	"channels":         {"id", "name", "member_count", "discoverability"},
	"channels/members": {"id", "name", "first_name", "last_name", "email"},
	"workspace":        {"id", "name", "member_count", "handle", "type"},
}

type capture struct {
	Object   string         `json:"object"`
	Response map[string]any `json:"response,omitempty"`
	Result   result         `json:"result"`
}

type result struct {
	Status      string           `json:"status"`
	RecordCount int64            `json:"record_count"`
	Records     []map[string]any `json:"records,omitempty"`
	Error       string           `json:"error,omitempty"`
}

func main() {
	out := flag.String("out", ".", "directory to write one JSON file per object")
	flag.Parse()

	ctx := context.Background()
	conn := tc.GetOtterAIConnector(ctx)

	for obj, fields := range objects {
		c := capture{Object: obj}

		res, err := conn.Read(ctx, common.ReadParams{
			ObjectName: obj,
			Fields:     datautils.NewStringSet(fields...),
		})
		if err != nil {
			// One object failing is data, not a reason to stop capturing the rest.
			c.Result = result{Status: "error", Error: err.Error()}
		} else {
			records := make([]map[string]any, 0, len(res.Data))
			raw := make([]any, 0, len(res.Data))

			for _, row := range res.Data {
				records = append(records, row.Fields)
				raw = append(raw, row.Raw)
			}

			c.Result = result{Status: "ok", RecordCount: res.Rows, Records: records}
			c.Response = map[string]any{"body": raw}

			// Exercise pagination: if the first page reports more records, follow
			// the cursor once to prove NextPage actually works end-to-end.
			if res.NextPage != "" {
				res2, err2 := conn.Read(ctx, common.ReadParams{
					ObjectName: obj,
					Fields:     datautils.NewStringSet(fields...),
					NextPage:   res.NextPage,
				})
				if err2 == nil {
					for _, row := range res2.Data {
						records = append(records, row.Fields)
						raw = append(raw, row.Raw)
					}

					c.Result = result{Status: "ok", RecordCount: res.Rows + res2.Rows, Records: records}
					c.Response = map[string]any{"body": raw}
				}
			}
		}

		writeCapture(*out, obj, c)
	}
}

func writeCapture(dir, obj string, c capture) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		utils.Fail("cannot create out dir", "error", err)
	}

	f, err := os.Create(filepath.Join(dir, strings.ReplaceAll(obj, "/", "_")+".json"))
	if err != nil {
		utils.Fail("cannot write capture", "error", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	if err := enc.Encode(c); err != nil {
		utils.Fail("cannot encode capture", "error", err)
	}
}
