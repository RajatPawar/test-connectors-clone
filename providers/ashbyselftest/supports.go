package ashbyselftest

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
	"github.com/amp-labs/connectors/internal/datautils"
)

// Object names — copied verbatim from the RPC endpoint paths (e.g. jobPosting
// from POST /jobPosting.list) per CLAUDE.md naming rules.
const (
	objectCandidate         = "candidate"
	objectApplication       = "application"
	objectJob               = "job"
	objectJobPosting        = "jobPosting"
	objectOffer             = "offer"
	objectOpening           = "opening"
	objectUser              = "user"
	objectInterviewSchedule = "interviewSchedule"
)

// readSupport is the complete list of objects this connector can read. It is
// also exposed via SupportedReadObjects for tooling (e.g. the live read/capture
// binary at test/ashbyselftest/read) that needs to iterate every supported
// object.
//
//nolint:gochecknoglobals
var readSupport = []string{
	objectCandidate,
	objectApplication,
	objectJob,
	objectJobPosting,
	objectOffer,
	objectOpening,
	objectUser,
	objectInterviewSchedule,
}

// supportsCursorPagination lists objects whose .list endpoint accepts a
// `cursor`/`limit` body param (per docs). jobPosting.list has no cursor param
// at all — it always returns every posting in a single response.
//
//nolint:gochecknoglobals
var supportsCursorPagination = datautils.NewSet(
	objectCandidate, objectApplication, objectJob, objectOffer, objectOpening,
	objectUser, objectInterviewSchedule,
)

// SupportedReadObjects returns every object name this connector can read.
func SupportedReadObjects() []string {
	return append([]string(nil), readSupport...)
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(readSupport, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
