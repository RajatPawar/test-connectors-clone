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

func TestRead(t *testing.T) { // nolint:funlen,gocognit,cyclop
	t.Parallel()

	responseAccounts := testutils.DataFromFile(t, "accounts.json")
	responseAccountsLastPage := testutils.DataFromFile(t, "accounts-last-page.json")
	responseMembers := testutils.DataFromFile(t, "members.json")
	responseForbidden := testutils.DataFromFile(t, "error-forbidden.json")

	versionHeader := mockcond.Header(http.Header{"Lightfield-Version": []string{"2026-03-01"}})

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
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
			Input:        common.ReadParams{ObjectName: "meetings", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider error is interpreted",
			Input: common.ReadParams{ObjectName: "accounts", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusForbidden, responseForbidden),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrForbidden,
				testutils.StringError("API key does not have the required scope 'accounts:read'."),
			},
		},
		{
			Name: "First page flattens fields and relationships and returns the next offset",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("$name", "icp-score", "$owner", "$contact", "updatedAt"),
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
						"$name":     "Acme Corp",
						"icp-score": float64(7),
						"$owner":    []any{"usr_owner1"},
						"$contact":  []any{"con_def456", "con_ghi789"},
						"updatedat": nil,
					},
					Raw: map[string]any{
						"id":         "acc_newest",
						"externalId": nil,
						"relationships": map[string]any{
							"$owner": map[string]any{
								"cardinality": "HAS_ONE",
								"objectType":  "user",
								"values":      []any{"usr_owner1"},
							},
							"$contact": map[string]any{
								"cardinality": "HAS_MANY",
								"objectType":  "contact",
								"values":      []any{"con_def456", "con_ghi789"},
							},
						},
					},
				}, {
					Id: "acc_older",
					Fields: map[string]any{
						"$name":     "Globex",
						"icp-score": nil,
						"$owner":    []any{},
						"updatedat": "2026-03-01T08:30:00.000Z",
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
			Name: "Short page is the last one",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("$name"),
				PageSize:   2,
				NextPage:   "2",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/accounts"),
					mockcond.QueryParam("limit", "2"),
					mockcond.QueryParam("offset", "2"),
					versionHeader,
				},
				Then: mockserver.Response(http.StatusOK, responseAccountsLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id:     "acc_oldest",
					Fields: map[string]any{"$name": "Initech"},
					Raw:    map[string]any{"id": "acc_oldest"},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Since keeps records updated (or created, when never updated) in the window and keeps paging",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("$name"),
				PageSize:   2,
				Since:      time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
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
					Fields: map[string]any{"$name": "Acme Corp"},
					Raw:    map[string]any{"id": "acc_newest"},
				}},
				NextPage: "2",
				Done:     false,
			},
		},
		{
			Name: "Until excludes records modified after it and an empty filtered page still paginates",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("$name"),
				PageSize:   2,
				Until:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/accounts"),
				Then:  mockserver.Response(http.StatusOK, responseAccounts),
			}.Server(),
			Expected: &common.ReadResult{
				Rows:     0,
				Data:     []common.ReadResultRow{},
				NextPage: "2",
				Done:     false,
			},
		},
		{
			Name: "Page size is capped at 25 and members are flattened",
			Input: common.ReadParams{
				ObjectName: "members",
				Fields:     connectors.Fields("$email", "$name", "$role", "$user", "$profileImage"),
				PageSize:   100,
			},
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
						"$email":        "jane@example.com",
						"$name":         map[string]any{"firstName": "Jane", "lastName": "Doe"},
						"$role":         "admin",
						"$user":         []any{"usr_owner1"},
						"$profileimage": nil,
					},
					Raw: map[string]any{"id": "mem_abc123"},
				}},
				Done: true,
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
