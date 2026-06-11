package unifi //nolint: testpackage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRequestHelperClient returns a minimal client suitable for unit-testing the
// extracted request helpers, with a noop logger and the default error handler.
func newRequestHelperClient() *client {
	return &client{
		Logger:       &noopLogger{},
		errorHandler: &DefaultResponseErrorHandler{},
	}
}

// TestOverrideHeadersReplacesExisting verifies that a header already set on the
// request (e.g. by an interceptor) is replaced rather than appended to. This
// locks in the behavior touched by dropping the redundant Header.Get guard.
func TestOverrideHeadersReplacesExisting(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	// Simulate a header set by an interceptor.
	req.Header.Set("Content-Type", "application/json")

	headers := http.Header{}
	headers.Set("Content-Type", "multipart/form-data")

	overrideHeaders(req, headers)

	// The interceptor-set value must be replaced, not duplicated.
	assert.Equal(t, []string{"multipart/form-data"}, req.Header.Values("Content-Type"))
}

// TestOverrideHeadersSetsAbsent verifies that overriding a header that is not
// already present simply sets it (Header.Del is a no-op on absent keys).
func TestOverrideHeadersSetsAbsent(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	headers := http.Header{}
	headers.Set("X-Requested-With", "XMLHttpRequest")

	overrideHeaders(req, headers)

	assert.Equal(t, "XMLHttpRequest", req.Header.Get("X-Requested-With"))
}

// TestOverrideHeadersMultiValue verifies that multi-value headers are preserved
// in order, replacing any previously set values for the same key.
func TestOverrideHeadersMultiValue(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	// Pre-existing single value that must be wiped before the multi-value set.
	req.Header.Set("X-Multi", "stale")

	headers := http.Header{
		"X-Multi": {"first", "second"},
	}

	overrideHeaders(req, headers)

	assert.Equal(t, []string{"first", "second"}, req.Header.Values("X-Multi"))
}

// TestOverrideHeadersNilNoop verifies that passing no override headers leaves
// the request headers untouched.
func TestOverrideHeadersNilNoop(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	overrideHeaders(req, nil)

	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

// TestHandleResponseNoBodyNilRespBody verifies the no-decode path when no
// response body is expected (respBody == nil).
func TestHandleResponseNoBodyNilRespBody(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: 42,
		Body:          io.NopCloser(strings.NewReader(`{"data":"ignored"}`)),
	}

	require.NoError(t, c.handleResponse(resp, nil, http.MethodGet, "/test"))
}

// TestHandleResponseZeroContentLength verifies the no-decode path when the
// response declares a zero content length, even though respBody is non-nil.
func TestHandleResponseZeroContentLength(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: 0,
		Body:          io.NopCloser(strings.NewReader("")),
	}

	var out map[string]any
	require.NoError(t, c.handleResponse(resp, &out, http.MethodGet, "/test"))
	assert.Nil(t, out, "respBody must remain untouched when ContentLength is 0")
}

// TestHandleResponseDecodesBody verifies the happy path where a non-empty body
// is decoded into respBody.
func TestHandleResponseDecodesBody(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	payload := `{"name":"unifi"}`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
	}

	var out struct {
		Name string `json:"name"`
	}
	require.NoError(t, c.handleResponse(resp, &out, http.MethodGet, "/test"))
	assert.Equal(t, "unifi", out.Name)
}

// TestHandleResponseDecodeError verifies that a decode failure is wrapped with
// the method and apiPath context.
func TestHandleResponseDecodeError(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	payload := `not-json`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
	}

	var out map[string]any
	err := c.handleResponse(resp, &out, http.MethodPost, "/widgets")
	require.ErrorContains(t, err, "unable to decode body")
	require.ErrorContains(t, err, http.MethodPost)
	require.ErrorContains(t, err, "/widgets")
}

// TestHandleResponseDecodesBodyWithZeroContentLength is the regression:
// a 200 carrying a real JSON body but reporting ContentLength==0 (as a proxy or
// HTTP/2 path can) must still decode into respBody instead of being silently
// skipped. The decode decision is made on the body, not the transport header.
func TestHandleResponseDecodesBodyWithZeroContentLength(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: 0, // header lies; body is non-empty
		Body:          io.NopCloser(strings.NewReader(`{"name":"unifi"}`)),
	}

	var out struct {
		Name string `json:"name"`
	}
	require.NoError(t, c.handleResponse(resp, &out, http.MethodGet, "/test"))
	assert.Equal(t, "unifi", out.Name)
}

// TestHandleResponseChunkedBody verifies the chunked transfer case
// (ContentLength == -1) still decodes correctly after the change.
func TestHandleResponseChunkedBody(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: -1, // chunked
		Body:          io.NopCloser(strings.NewReader(`{"name":"chunked"}`)),
	}

	var out struct {
		Name string `json:"name"`
	}
	require.NoError(t, c.handleResponse(resp, &out, http.MethodGet, "/test"))
	assert.Equal(t, "chunked", out.Name)
}

// TestHandleResponseEmptyBodyNoContent verifies that a genuinely empty body
// (io.EOF on decode) is treated as "no content": no error, respBody untouched.
func TestHandleResponseEmptyBodyNoContent(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: -1, // unknown length, but body is truly empty
		Body:          io.NopCloser(strings.NewReader("")),
	}

	var out map[string]any
	require.NoError(t, c.handleResponse(resp, &out, http.MethodGet, "/test"))
	assert.Nil(t, out, "respBody must remain untouched on a genuinely empty body")
}

// TestHandleResponseMetaRcError is the regression: a 200 carrying
// meta.rc=="error" (a soft application failure) must surface as a *ServerError
// carrying the rc/msg, NOT be swallowed into an empty decode or ErrNotFound.
func TestHandleResponseMetaRcError(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	payload := `{"meta":{"rc":"error","msg":"api.err.InvalidPayload"},"data":[]}`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
	}

	var out struct {
		Meta Meta  `json:"meta"`
		Data []int `json:"data"`
	}
	err := c.handleResponse(resp, &out, http.MethodPost, "/test")
	require.Error(t, err)

	var serverErr *ServerError
	require.ErrorAs(t, err, &serverErr)
	assert.Equal(t, "error", serverErr.ErrorCode)
	assert.Equal(t, "api.err.InvalidPayload", serverErr.Message)
	// A soft 200-rc-error is NOT a 404 and must not satisfy the ErrNotFound sentinel.
	assert.NotErrorIs(t, err, ErrNotFound)
}

// TestHandleResponseMetaRcErrorCapitalMeta pins the centralized soft-error probe's
// reliance on encoding/json case-insensitive key matching. Real v1 envelopes
// (sysinfo, user, sites, …) serialize the wrapper as capital-M "Meta", whereas
// metaEnvelopeError probes with the canonical lowercase `json:"meta"` tag. The probe
// surfaces the soft failure ONLY because encoding/json matches object keys
// case-insensitively; a future exact-case refactor of the probe would silently stop
// catching soft rc:error responses — the exact regression set out to kill.
// This test fails loudly if that reliance ever breaks.
func TestHandleResponseMetaRcErrorCapitalMeta(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	// Capital-M "Meta" — the wire form real v1 endpoints actually emit.
	payload := `{"Meta":{"rc":"error","msg":"api.err.InvalidPayload"},"data":[]}`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
	}

	var out struct {
		Meta Meta  `json:"meta"`
		Data []int `json:"data"`
	}
	err := c.handleResponse(resp, &out, http.MethodPost, "/test")
	require.Error(t, err)

	var serverErr *ServerError
	require.ErrorAs(t, err, &serverErr)
	assert.Equal(t, "error", serverErr.ErrorCode)
	assert.Equal(t, "api.err.InvalidPayload", serverErr.Message)
	assert.NotErrorIs(t, err, ErrNotFound)
}

// TestHandleResponseMetaRcErrorCarriesResponseContext is the fidelity
// regression: the *ServerError surfaced for a soft (HTTP 200) meta.rc=="error"
// must carry the HTTP context (status code, request method, request URL) stamped
// from the response — not render the lossy "Server error (0) for  : <msg>". The
// status must remain 200 (a soft rc:error is NOT a 404), so errors.Is(ErrNotFound)
// stays false.
func TestHandleResponseMetaRcErrorCarriesResponseContext(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	payload := `{"meta":{"rc":"error","msg":"api.err.InvalidPayload"},"data":[]}`
	reqURL, err := url.Parse("https://controller.example/proxy/network/api/s/default/group/user")
	require.NoError(t, err)
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
		Request: &http.Request{
			Method: http.MethodPost,
			URL:    reqURL,
		},
	}

	var out struct {
		Meta Meta  `json:"meta"`
		Data []int `json:"data"`
	}
	err = c.handleResponse(resp, &out, http.MethodPost, "s/default/group/user")
	require.Error(t, err)

	var serverErr *ServerError
	require.ErrorAs(t, err, &serverErr)
	assert.Equal(t, http.StatusOK, serverErr.StatusCode, "soft-error ServerError must carry the 200 status, not zero")
	assert.Equal(t, http.MethodPost, serverErr.RequestMethod, "soft-error ServerError must carry the request method")
	assert.Equal(t, reqURL.String(), serverErr.RequestURL, "soft-error ServerError must carry the request URL")
	// The rendered message must include the HTTP context rather than the lossy
	// "Server error (0) for  : <msg>".
	assert.Contains(t, serverErr.Error(), "(200)")
	assert.Contains(t, serverErr.Error(), reqURL.String())
	// A soft rc:error is NOT a 404.
	assert.NotErrorIs(t, err, ErrNotFound)
}

// TestHandleResponseMetaRcErrorNilRequest guards the resp.Request==nil branch of
// the enrichment: a hand-built response with no Request must not panic and
// must still carry the status code (method/URL stay empty).
func TestHandleResponseMetaRcErrorNilRequest(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	payload := `{"meta":{"rc":"error","msg":"boom"},"data":[]}`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
		// Request intentionally nil.
	}

	var out struct {
		Meta Meta  `json:"meta"`
		Data []int `json:"data"`
	}
	err := c.handleResponse(resp, &out, http.MethodPost, "/test")
	require.Error(t, err)

	var serverErr *ServerError
	require.ErrorAs(t, err, &serverErr)
	assert.Equal(t, http.StatusOK, serverErr.StatusCode)
	assert.Empty(t, serverErr.RequestMethod)
	assert.Empty(t, serverErr.RequestURL)
}

// TestDecodeResponseBodyExceedsCap is the regression: a body larger than
// maxResponseBodySize must surface an explicit "exceeded N bytes" error BEFORE any
// decode attempt — not a silently-truncated body that fails with an opaque JSON
// decode error. The cap is temporarily lowered (and restored via defer) so the
// test stays cheap and the production 64 MiB default is unchanged.
func TestDecodeResponseBodyExceedsCap(t *testing.T) {
	// NOT parallel: it mutates the package-level maxResponseBodySize cap, which
	// every concurrent handleResponse reads. Restored via Cleanup.
	orig := maxResponseBodySize
	maxResponseBodySize = 16
	t.Cleanup(func() { maxResponseBodySize = orig })

	c := newRequestHelperClient()
	// Valid JSON, but longer than the lowered cap so truncation would otherwise
	// produce a JSON decode error rather than the explicit overflow error.
	payload := `{"name":"` + strings.Repeat("x", 64) + `"}`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
	}

	var out map[string]any
	err := c.handleResponse(resp, &out, http.MethodGet, "/big")
	require.Error(t, err)
	require.ErrorContains(t, err, "exceeded 16 bytes")
	// It must be the explicit overflow error, not a downstream JSON decode error.
	assert.NotContains(t, err.Error(), "unable to decode body")
}

// TestHandleResponseMetaRcOk verifies that a 200 with meta.rc=="ok" decodes
// normally and surfaces no error.
func TestHandleResponseMetaRcOk(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	payload := `{"meta":{"rc":"ok"},"data":[{"name":"unifi"}]}`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
	}

	var out struct {
		Meta Meta `json:"meta"`
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, c.handleResponse(resp, &out, http.MethodGet, "/test"))
	require.Len(t, out.Data, 1)
	assert.Equal(t, "unifi", out.Data[0].Name)
}

// TestHandleResponseNoMetaBlock verifies that a v2-style bare body that carries
// NO meta envelope decodes normally — the centralized rc-error check must be
// gated on a meta block actually being present and never fabricate an error.
func TestHandleResponseNoMetaBlock(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	payload := `{"name":"v2-bare","value":42}`
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(strings.NewReader(payload)),
	}

	var out struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	require.NoError(t, c.handleResponse(resp, &out, http.MethodGet, "/test"))
	assert.Equal(t, "v2-bare", out.Name)
	assert.Equal(t, 42, out.Value)
}

// TestHandleResponseMetaRcOkButRespBodyNil verifies the rc-error check is
// skipped entirely when no respBody is expected (respBody == nil) even if the
// body would carry a meta envelope: nothing to decode, nothing to validate.
func TestHandleResponseRespBodyNilSkipsMetaCheck(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: 0,
		Body:          io.NopCloser(strings.NewReader(`{"meta":{"rc":"error","msg":"ignored"}}`)),
	}

	require.NoError(t, c.handleResponse(resp, nil, http.MethodGet, "/test"))
}

// TestMetaErrorSemantics pins the refined Meta.error() gating used by the
// centralized handleResponse rc-error check: rc=="ok" and an
// absent rc (rc=="") both carry no failure; only a non-empty, non-"ok" rc
// surfaces a *ServerError carrying the rc/msg.
func TestMetaErrorSemantics(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		meta     Meta
		wantErr  bool
		wantCode string
		wantMsg  string
	}{
		"rc ok is no error":    {meta: Meta{RC: "ok"}, wantErr: false},
		"empty rc is no error": {meta: Meta{RC: ""}, wantErr: false},
		"rc error surfaces": {
			meta:     Meta{RC: "error", Message: "api.err.Invalid"},
			wantErr:  true,
			wantCode: "error",
			wantMsg:  "api.err.Invalid",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := tc.meta
			err := m.error()
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			var serverErr *ServerError
			require.ErrorAs(t, err, &serverErr)
			assert.Equal(t, tc.wantCode, serverErr.ErrorCode)
			assert.Equal(t, tc.wantMsg, serverErr.Message)
		})
	}
}

// TestApplyRequestInterceptorsError verifies that the first interceptor error
// short-circuits and is returned.
func TestApplyRequestInterceptorsError(t *testing.T) {
	t.Parallel()

	interceptor := NewTestInterceptor()
	interceptor.failOnRequest = true

	c := newRequestHelperClient()
	c.interceptors = []ClientInterceptor{interceptor}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	err = c.applyRequestInterceptors(req)
	require.ErrorContains(t, err, "request interceptor failed")
	assert.True(t, interceptor.IsRequestIntercepted())
}

// runClientGetRequest creates a new offline test client wired with a fresh
// interceptor, performs a GET request against the unreachable testUrl (so the
// round-trip fails after the interceptor has captured the request), asserts that
// an error occurred, and returns the client and its interceptor.
func runClientGetRequest(t *testing.T, path string, data any) (*client, *TestInterceptor) {
	t.Helper()
	c, interceptor := newInterceptedClient(t)
	err := c.Get(context.Background(), path, data, nil)
	require.Error(t, err)
	return c, interceptor
}

// runClientRequest creates a new offline test client wired with a fresh
// interceptor, performs a request with the given method, asserts that an error
// occurred, and returns the client and its interceptor.
func runClientRequest(t *testing.T, method, path string, body any) (*client, *TestInterceptor) {
	t.Helper()
	c, interceptor := newInterceptedClient(t)
	err := c.Do(context.Background(), method, path, body, nil)
	require.Error(t, err)
	return c, interceptor
}

// TestRequestInterceptorBehavior tests the interceptor's behavior in both normal and failing scenarios.
func TestRequestInterceptorBehavior(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		failOnRequest          bool
		expectedErrorSubstring string
		expectRequest          bool
		expectResponse         bool
	}{
		{"Normal interceptor", false, "", true, false},
		{"Failing interceptor", true, "request interceptor failed", true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, interceptor := newInterceptedClient(t)
			interceptor.failOnRequest = tc.failOnRequest
			err := c.Get(context.Background(), "/", nil, nil)
			require.Error(t, err)
			if tc.expectedErrorSubstring != "" {
				require.ErrorContains(t, err, tc.expectedErrorSubstring)
			}
			assert.Equal(t, tc.expectRequest, interceptor.IsRequestIntercepted())
			assert.Equal(t, tc.expectResponse, interceptor.IsResponseIntercepted())
		})
	}
}

func TestProperRequestUrl(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	testCases := []struct {
		path     string
		expected string
	}{
		{"", testUrl + NewStyleAPI.ApiPath},
		{"test", testUrl + NewStyleAPI.ApiPath + "/test"},
		{"test/", testUrl + NewStyleAPI.ApiPath + "/test"},
		{"test/test", testUrl + NewStyleAPI.ApiPath + "/test/test"},
		{"/test/", testUrl + "/test/"},
		{"/test", testUrl + "/test"},
		{"/test/test", testUrl + "/test/test"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			// Use the helper to perform a GET request and capture the interceptor.
			_, interceptor := runClientGetRequest(t, tc.path, nil)
			a.Equal(tc.expected, interceptor.request.URL.String())
		})
	}
}

func TestRequestHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"API Key Header", ApiKeyHeader, "test-key"},
		{"Accept Header", AcceptHeader, "application/json"},
		{"Content-Type Header", ContentTypeHeader, "application/json; charset=utf-8"},
		{"User-Agent Header", UserAgentHeader, defaultUserAgent},
	}

	_, interceptor := runClientGetRequest(t, "/", nil)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, interceptor.RequestHeader(tc.header))
		})
	}
}

func TestRequestSentWithJson(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	data := &TestData{Data: "test"}
	_, interceptor := runClientGetRequest(t, "/", data)
	var body TestData
	err := json.NewDecoder(interceptor.request.Body).Decode(&body)
	require.NoError(t, err)
	a.Equal(data, &body)
}

func TestRequestMethod(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	testCases := []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions, http.MethodHead, http.MethodTrace, http.MethodConnect,
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			_, interceptor := runClientRequest(t, tc, "", nil)
			a.Equal(tc, interceptor.Method())
		})
	}
}

func TestResponseDataHandling(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	reqData := TestData{
		Data: "request",
	}
	cs := newControllerServer(t, route{
		path: apiV1Path("test"),
		fn: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(TestData{Data: "test"})
		},
	})
	c := cs.client()
	var data TestData

	// when
	err := c.Get(context.Background(), "test", reqData, &data)

	// then
	require.NoError(t, err)
	a.Equal("test", data.Data)
}

func TestOverrideUserAgent(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := newInterceptedClient(t, func(cfg *ClientConfig) {
		cfg.UserAgent = "test-agent"
	})

	// when
	err := c.Get(context.Background(), "", nil, nil)

	// then
	require.Error(t, err)
	a.Equal("test-agent", interceptor.RequestHeader(UserAgentHeader))
}

func TestDoInvalidJsonResponse(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For API style determination.
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// When handling the API call, return an invalid JSON.
		if r.URL.Path == NewStyleAPI.ApiPath+"/any" {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("invalid json"))
			if err != nil {
				t.Error(err)
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	srvTransport := ts.Client().Transport
	c, err := newClient(&ClientConfig{
		URL:                      ts.URL,
		APIKey:                   "test-key",
		HttpRoundTripperProvider: func() http.RoundTripper { return srvTransport },
	})
	require.NoError(t, err)

	var result map[string]any
	err = c.Get(context.Background(), "any", nil, &result)
	require.ErrorContains(t, err, "unable to decode body")
}

type failingErrorHandler struct{}

func (f *failingErrorHandler) HandleError(resp *http.Response) error {
	return errors.New("custom error")
}

func TestErrorHandlerCustom(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For API style determination.
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// For the API call.
		if r.URL.Path == NewStyleAPI.ApiPath+"/error" {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"data":"ok"}`))
			if err != nil {
				t.Error(err)
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	srvTransport := ts.Client().Transport
	customErrorHandler := &failingErrorHandler{}
	c, err := newClient(&ClientConfig{
		URL:                      ts.URL,
		APIKey:                   "test-key",
		ErrorHandler:             customErrorHandler,
		HttpRoundTripperProvider: func() http.RoundTripper { return srvTransport },
	})
	require.NoError(t, err)

	var result map[string]any
	err = c.Get(context.Background(), "error", nil, &result)
	require.Error(t, err)
	assert.Equal(t, "custom error", err.Error())
}

func TestCreateRequestURLInvalid(t *testing.T) {
	t.Parallel()
	c := &client{
		baseURL:  &url.URL{Scheme: "http", Host: "localhost"},
		apiPaths: &NewStyleAPI,
	}
	_, err := c.buildRequestURL("://bad-url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestCreateRequestURLAbsolute(t *testing.T) {
	t.Parallel()
	c := &client{
		baseURL:  &url.URL{Scheme: "http", Host: "localhost"},
		apiPaths: &NewStyleAPI,
	}
	reqURL, err := c.buildRequestURL("http://example.com/test")
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/test", reqURL.String())
}

func TestCreateRequestContextTimeout(t *testing.T) {
	t.Parallel()
	c := &client{
		timeout: 100 * time.Millisecond,
	}
	ctx, cancel := c.newRequestContext()
	defer cancel()
	_, ok := ctx.Deadline()
	require.True(t, ok)

	// Wait for the deadline to expire.
	time.Sleep(150 * time.Millisecond)
	select {
	case <-ctx.Done():
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	default:
		t.Error("expected context deadline exceeded")
	}
}

func TestMarshalRequestInvalid(t *testing.T) {
	t.Parallel()
	r, err := marshalRequest(make(chan int))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json")
	assert.Nil(t, r)
}

func TestMarshalRequestValid(t *testing.T) {
	t.Parallel()
	r, err := marshalRequest(map[string]string{"key": "value"})
	require.NoError(t, err)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.JSONEq(t, `{"key":"value"}`, string(data))
}

// TestPatchWrapperSendsPatch asserts the public Client.Patch wrapper issues an
// HTTP PATCH to the expected path and round-trips the response — closing the
// gap left when Patch joined the curated public surface (mirrors the Get/Put pattern).
func TestPatchWrapperSendsPatch(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	var gotMethod string
	cs := newControllerServer(t, route{
		path: apiV1Path("test"),
		fn: func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(TestData{Data: "patched"})
		},
	})
	c := cs.client()

	var data TestData
	err := c.Patch(context.Background(), "test", TestData{Data: "request"}, &data)

	require.NoError(t, err)
	a.Equal(http.MethodPatch, gotMethod, "Patch wrapper must send an HTTP PATCH")
	a.Equal("patched", data.Data)
}
