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

	responseAccountsFirstPage := testutils.DataFromFile(t, "accounts-first-page.json")
	responseAccountsLastPage := testutils.DataFromFile(t, "accounts-last-page.json")
	responseMembers := testutils.DataFromFile(t, "members.json")
	responseUnauthorized := testutils.DataFromFile(t, "error-unauthorized.json")
	responseRateLimit := testutils.DataFromFile(t, "error-rate-limit.json")

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
			Input:        common.ReadParams{ObjectName: "tasks", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Invalid API key is an access token error",
			Input: common.ReadParams{ObjectName: "accounts", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusUnauthorized, responseUnauthorized),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrAccessToken,
				testutils.StringError("[unauthorized] Invalid API key."),
			},
		},
		{
			Name:  "Unwrapped rate limit body is understood",
			Input: common.ReadParams{ObjectName: "accounts", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusTooManyRequests, responseRateLimit),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrLimitExceeded,
				testutils.StringError("[too_many_requests] Rate limit exceeded. Please try again later."),
			},
		},
		{
			Name: "First page flattens fields and relationships and points to the next offset",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id", "$name", "icp-score", "$owner", "$contact", "updatedAt"),
				PageSize:   2,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/accounts"),
					mockcond.QueryParam("limit", "2"),
					mockcond.QueryParam("offset", "0"),
					versionHeader,
				},
				Then: mockserver.Response(http.StatusOK, responseAccountsFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":        "acc_newest",
						"$name":     "Acme Corp",
						"icp-score": float64(7),
						"$owner":    []any{"mem_1"},
						"$contact":  []any{"con_1", "con_2"},
						"updatedat": "2026-05-03T09:30:00.000Z",
					},
					Raw: map[string]any{
						"id":       "acc_newest",
						"httpLink": "https://crm.lightfield.app/accounts/acc_newest",
					},
					Id: "acc_newest",
				}, {
					Fields: map[string]any{
						"id":        "acc_older",
						"$name":     "Globex",
						"updatedat": nil,
					},
					Raw: map[string]any{
						"id": "acc_older",
					},
					Id: "acc_older",
				}},
				NextPage: testconn.URLTestServer + "/v1/accounts?limit=2&offset=2",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Page reaching totalCount is the last one",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id", "$name"),
				NextPage:   testconn.URLTestServer + "/v1/accounts?limit=2&offset=2",
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
					Fields: map[string]any{"id": "acc_oldest", "$name": "Initech"},
					Raw:    map[string]any{"id": "acc_oldest"},
					Id:     "acc_oldest",
				}},
				NextPage: "",
				Done:     true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Since filters on updatedAt, falling back to createdAt, and keeps paging",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id"),
				PageSize:   2,
				Since:      time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/accounts"),
					mockcond.QueryParam("offset", "0"),
				},
				Then: mockserver.Response(http.StatusOK, responseAccountsFirstPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{"id": "acc_newest"},
					Raw:    map[string]any{"id": "acc_newest"},
					Id:     "acc_newest",
				}},
				NextPage: testconn.URLTestServer + "/v1/accounts?limit=2&offset=2",
				Done:     false,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Until bounds the window inclusively",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("id"),
				NextPage:   testconn.URLTestServer + "/v1/accounts?limit=2&offset=2",
				Since:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, 4, 20, 16, 45, 0, 0, time.UTC),
			},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusOK, responseAccountsLastPage),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{"id": "acc_oldest"},
					Raw:    map[string]any{"id": "acc_oldest"},
					Id:     "acc_oldest",
				}},
				Done: true,
			},
			ExpectedErrs: nil,
		},
		{
			Name: "Members default to the maximum page size",
			Input: common.ReadParams{
				ObjectName: "members",
				Fields:     connectors.Fields("id", "$email", "$name", "$role", "$user"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/members"),
					mockcond.QueryParam("limit", "25"),
					mockcond.QueryParam("offset", "0"),
					versionHeader,
				},
				Then: mockserver.Response(http.StatusOK, responseMembers),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Fields: map[string]any{
						"id":     "mem_1",
						"$email": "jane@example.com",
						"$name":  map[string]any{"firstName": "Jane", "lastName": "Doe"},
						"$role":  "admin",
						"$user":  []any{"usr_1"},
					},
					Raw: map[string]any{
						"id": "mem_1",
						"relationships": map[string]any{
							"$user": map[string]any{"cardinality": "HAS_ONE", "objectType": "user", "values": []any{"usr_1"}},
						},
					},
					Id: "mem_1",
				}},
				Done: true,
			},
			ExpectedErrs: nil,
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
