package coda

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestWrite(t *testing.T) { //nolint:funlen
	t.Parallel()

	createDoc := testutils.DataFromFile(t, "write/docs-create.json")
	badRequest := testutils.DataFromFile(t, "write/error-bad-request.json")

	tests := []testconn.TestCaseWrite{
		{
			Name:         "Write object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "RecordData is required",
			Input:        common.WriteParams{ObjectName: "docs"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingRecordData},
		},
		{
			Name: "Tables are not writable",
			Input: common.WriteParams{
				ObjectName: "tables",
				RecordData: map[string]any{"name": "x"},
			},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name: "Create doc",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordData: map[string]any{"title": "Project Tracker", "timezone": "America/Los_Angeles"},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPOST(),
					mockcond.Path("/apis/v1/docs"),
					mockcond.Body(`{"timezone":"America/Los_Angeles","title":"Project Tracker"}`),
				},
				Then: mockserver.Response(http.StatusCreated, createDoc),
			}.Server(),
			Comparator: testconn.ComparatorSubsetWrite,
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "AbCDeFGH",
				Data: map[string]any{
					"id":        "AbCDeFGH",
					"name":      "Project Tracker",
					"requestId": "abc-123-def-456",
				},
			},
		},
		{
			// PATCH /docs/{docId} responds 200 with `{}` (DocUpdateResult), which carries no id.
			Name: "Update doc uses PATCH /docs/{docId} and RecordId falls back to the updated id",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordId:   "QrStUvWx",
				RecordData: map[string]any{"title": "Renamed", "iconName": "rocket"},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPATCH(),
					mockcond.Path("/apis/v1/docs/QrStUvWx"),
					mockcond.Body(`{"iconName":"rocket","title":"Renamed"}`),
				},
				Then: mockserver.Response(http.StatusOK, []byte(`{}`)),
			}.Server(),
			Comparator: testconn.ComparatorSubsetWrite,
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "QrStUvWx",
				Data:     map[string]any{},
			},
		},
		{
			Name: "Provider error is returned",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordData: map[string]any{"title": "Project Tracker"},
			},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusBadRequest, badRequest),
			}.Server(),
			ExpectedErrs: []error{common.ErrCaller},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (testconn.TestableWriter, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}
