package activecampaigncl

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

	contactsPage1 := testutils.DataFromFile(t, "contacts_page1.json")
	contactsPage2 := testutils.DataFromFile(t, "contacts_page2.json")
	errNotFound := testutils.DataFromFile(t, "error_not_found.json")
	dealsSingle := testutils.DataFromFile(t, "deals_single.json")
	accountsPage1 := testutils.DataFromFile(t, "accounts_page1.json")

	tests := []testroutines.Read{
		{
			Name:         "Read object must be included",
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: objectContacts},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:  "Provider error response is surfaced",
			Input: common.ReadParams{ObjectName: objectContacts, Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/contacts"),
				Then:  mockserver.Response(http.StatusNotFound, errNotFound),
			}.Server(),
			ExpectedErrs: []error{common.ErrRetryable},
		},
		{
			Name: "Read contacts (native incremental), first page leads to a second page",
			Input: common.ReadParams{
				ObjectName: objectContacts, Fields: connectors.Fields("id", "email"), PageSize: 1,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/contacts"),
				Then:  mockserver.Response(http.StatusOK, contactsPage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "68",
						Fields: map[string]any{"id": "68", "email": "janedoe@example.com"},
						Raw: map[string]any{
							"id": "68", "email": "janedoe@example.com", "firstName": "John", "lastName": "Doe",
							"cdate": "2017-01-25T23:58:14-06:00", "udate": "2017-01-25T23:58:14-06:00",
						},
					},
				},
				NextPage: testroutines.URLTestServer + "/api/3/contacts?limit=1&offset=1",
				Done:     false,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Read contacts, second (last) page",
			Input: common.ReadParams{
				ObjectName: objectContacts,
				Fields:     connectors.Fields("id"),
				NextPage:   common.NextPageToken(testroutines.URLTestServer + "/api/3/contacts?limit=1&offset=1"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/contacts"),
				Then:  mockserver.Response(http.StatusOK, contactsPage2),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "73",
						Fields: map[string]any{"id": "73"},
						Raw: map[string]any{
							"id": "73", "email": "aaronallen@example.com", "firstName": "Aaron", "lastName": "Allen",
							"cdate": "2017-02-09T12:14:58-06:00", "udate": "2017-02-09T12:14:58-06:00",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "Read deals extracts records under the 'deals' key with id and requested fields",
			Input: common.ReadParams{ObjectName: objectDeals, Fields: connectors.Fields("title", "value")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/deals"),
				Then:  mockserver.Response(http.StatusOK, dealsSingle),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "1",
						Fields: map[string]any{"title": "New deal", "value": "5000"},
						Raw: map[string]any{
							"id": "1", "cdate": "2019-04-29T07:51:31-05:00", "mdate": "2019-04-29T07:51:31-05:00",
							"title": "New deal", "value": "5000", "currency": "usd", "status": "0",
							"stage": "1", "group": "1",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Read accounts (client-side incremental): only records updated after Since are kept",
			Input: common.ReadParams{
				ObjectName: objectAccounts,
				Fields:     connectors.Fields("id", "name"),
				Since:      time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/accounts"),
				Then:  mockserver.Response(http.StatusOK, accountsPage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "2",
						Fields: map[string]any{"id": "2", "name": "Recent Account"},
						Raw: map[string]any{
							"id": "2", "name": "Recent Account",
							"createdTimestamp": "2024-06-01T00:00:00-05:00", "updatedTimestamp": "2024-06-01T00:00:00-05:00",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
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
		Metadata: map[string]string{
			"workspace": "test-company",
		},
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
