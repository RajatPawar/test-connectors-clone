package activecampaignoh

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

func TestRead(t *testing.T) { // nolint:funlen
	t.Parallel()

	contactsPage1 := testutils.DataFromFile(t, "contacts_page1.json")
	contactsPage2 := testutils.DataFromFile(t, "contacts_page2.json")
	dealTasksSingle := testutils.DataFromFile(t, "dealtasks_single.json")
	tagsPage1 := testutils.DataFromFile(t, "tags_page1.json")
	errNotFound := testutils.DataFromFile(t, "error_not_found.json")

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
			Name:  "Read contacts, first page leads to a second page via offset",
			Input: common.ReadParams{ObjectName: objectContacts, Fields: connectors.Fields("id", "email")},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					{
						If:   mockcond.And{mockcond.Path("/api/3/contacts"), mockcond.QueryParam("offset", "0")},
						Then: mockserver.Response(http.StatusOK, contactsPage1),
					},
				},
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "68",
						Fields: map[string]any{"id": "68", "email": "janedoe@example.com"},
						Raw: map[string]any{
							"id": "68", "email": "janedoe@example.com",
							"firstName": "John", "lastName": "Doe",
							"cdate": "2017-01-25T23:58:14-06:00", "udate": "2017-01-25T23:58:14-06:00",
						},
					},
					{
						Id:     "73",
						Fields: map[string]any{"id": "73", "email": "aaronallen@example.com"},
						Raw: map[string]any{
							"id": "73", "email": "aaronallen@example.com",
							"firstName": "Aaron", "lastName": "Allen",
							"cdate": "2017-02-09T12:14:58-06:00", "udate": "2017-02-09T12:14:58-06:00",
						},
					},
				},
				NextPage: testroutines.URLTestServer + "/api/3/contacts?limit=100&offset=100",
				Done:     false,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Read contacts, second (last) page — offset+limit exceeds meta.total",
			Input: common.ReadParams{
				ObjectName: objectContacts,
				Fields:     connectors.Fields("id"),
				NextPage:   testroutines.URLTestServer + "/api/3/contacts?limit=100&offset=100",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.And{mockcond.Path("/api/3/contacts"), mockcond.QueryParam("offset", "100")},
				Then:  mockserver.Response(http.StatusOK, contactsPage2),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "90",
						Fields: map[string]any{"id": "90"},
						Raw: map[string]any{
							"id": "90", "email": "mary@example.com",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "Contacts Since sets the native filters[updated_after] query parameter",
			Input: common.ReadParams{
				ObjectName: objectContacts,
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2017, time.February, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/3/contacts"),
					mockcond.QueryParam("filters[updated_after]", "2017-02-01T00:00:00Z"),
				},
				Then: mockserver.Response(http.StatusOK, contactsPage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "68",
						Fields: map[string]any{"id": "68"},
						Raw: map[string]any{
							"id": "68", "email": "janedoe@example.com",
						},
					},
				},
				NextPage: testroutines.URLTestServer +
					"/api/3/contacts?filters%5Bupdated_after%5D=2017-02-01T00%3A00%3A00Z&limit=100&offset=100",
				Done: false,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name: "dealTasks: connector-side filtering by udate drops records older than Since",
			Input: common.ReadParams{
				ObjectName: objectDealTasks,
				Fields:     connectors.Fields("id", "title"),
				Since:      time.Date(2017, time.February, 25, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/dealTasks"),
				Then:  mockserver.Response(http.StatusOK, dealTasksSingle),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{
					{
						Id:     "2",
						Fields: map[string]any{"id": "2", "title": "New Task"},
						Raw: map[string]any{
							"id": "2", "title": "New Task", "status": float64(0),
							"cdate": "2017-03-01T10:00:00-06:00", "udate": "2017-03-01T10:00:00-06:00",
						},
					},
				},
				NextPage: "",
				Done:     true,
			},
			Comparator: testroutines.ComparatorSubsetRead,
		},
		{
			Name:  "tags: full read, no incremental filter available",
			Input: common.ReadParams{ObjectName: objectTags, Fields: connectors.Fields("id", "tag")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/3/tags"),
				Then:  mockserver.Response(http.StatusOK, tagsPage1),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{
					{
						Id:     "1",
						Fields: map[string]any{"id": "1", "tag": "vip"},
						Raw: map[string]any{
							"id": "1", "tag": "vip", "tagType": "contact", "cdate": "2018-01-01T00:00:00-06:00",
						},
					},
					{
						Id:     "2",
						Fields: map[string]any{"id": "2", "tag": "lead"},
						Raw: map[string]any{
							"id": "2", "tag": "lead", "tagType": "contact", "cdate": "2018-02-01T00:00:00-06:00",
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
			"workspace": "test-account",
		},
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
