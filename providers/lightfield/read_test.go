package lightfield

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop,maintidx
	t.Parallel()

	responseAccounts := testutils.DataFromFile(t, "accounts.json")
	responseAccountsLastPage := testutils.DataFromFile(t, "accounts-last-page.json")
	responseMembers := testutils.DataFromFile(t, "members.json")
	responseEmpty := testutils.DataFromFile(t, "empty.json")
	responseForbidden := testutils.DataFromFile(t, "error-forbidden.json")
	responseVersionHeader := testutils.DataFromFile(t, "error-version-header.json")

	versionHeader := mockcond.Header(http.Header{"Lightfield-Version": []string{"2026-03-01"}})

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "accounts"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "notes", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Missing scope is a forbidden error with the provider message",
			Input: common.ReadParams{ObjectName: "contacts", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrForbidden,
				testutils.StringError("API key does not have the required scope."),
			},
		},
		{
			Name:  "Bad request carries the provider error code",
			Input: common.ReadParams{ObjectName: "contacts", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusBadRequest, responseVersionHeader),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrBadRequest,
				testutils.StringError("The Lightfield-Version header is missing or invalid. (bad_request, version_header, Lightfield-Version)"), // nolint:lll
			},
		},
		{
			Name:  "Empty list is done",
			Input: common.ReadParams{ObjectName: "opportunities", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/opportunities"),
					mockcond.QueryParam("limit", "25"),
					versionHeader,
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Expected: &common.ReadResult{
				Rows: 0,
				Data: []common.ReadResultRow{},
				Done: true,
			},
		},
		{
			Name: "First page flattens fields and relationships and returns the next offset",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id", "updatedAt", "$name", "icp-score", "$contact", "$owner"),
				PageSize:   2,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/accounts"),
					mockcond.QueryParam("limit", "2"),
					mockcond.QueryParamsMissing("offset"),
					versionHeader,
				},
				Then: mockserver.Response(http.StatusOK, responseAccounts),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Id: "acc_newest",
					Fields: map[string]any{
						"id":        "acc_newest",
						"updatedat": nil,
						"$name":     "Acme Corp",
						"icp-score": float64(7),
						"$contact":  []any{"con_a", "con_b"},
						"$owner":    []any{"usr_owner1"},
					},
					Raw: map[string]any{
						"id":        "acc_newest",
						"createdAt": "2026-03-10T09:00:00.000Z",
					},
				}, {
					Id: "acc_older",
					Fields: map[string]any{
						"id":        "acc_older",
						"updatedat": "2026-03-12T15:30:00.000Z",
						"$name":     "Globex",
						"icp-score": nil,
						"$contact":  []any{},
						"$owner":    []any{},
					},
					Raw: map[string]any{
						"id":         "acc_older",
						"externalId": "ext-42",
					},
				}},
				NextPage: "2",
				Done:     false,
			},
		},
		{
			Name: "Last page is detected by a short page",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id", "$name"),
				PageSize:   2,
				NextPage:   "2",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/accounts"),
					mockcond.QueryParam("limit", "2"),
					mockcond.QueryParam("offset", "2"),
				},
				Then: mockserver.Response(http.StatusOK, responseAccountsLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "acc_oldest",
					Fields: map[string]any{
						"id":    "acc_oldest",
						"$name": "Initech",
					},
					Raw: map[string]any{
						"id":            "acc_oldest",
						"relationships": map[string]any{},
					},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Page size above the provider maximum is capped",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id"),
				PageSize:   100,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/accounts"),
					mockcond.QueryParam("limit", "25"),
				},
				Then: mockserver.Response(http.StatusOK, responseAccounts),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     2,
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Since keeps records updated after it, using createdAt when updatedAt is null",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id"),
				PageSize:   2,
				Since:      time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/accounts"),
				Then:  mockserver.Response(http.StatusOK, responseAccounts),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "acc_older",
					Fields: map[string]any{"id": "acc_older"},
					Raw:    map[string]any{"id": "acc_older"},
				}},
				NextPage: "2",
				Done:     false,
			},
		},
		{
			Name: "Until keeps records updated before it",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id"),
				PageSize:   2,
				Until:      time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/accounts"),
				Then:  mockserver.Response(http.StatusOK, responseAccounts),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "acc_newest",
					Fields: map[string]any{"id": "acc_newest"},
					Raw:    map[string]any{"id": "acc_newest"},
				}},
				NextPage: "2",
				Done:     false,
			},
		},
		{
			Name:  "Members are read from the members endpoint",
			Input: common.ReadParams{ObjectName: "members", Fields: connectors.Fields("$email", "$name", "$user")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/members"),
					mockcond.QueryParam("limit", "25"),
					versionHeader,
				},
				Then: mockserver.Response(http.StatusOK, responseMembers),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "mem_abc123",
					Fields: map[string]any{
						"$email": "jane@example.com",
						"$name":  map[string]any{"firstName": "Jane", "lastName": "Doe"},
						"$user":  []any{"usr_owner1"},
					},
					Raw: map[string]any{
						"id":       "mem_abc123",
						"httpLink": nil,
						"relationships": map[string]any{
							"$user": map[string]any{
								"cardinality": "HAS_ONE",
								"objectType":  "user",
								"values":      []any{"usr_owner1"},
							},
						},
					},
				}},
				NextPage: "",
				Done:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (testconn.TestableReader, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
