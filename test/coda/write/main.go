package main

import (
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/test/capturekit"
)

func main() {
	capturekit.MainWrite(providers.Coda, map[string]map[string]any{
		"docs": {"title": "connie-test-coda-write-docs"},
	})
}
