package coda

import (
	"net/http"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testroutines"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestWrite(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	responseCreateDoc := testutils.DataFromFile(t, "write/docs-create.json")
	responseUpdateDoc := testutils.DataFromFile(t, "write/docs-update.json")
	responseForbidden := testutils.DataFromFile(t, "write/forbidden.json")

	tests := []testroutines.Write{
		{
			Name:         "Write object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "Write needs data payload",
			Input:        common.WriteParams{ObjectName: "docs"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingRecordData},
		},
		{
			Name:         "Rows are not writable",
			Input:        common.WriteParams{ObjectName: "rows", RecordData: map[string]any{"a": "b"}},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:         "Tables are not writable",
			Input:        common.WriteParams{ObjectName: "tables", RecordData: map[string]any{"a": "b"}},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider error is surfaced",
			Input: common.WriteParams{ObjectName: "docs", RecordData: map[string]any{"title": "x"}},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrForbidden,
				testutils.StringError("The API token does not grant access to this resource."),
			},
		},
		{
			Name: "Create doc via POST",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordData: map[string]any{"title": "Project Tracker"},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPOST(),
					mockcond.Path("/apis/v1/docs"),
					mockcond.Body(`{"title":"Project Tracker"}`),
				},
				Then: mockserver.Response(http.StatusCreated, responseCreateDoc),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetWrite,
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "AbCDeFGH",
				Data: map[string]any{
					"id":   "AbCDeFGH",
					"name": "Project Tracker",
				},
			},
		},
		{
			Name: "Update doc via PATCH",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordId:   "AbCDeFGH",
				RecordData: map[string]any{"title": "Renamed", "iconName": "rocket"},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPATCH(),
					mockcond.Path("/apis/v1/docs/AbCDeFGH"),
					mockcond.Body(`{"title":"Renamed","iconName":"rocket"}`),
				},
				Then: mockserver.Response(http.StatusOK, responseUpdateDoc),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetWrite,
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "AbCDeFGH",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (connectors.WriteConnector, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}
