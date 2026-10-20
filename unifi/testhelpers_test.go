package unifi //nolint: testpackage

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// recordedRequest captures the method, path and body of a request that reached
// the mock controller, so wrapper tests can assert on what the hand-written
// wrapper actually sent over the wire.
type recordedRequest struct {
	Method string
	Path   string
	Body   []byte
}

// route is a single mock endpoint: matched on the full request path (including
// the new-style /proxy/network prefix) and handled by fn.
type route struct {
	path string
	fn   http.HandlerFunc
}

// controllerServer is a mock UniFi controller backed by an httptest.Server. It
// routes requests by URL path, records every request it serves (except the API
// style probe on "/"), and hands out clients pointed at it. Unmatched paths
// return 404 so a wrapper hitting the wrong endpoint fails loudly.
//
// The httptest.Server serves each request on its own goroutine, so the request
// log is guarded by mu.
type controllerServer struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
	requests []recordedRequest
}

func newControllerServer(t *testing.T, routes ...route) *controllerServer {
	t.Helper()
	cs := &controllerServer{t: t}
	mux := http.NewServeMux()
	// The API style probe made while constructing a client; a 200 selects the new-style API.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})
	for _, rt := range routes {
		mux.HandleFunc(rt.path, rt.fn)
	}
	cs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			body, _ := io.ReadAll(r.Body)
			cs.mu.Lock()
			cs.requests = append(cs.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Body: body})
			cs.mu.Unlock()
			// Restore the body so the matched handler can decode it.
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(cs.srv.Close)
	return cs
}

// clientWith builds a client pointed at the mock server, applying opts to the
// config before construction.
func (cs *controllerServer) clientWith(opts ...func(*ClientConfig)) *client {
	cs.t.Helper()
	cfg := &ClientConfig{URL: cs.srv.URL, APIKey: "test-key"}
	for _, opt := range opts {
		opt(cfg)
	}
	c, err := newBareClient(cfg)
	require.NoError(cs.t, err)
	return c
}

// client builds a client pointed at the mock server with default options.
func (cs *controllerServer) client() *client {
	cs.t.Helper()
	return cs.clientWith()
}

// lastRequest returns the most recently recorded request, failing the test if none was served.
func (cs *controllerServer) lastRequest() recordedRequest {
	cs.t.Helper()
	cs.mu.Lock()
	defer cs.mu.Unlock()
	require.NotEmpty(cs.t, cs.requests, "expected at least one request to reach the mock controller")
	return cs.requests[len(cs.requests)-1]
}

// requestCount returns the number of requests recorded so far.
func (cs *controllerServer) requestCount() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.requests)
}

// countRequestsTo returns how many recorded requests matched path.
func (cs *controllerServer) countRequestsTo(path string) int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	n := 0
	for _, r := range cs.requests {
		if r.Path == path {
			n++
		}
	}
	return n
}

// apiV2 returns the full new-style v2 request path for a v2-relative path.
func apiV2(rel string) string {
	return NewStyleAPI.ApiV2Path + "/" + rel
}

// writeJSON marshals v and writes it as the response body.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(v))
}

// staticJSONStatus replies with status and body (as JSON) for every request.
func staticJSONStatus(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// staticJSON replies with body for every request (200 OK).
func staticJSON(body string) http.HandlerFunc {
	return staticJSONStatus(http.StatusOK, body)
}

// echoJSON replies with the request body, the way the v2 API returns the stored object.
func echoJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = w.Write(raw)
}
