package main

import (
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/nooks"
	"github.com/amp-labs/connectors/test/capturekit"
)

func main() {
	capturekit.Main(providers.Nooks, nooks.SupportedReadObjects())
}
