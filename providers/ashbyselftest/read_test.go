package ashbyselftest

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	candidatePage1 := testutils.DataFromFile(t, "candidate_page1.json")
	candidatePage2 := testutils.DataFromFile(t, "candidate_page2.json")
	jobPostings := testutils.DataFromFile(t, "jobposting_list.json")
	errForbidden := testutils.DataFromFile(t, "error_forbidden.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectCandidate},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Unknown object name is not supported",
			Input: common.ReadParams{ObjectName: "unknown", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/candidate.list"),
				Then:  mockserver.Response(http.StatusOK, candidatePage1),
			}.Server(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider error response is surfaced",
			Input: common.ReadParams{ObjectName: objectCandidate, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/candidate.list"),
				Then:  mockserver.Response(http.StatusForbidden, errForbidden),
			}.Server(),
			ExpectedErrs: []error{common.ErrForbidden},
		},
		{
			Name:  "Read candidates, first page leads to a second page",
			Input: common.ReadParams{ObjectName: objectCandidate, Fields: connectors.Fields("id", "name")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/candidate.list"),
					mockcond.MethodPOST(),
				},
				Then: mockserver.Response(http.StatusOK, candidatePage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "e9ed20fd-d45f-4aad-8a00-a19bfba0083e",
						Fields: map[string]any{"id": "e9ed20fd-d45f-4aad-8a00-a19bfba0083e", "name": "Adam Hart"},
						Raw: map[string]any{
							"id":   "e9ed20fd-d45f-4aad-8a00-a19bfba0083e",
							"name": "Adam Hart",
						},
					},
				},
				NextPage: "G8",
				Done:     false,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "Read candidates, second (last) page",
			Input: common.ReadParams{ObjectName: objectCandidate, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/candidate.list"),
				Then:  mockserver.Response(http.StatusOK, candidatePage2),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "f1a2b3c4-2222-4aad-8a00-a19bfba0083e",
						Fields: map[string]any{"id": "f1a2b3c4-2222-4aad-8a00-a19bfba0083e"},
						Raw:    map[string]any{"id": "f1a2b3c4-2222-4aad-8a00-a19bfba0083e"},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "Read jobPosting: no cursor/pagination fields at all in the response",
			Input: common.ReadParams{ObjectName: objectJobPosting, Fields: connectors.Fields("id", "title")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/jobPosting.list"),
				Then:  mockserver.Response(http.StatusOK, jobPostings),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id: "e9ed20fd-d45f-4aad-8a00-a19bfba0083e",
						Fields: map[string]any{
							"id":    "e9ed20fd-d45f-4aad-8a00-a19bfba0083e",
							"title": "Senior Backend Engineer",
						},
						Raw: map[string]any{
							"id":    "e9ed20fd-d45f-4aad-8a00-a19bfba0083e",
							"title": "Senior Backend Engineer",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			// jobPosting.list has no server-side time filter at all, so this
			// exercises the client-side Since filter in parse.go against the
			// record's own updatedAt (2024-02-20T12:00:00.000Z).
			Name: "Read jobPosting: client-side Since filter drops a stale record",
			Input: common.ReadParams{
				ObjectName: objectJobPosting,
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/jobPosting.list"),
				Then:  mockserver.Response(http.StatusOK, jobPostings),
			}.Server(),
			Expected: &common.ReadResult{
				Rows:     0,
				Data:     []common.ReadResultRow{},
				NextPage: "",
				Done:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.ReadConnector, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
