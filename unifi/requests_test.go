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

func newTestClientWithInterceptor() (*client, *TestInterceptor) {
	interceptor := NewTestInterceptor()
	c := newNewStyleClient(&ClientConfig{
		URL:          testUrl,
		APIKey:       "test-key",
		Interceptors: interceptor.AsList(),
	})
	return c, interceptor
}

// runClientGetRequest creates a new test client, performs a GET request,
// asserts that an error occurred, and returns the client and its interceptor.
func runClientGetRequest(t *testing.T, path string, data any) (*client, *TestInterceptor) {
	t.Helper()
	c, interceptor := newTestClientWithInterceptor()
	err := c.Get(context.Background(), path, data, nil)
	require.Error(t, err)
	return c, interceptor
}

// runClientRequest creates a new test client, performs a request with the given method,
// asserts that an error occurred, and returns the client and its interceptor.
func runClientRequest(t *testing.T, method, path string, body any) (*client, *TestInterceptor) {
	t.Helper()
	c, interceptor := newTestClientWithInterceptor()
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
			c, interceptor := newTestClientWithInterceptor()
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
	srv := runTestServer(NewStyleAPI.ApiPath + "/test")
	c := newNewStyleClient(&ClientConfig{
		URL:    srv.URL,
		APIKey: "test-key",
	})
	var data TestData

	// when
	err := c.Get(context.Background(), "test", reqData, &data)

	// then
	require.NoError(t, err)
	a.Equal("test", data.Data)
}

func TestCsrfHandling(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	srv := runTestServer("")
	interceptor := NewTestInterceptor()
	c := newNewStyleClient(&ClientConfig{
		URL:          srv.URL,
		User:         "test-user",
		Password:     "test-pass",
		Interceptors: interceptor.AsList(),
	})

	// when
	err := c.Get(context.Background(), "", nil, nil)

	// then
	require.Error(t, err)
	a.Empty(interceptor.RequestHeader(CsrfHeader))
	a.Equal("csrf-token", interceptor.ResponseHeader(CsrfHeader))

	// when
	err = c.Get(context.Background(), "", nil, nil)

	// then
	require.Error(t, err)
	a.Equal("csrf-token", interceptor.RequestHeader(CsrfHeader))
}

func TestOverrideUserAgent(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	interceptor := NewTestInterceptor()
	c := newNewStyleClient(&ClientConfig{
		URL:          testUrl,
		APIKey:       "test-key",
		Interceptors: interceptor.AsList(),
		UserAgent:    "test-agent",
	})

	// when
	err := c.Get(context.Background(), "", nil, nil)

	// then
	require.Error(t, err)
	a.Equal("test-agent", interceptor.RequestHeader(UserAgentHeader))
}

func TestDoInvalidJsonResponse(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	c, err := newBareClient(&ClientConfig{
		URL:    ts.URL,
		APIKey: "test-key",
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	customErrorHandler := &failingErrorHandler{}
	c, err := newBareClient(&ClientConfig{
		URL:          ts.URL,
		APIKey:       "test-key",
		ErrorHandler: customErrorHandler,
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
