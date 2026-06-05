package unifi //nolint: testpackage

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// recordedRequest captures the method, path and decoded body of a request that
// reached the mock controller, so wrapper tests can assert on what the
// hand-written wrapper actually sent over the wire.
type recordedRequest struct {
	Method string
	Path   string
	Body   []byte
}

// controllerServer is a mock UniFi controller backed by an httptest.Server. It
// routes requests by URL path to a handler, records every request it serves, and
// exposes a client pinned (via the APIStyle seam) to that server so the whole
// thing is constructed fully offline — no live controller, no network probe.
type controllerServer struct {
	t        *testing.T
	srv      *httptest.Server
	requests []recordedRequest
}

// route is a single mock endpoint: matched on the request URL path (already
// including the new-style /proxy/network/api prefix) and handled by fn.
type route struct {
	path string
	fn   http.HandlerFunc
}

// newControllerServer spins up a mock controller serving the given routes. Each
// route's path is the FULL request path as seen on the wire (e.g.
// NewStyleAPI.ApiPath+"/s/default/rest/user"). Unmatched paths return 404 so a
// wrapper hitting the wrong endpoint fails loudly. The server is closed via
// t.Cleanup.
func newControllerServer(t *testing.T, routes ...route) *controllerServer {
	t.Helper()
	cs := &controllerServer{t: t}
	mux := http.NewServeMux()
	cs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record every request (body is fully buffered for later assertions).
		body, _ := io.ReadAll(r.Body)
		cs.requests = append(cs.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Body: body})
		// Restore the body so the matched handler can decode it.
		r.Body = io.NopCloser(bytes.NewReader(body))
		w.Header().Set(CsrfHeader, "csrf-token")
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(cs.srv.Close)

	for _, rt := range routes {
		mux.HandleFunc(rt.path, rt.fn)
	}
	return cs
}

// client builds a new-style client pointed at the mock server, constructed fully
// offline via the APIStyle override (no network probe, no login).
func (cs *controllerServer) client() *client {
	cs.t.Helper()
	c, err := newBareClient(&ClientConfig{
		URL:      cs.srv.URL,
		APIKey:   "test-key",
		APIStyle: APIStyleNew,
	})
	require.NoError(cs.t, err)
	return c
}

// lastRequest returns the most recently recorded request, failing the test if
// none was served.
func (cs *controllerServer) lastRequest() recordedRequest {
	cs.t.Helper()
	require.NotEmpty(cs.t, cs.requests, "expected at least one request to reach the mock controller")
	return cs.requests[len(cs.requests)-1]
}

// apiPath returns the full new-style request path for a controller-relative path
// (the same join the client performs), e.g. "s/default/rest/user" ->
// "/proxy/network/api/s/default/rest/user".
func apiV1Path(rel string) string {
	return NewStyleAPI.ApiPath + "/" + rel
}

// apiV2 returns the full new-style v2 request path for a v2-relative path.
func apiV2(rel string) string {
	return NewStyleAPI.ApiV2Path + "/" + rel
}
