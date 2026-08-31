// Command read is the live-read/capture shim for the ActiveCampaign connector.
// All real work is in the shared capturekit.
package main

import (
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/activecampaign"
	"github.com/amp-labs/connectors/test/capturekit"
)

func main() {
	capturekit.Main(providers.ActiveCampaign, activecampaign.SupportedReadObjects())
}
