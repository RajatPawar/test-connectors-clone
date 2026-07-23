// Command read is the live-read/capture shim for the Sage HR connector.
// All real work is in the shared capturekit.
package main

import (
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/sagehr"
	"github.com/amp-labs/connectors/test/capturekit"
)

func main() {
	capturekit.Main(providers.SageHR, sagehr.SupportedReadObjects())
}
