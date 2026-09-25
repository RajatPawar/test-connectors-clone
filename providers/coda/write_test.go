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
	responseForbidden := testutils.DataFromFile(t, "write/error-forbidden.json")
	responseBadRequest := testutils.DataFromFile(t, "write/error-bad-request.json")

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
			Name:         "Tables are not writable",
			Input:        common.WriteParams{ObjectName: "tables", RecordData: map[string]any{"name": "x"}},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:         "Rows are not writable",
			Input:        common.WriteParams{ObjectName: "rows", RecordData: map[string]any{"cells": []any{}}},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Forbidden error is surfaced with provider message",
			Input: common.WriteParams{ObjectName: "docs", RecordData: map[string]any{"title": "Project Tracker"}},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrForbidden,
				testutils.StringError("Forbidden"),
			},
		},
		{
			Name: "Bad request error on update",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordId:   "AbCDeFGH",
				RecordData: map[string]any{"title": ""},
			},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusBadRequest, responseBadRequest),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrBadRequest,
				testutils.StringError("Bad Request"),
			},
		},
		{
			Name: "Create doc via POST",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordData: map[string]any{
					"title":    "Project Tracker",
					"timezone": "America/Los_Angeles",
				},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPOST(),
					mockcond.Path("/apis/v1/docs"),
					mockcond.Body(`{"title":"Project Tracker","timezone":"America/Los_Angeles"}`),
				},
				Then: mockserver.Response(http.StatusCreated, responseCreateDoc),
			}.Server(),
			Comparator: testroutines.ComparatorSubsetWrite,
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "AbCDeFGH",
				Errors:   nil,
				Data: map[string]any{
					"id":        "AbCDeFGH",
					"name":      "Project Tracker",
					"type":      "doc",
					"folderId":  "fl-1Ab234",
					"requestId": "abc-123-def-456",
				},
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Update doc via PATCH",
			Input: common.WriteParams{
				ObjectName: "docs",
				RecordId:   "AbCDeFGH",
				RecordData: map[string]any{
					"title":    "Project Tracker v2",
					"iconName": "rocket",
				},
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodPATCH(),
					mockcond.Path("/apis/v1/docs/AbCDeFGH"),
					mockcond.Body(`{"title":"Project Tracker v2","iconName":"rocket"}`),
				},
				Then: mockserver.Response(http.StatusOK, responseUpdateDoc),
			}.Server(),
			Expected: &common.WriteResult{
				Success:  true,
				RecordId: "AbCDeFGH",
				Errors:   nil,
				Data:     map[string]any{},
			},
			ExpectedErrs: nil,
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
