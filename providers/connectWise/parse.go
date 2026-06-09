package connectWise

import (
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/httpkit"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// getRecords reads the top-level JSON array that ConnectWise list endpoints return.
func getRecords(node *ajson.Node) ([]map[string]any, error) {
	arr, err := jsonquery.New(node).ArrayRequired("")
	if err != nil {
		return nil, err
	}

	return jsonquery.Convertor.ArrayToMap(arr)
}

// nextRecordsURL follows the RFC 5988 Link header (rel="next") that ConnectWise
// returns for navigable pagination. The absence of a `next` link ends paging.
func nextRecordsURL(headers http.Header) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		return httpkit.HeaderLink(&common.JSONHTTPResponse{Headers: headers}, "next"), nil
	}
}
