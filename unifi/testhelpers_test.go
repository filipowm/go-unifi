package unifi //nolint: testpackage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
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
//
// The httptest.Server serves each request on its own goroutine, so the request
// log is guarded by mu: the handler appends under the lock and every reader
// (lastRequest/requestsSnapshot/requestCount) reads under it, making the helper
// concurrency-safe by construction (-race clean even when a test drives the
// client from multiple goroutines).
type controllerServer struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
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
		// Record every request (body is fully buffered for later assertions). The
		// append runs on the per-request server goroutine, so guard it with mu.
		body, _ := io.ReadAll(r.Body)
		cs.mu.Lock()
		cs.requests = append(cs.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Body: body})
		cs.mu.Unlock()
		// Restore the body so the matched handler can decode it.
		r.Body = io.NopCloser(bytes.NewReader(body))
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(cs.srv.Close)

	for _, rt := range routes {
		mux.HandleFunc(rt.path, rt.fn)
	}
	return cs
}

// sysinfoRoute serves the new-style sysinfo endpoint with the given controller
// version — the route the client hits behind Version()/GetSystemInformation().
// It is the network-API counterpart of infoRoute (official /v1/info).
func sysinfoRoute(version string) route {
	return route{apiV1Path("s/default/stat/sysinfo"), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"data": [{"version": %q}]}`, version)
	}}
}

// clientConfig returns a ClientConfig pointed at the mock server with
// offline-friendly defaults — pinned new-style API (no network probe), dummy
// creds — then applies opts. It is the shared config half of clientWith and
// newClientWith, so a test only has to state the fields it actually varies.
func (cs *controllerServer) clientConfig(opts ...func(*ClientConfig)) *ClientConfig {
	cfg := &ClientConfig{
		URL:      cs.srv.URL,
		APIKey:   "test-key",
		APIStyle: APIStyleNew,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// clientWith builds a new-style client pointed at the mock server, constructed
// fully offline via the APIStyle override (no network probe, no login), applying
// opts to the config before construction. client() is clientWith with no options.
func (cs *controllerServer) clientWith(opts ...func(*ClientConfig)) *client {
	cs.t.Helper()
	c, err := newClient(cs.clientConfig(opts...))
	require.NoError(cs.t, err)
	return c
}

// newClientWith builds a client through the PUBLIC NewClient constructor pointed
// at the mock server, with the same offline defaults as clientWith, applying opts
// before construction. Unlike clientWith (private newClient), this exercises
// NewClient's eager-sysinfo path, so it returns the (Client, error) pair for the
// caller to assert on (e.g. SkipSystemInfo deferring an error to the first call).
func (cs *controllerServer) newClientWith(opts ...func(*ClientConfig)) (Client, error) {
	cs.t.Helper()
	return NewClient(cs.clientConfig(opts...))
}

// client builds a new-style client pointed at the mock server, constructed fully
// offline via the APIStyle override (no network probe, no login).
func (cs *controllerServer) client() *client {
	cs.t.Helper()
	return cs.clientWith()
}

// newOfflineClient builds a *client from cfg WITHOUT a swallowed construction
// error: it pins the new-style API (skipping the network probe) when the caller
// has not chosen a style, so construction succeeds even against an unreachable URL,
// and asserts no error via require. This replaces the old swallowed-error
// newNewStyleClient foot-gun. Tests that exercise request behavior
// against an unreachable URL still get the later request failure they rely on,
// without silently dropping the construction error.
func newOfflineClient(t *testing.T, cfg *ClientConfig) *client {
	t.Helper()
	if cfg.APIStyle == APIStyleAuto {
		cfg.APIStyle = APIStyleNew
	}
	c, err := newClient(cfg)
	require.NoError(t, err)
	return c
}

// newInterceptedClient builds an offline new-style client pointed at the
// unreachable testUrl and wired with a fresh TestInterceptor, returning both so a
// test can drive a request (which fails at dial after the interceptor has captured
// it) and then assert on what the interceptor captured. opts mutate the config
// before construction (e.g. to set a custom User-Agent or validation mode).
func newInterceptedClient(t *testing.T, opts ...func(*ClientConfig)) (*client, *TestInterceptor) {
	t.Helper()
	interceptor := NewTestInterceptor()
	cfg := &ClientConfig{
		URL:          testUrl,
		APIKey:       "test-key",
		Interceptors: interceptor.AsList(),
		APIStyle:     APIStyleNew,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newOfflineClient(t, cfg), interceptor
}

// lastRequest returns the most recently recorded request, failing the test if
// none was served. Read under mu so it is safe against the handler goroutine.
func (cs *controllerServer) lastRequest() recordedRequest {
	cs.t.Helper()
	cs.mu.Lock()
	defer cs.mu.Unlock()
	require.NotEmpty(cs.t, cs.requests, "expected at least one request to reach the mock controller")
	return cs.requests[len(cs.requests)-1]
}

// requestCount returns the number of requests recorded so far, read under mu.
func (cs *controllerServer) requestCount() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.requests)
}

// countRequestsTo returns how many recorded requests matched path, read under mu.
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
