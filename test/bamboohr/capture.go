package bamboohr

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// HTTPMessage is the JSON-friendly shape of one HTTP request or response, as actually observed
// on the wire (after auth headers are applied).
type HTTPMessage struct {
	Method  string              `json:"method,omitempty"`
	URL     string              `json:"url,omitempty"`
	Status  int                 `json:"status,omitempty"`
	Headers map[string][]string `json:"headers"`
	Query   map[string][]string `json:"query,omitempty"`
	Body    string              `json:"body"`
}

// Interaction is a single, complete HTTP round trip made by the connector.
type Interaction struct {
	Request    HTTPMessage `json:"request"`
	Response   HTTPMessage `json:"response"`
	DurationMs int64       `json:"duration_ms"`
}

// InteractionRecorder is called once per completed HTTP round trip.
type InteractionRecorder func(Interaction)

// capturingTransport is an http.RoundTripper that records every request/response pair it
// forwards, then hands an untouched, fully-buffered response back up the chain so the real
// read logic is unaffected.
type capturingTransport struct {
	base          http.RoundTripper
	onInteraction InteractionRecorder
}

func (t *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var reqBody []byte

	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body) //nolint:errcheck
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	start := time.Now()

	resp, err := t.base.RoundTrip(req)

	duration := time.Since(start)
	if err != nil {
		return resp, err
	}

	var respBody []byte

	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body) //nolint:errcheck
		resp.Body.Close()                   //nolint:errcheck

		resp.Body = io.NopCloser(bytes.NewReader(respBody))
	}

	if t.onInteraction != nil {
		t.onInteraction(Interaction{
			Request: HTTPMessage{
				Method:  req.Method,
				URL:     req.URL.String(),
				Headers: map[string][]string(req.Header),
				Query:   map[string][]string(req.URL.Query()),
				Body:    string(reqBody),
			},
			Response: HTTPMessage{
				Status:  resp.StatusCode,
				Headers: map[string][]string(resp.Header),
				Body:    string(respBody),
			},
			DurationMs: duration.Milliseconds(),
		})
	}

	return resp, nil
}
