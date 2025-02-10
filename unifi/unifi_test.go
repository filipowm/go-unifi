package unifi //nolint: testpackage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	localUrl = "http://127.0.0.1:64431"
	testUrl  = "http://test.url"
)

// verifyInterceptorPresence checks each expected interceptor type for presence or absence in the client.
func verifyInterceptorPresence(a *assert.Assertions, c *Client, interceptors []interface{}, shouldExist bool) {
	expectedTypes := make([]reflect.Type, 0, len(interceptors))
	for _, i := range interceptors {
		expectedTypes = append(expectedTypes, reflect.TypeOf(i))
	}
	for _, et := range expectedTypes {
		found := false
		for _, actual := range c.interceptors {
			if reflect.TypeOf(actual) == et {
				found = true
				break
			}
		}
		if shouldExist && !found {
			a.Fail(fmt.Sprintf("expected interceptor %v not found", et))
		}
		if !shouldExist && found {
			a.Fail(fmt.Sprintf("unexpected interceptor %v found", et))
		}
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	c, err := NewClient(&ClientConfig{
		URL:       localUrl,
		User:      "admin",
		Pass:      "password",
		VerifySSL: false,
	})
	require.Error(t, err)
	a.EqualValues(localUrl, c.BaseURL.String())
	a.Contains(err.Error(), "connection refused", "an invalid destination should produce a connection error.")
	verifyInterceptorPresence(a, c, []interface{}{&CsrfInterceptor{}, &DefaultHeadersInterceptor{}}, true)
	verifyInterceptorPresence(a, c, []interface{}{&ApiKeyAuthInterceptor{}}, false)
}

func TestNewClientWithApiKey(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// when
	c, err := NewClient(&ClientConfig{
		URL:       localUrl,
		APIKey:    "test",
		VerifySSL: false,
	})

	// then
	require.Error(t, err)
	a.EqualValues(localUrl, c.BaseURL.String())
	a.Contains(err.Error(), "connection refused", "an invalid destination should produce a connection error.")
	verifyInterceptorPresence(a, c, []interface{}{&ApiKeyAuthInterceptor{}, &DefaultHeadersInterceptor{}}, true)
	verifyInterceptorPresence(a, c, []interface{}{&CsrfInterceptor{}}, false)
}

func TestCustomizeHttpClient(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	called := false

	// when
	_, err := NewClient(&ClientConfig{
		URL:    localUrl,
		APIKey: "test-key",
		HttpCustomizer: func(transport *http.Transport) error {
			called = true
			return nil
		},
	})

	// then
	require.Error(t, err)
	a.True(called, "http customizer not called")
}

type TestInterceptor struct {
	request       *http.Request
	response      *http.Response
	failOnRequest bool
}

func (i *TestInterceptor) IsRequestIntercepted() bool {
	return i.request != nil
}

func (i *TestInterceptor) IsResponseIntercepted() bool {
	return i.response != nil
}

func (i *TestInterceptor) InterceptRequest(req *http.Request) error {
	i.request = req
	if i.failOnRequest {
		return errors.New("request interceptor failed")
	}
	return nil
}

func (i *TestInterceptor) InterceptResponse(resp *http.Response) error {
	i.response = resp
	return nil
}

func (i *TestInterceptor) RequestHeader(key string) string {
	return i.request.Header.Get(key)
}

func (i *TestInterceptor) ResponseHeader(key string) string {
	return i.response.Header.Get(key)
}

func (i *TestInterceptor) Method() string {
	return i.request.Method
}

func NewTestInterceptor() *TestInterceptor {
	return &TestInterceptor{}
}

func (i *TestInterceptor) AsList() []ClientInterceptor {
	return []ClientInterceptor{i}
}

func NewTestClientWithInterceptor() (*Client, *TestInterceptor) {
	interceptor := NewTestInterceptor()
	c, _ := NewClient(&ClientConfig{
		URL:          testUrl,
		APIKey:       "test-key",
		Interceptors: interceptor.AsList(),
	})
	c.apiPaths = &NewStyleAPI
	return c, interceptor
}

// runClientGetRequest creates a new test client, performs a GET request,
// asserts that an error occurred, and returns the client and its interceptor.
func runClientGetRequest(t *testing.T, path string, data interface{}) (*Client, *TestInterceptor) {
	c, interceptor := NewTestClientWithInterceptor()
	err := c.Get(context.Background(), path, data, nil)
	require.Error(t, err)
	return c, interceptor
}

// runClientRequest creates a new test client, performs a request with the given method,
// asserts that an error occurred, and returns the client and its interceptor.
func runClientRequest(t *testing.T, method, path string, body interface{}) (*Client, *TestInterceptor) {
	c, interceptor := NewTestClientWithInterceptor()
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
			c, interceptor := NewTestClientWithInterceptor()
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
			a.EqualValues(tc.expected, interceptor.request.URL.String())
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
			assert.EqualValues(t, tc.expected, interceptor.RequestHeader(tc.header))
		})
	}
}

type TestData struct {
	Data string `json:"data"`
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
			a.EqualValues(tc, interceptor.Method())
		})
	}
}

func runTestServer(path string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always set the CSRF header on the response.
		w.Header().Set(CsrfHeader, "csrf-token")
		if !strings.EqualFold(r.URL.Path, path) {
			http.NotFound(w, r)
			return
		}

		// Return a JSON response
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TestData{Data: "test"})
	}))
}

func TestUnifiIntegrationUserPassInjected(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	type userPass struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	srv := runTestServer(NewStyleAPI.LoginPath)
	interceptor := NewTestInterceptor()
	c, _ := NewClient(&ClientConfig{
		URL:          srv.URL,
		User:         "test-user",
		Pass:         "test-pass",
		Interceptors: interceptor.AsList(),
	})
	c.apiPaths = &NewStyleAPI

	// when
	err := c.Login()

	// then
	require.NoError(t, err, "user/pass login must not produce an error")
	a.EqualValues(http.MethodPost, interceptor.Method())
	a.EqualValues(http.StatusOK, interceptor.response.StatusCode)
}

func TestResponseDataHandling(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	reqData := TestData{
		Data: "request",
	}
	srv := runTestServer(NewStyleAPI.ApiPath + "/test")
	c, _ := NewClient(&ClientConfig{
		URL:    srv.URL,
		APIKey: "test-key",
	})
	c.apiPaths = &NewStyleAPI
	var data TestData

	// when
	err := c.Get(context.Background(), "test", reqData, &data)

	// then
	require.NoError(t, err)
	a.EqualValues("test", data.Data)
}

func TestCsrfHandling(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	srv := runTestServer("")
	interceptor := NewTestInterceptor()
	c, _ := NewClient(&ClientConfig{
		URL:          srv.URL,
		User:         "test-user",
		Pass:         "test-pass",
		Interceptors: interceptor.AsList(),
	})
	c.apiPaths = &NewStyleAPI

	// when
	err := c.Get(context.Background(), "", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues("", interceptor.RequestHeader(CsrfHeader))
	a.EqualValues("csrf-token", interceptor.ResponseHeader(CsrfHeader))

	// when
	err = c.Get(context.Background(), "", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues("csrf-token", interceptor.RequestHeader(CsrfHeader))
}

func TestOverrideUserAgent(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	interceptor := NewTestInterceptor()
	c, _ := NewClient(&ClientConfig{
		URL:          testUrl,
		APIKey:       "test-key",
		Interceptors: interceptor.AsList(),
		UserAgent:    "test-agent",
	})
	c.apiPaths = &NewStyleAPI

	// when
	err := c.Get(context.Background(), "", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues("test-agent", interceptor.RequestHeader(UserAgentHeader))
}

func TestAuthConfigurationValidation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		User, Pass, APIKey string
		shouldFail         bool
	}{
		{"", "", "", true},
		{"", "", "test", false},
		{"", "test", "", true},
		{"", "test", "test", true},
		{"test", "", "", true},
		{"test", "", "test", true},
		{"test", "test", "", false},
		{"test", "test", "test", true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("user:%s-pass:%s-apikey:%s", tc.User, tc.Pass, tc.APIKey), func(t *testing.T) {
			t.Parallel()
			// given
			_, err := NewClient(&ClientConfig{
				URL:    testUrl,
				User:   tc.User,
				Pass:   tc.Pass,
				APIKey: tc.APIKey,
			})

			// then
			if tc.shouldFail {
				require.ErrorContains(t, err, "validation failed")
				return
			}
			require.ErrorContains(t, err, "dial tcp") // error will anyway exist, but it will be not related to config
		})
	}
}

func TestUrlValidation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		URL         string
		shouldFail  bool
		errorString string
	}{
		{"", true, "required"},
		{"http://test.url", false, ""},
		{"http://test.url:3999", false, ""},
		{"https://test.url:3999", false, ""},
		{"ftp://test.url", true, "http"},
		{"test.url", true, "http"},
		{"http://127.0.0.1", false, ""},
		{"http://127.0.0.1:3999", false, ""},
		{"test", true, "http"},
	}

	for _, tc := range testCases {
		t.Run(tc.URL, func(t *testing.T) {
			t.Parallel()
			// given
			_, err := NewClient(&ClientConfig{
				URL:    tc.URL,
				APIKey: "test-key",
			})

			// then
			if tc.shouldFail {
				require.ErrorContains(t, err, "validation failed")
				require.ErrorContains(t, err, tc.errorString)
				return
			}
			require.ErrorContains(t, err, "dial tcp") // error will anyway exist, but it will be not related to config
		})
	}
}

type validateableBody struct {
	Data string `json:"data" validate:"required"`
}

func TestValidationModes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		validationMode validationMode
		expectedError  string
		expectRequest  bool
	}{
		{SoftValidation, "dial tcp", true},
		{HardValidation, "validation failed", false},
		{DisableValidation, "dial tcp", true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.validationMode), func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			// given
			interceptor := NewTestInterceptor()
			c, _ := NewClient(&ClientConfig{
				URL:            testUrl,
				APIKey:         "test-key",
				Interceptors:   []ClientInterceptor{interceptor},
				ValidationMode: tc.validationMode,
			})
			c.apiPaths = &NewStyleAPI
			// when
			err := c.Get(context.Background(), "", validateableBody{}, nil)

			// then
			require.ErrorContains(t, err, tc.expectedError)
			if tc.expectRequest {
				a.NotNil(interceptor.request)
			} else {
				a.Nil(interceptor.request)
			}
		})
	}
}

// Common test server setup for system information tests
type sysInfoTestCase struct {
	name           string
	newAPIVersion  string
	oldAPIVersion  string
	expectedError  string
	expectedResult string
}

func setupSysInfoTestServer(tc sysInfoTestCase) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "", "/":
			w.WriteHeader(http.StatusOK)
		case "/proxy/network/api/s/default/stat/sysinfo":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"data": [{"version": "%s"}]}`, tc.newAPIVersion)
		case "/proxy/network/status":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"Meta": {"server_version": "%s"}}`, tc.oldAPIVersion)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestGetSystemInformation(t *testing.T) {
	t.Parallel()

	testCases := []sysInfoTestCase{
		{
			name:           "New API Success",
			newAPIVersion:  "v2-success",
			oldAPIVersion:  "",
			expectedResult: "v2-success",
		},
		{
			name:           "Fallback to Old API",
			newAPIVersion:  "",
			oldAPIVersion:  "old-success",
			expectedResult: "old-success",
		},
		{
			name:          "Both APIs Failure",
			newAPIVersion: "",
			oldAPIVersion: "",
			expectedError: "new API returned empty server info",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			ts := setupSysInfoTestServer(tc)
			defer ts.Close()

			c, err := NewClient(&ClientConfig{
				URL:       ts.URL,
				APIKey:    "dummy",
				VerifySSL: false,
			})

			sysInfo, err := c.getSystemInformation()

			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				a.Nil(sysInfo)
			} else {
				require.NoError(t, err)
				a.Equal(tc.expectedResult, sysInfo.Version)
			}
		})
	}
}
