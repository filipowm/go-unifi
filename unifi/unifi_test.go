package unifi //nolint: testpackage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	localUrl = "http://127.0.0.1:64431"
	testUrl  = "http://test.url"
)

func verifyContainsInterceptors(a *assert.Assertions, c *Client, interceptors ...interface{}) {
	var (
		expectedTypes = make([]reflect.Type, len(interceptors))
		matchingTypes = make([]reflect.Type, len(interceptors))
	)
	for _, i := range interceptors {
		expectedTypes = append(expectedTypes, reflect.TypeOf(i))
	}
	for _, i := range c.interceptors {
		actualType := reflect.TypeOf(i)
		if slices.Contains(expectedTypes, actualType) {
			matchingTypes = append(matchingTypes, actualType)
		}
	}
	if len(matchingTypes) != len(expectedTypes) {
		a.Fail(fmt.Sprintf("interceptors not found; expected: %v, found: %v", expectedTypes, matchingTypes))
	}
}

func verifyDoesNotContainInterceptors(a *assert.Assertions, c *Client, interceptors ...interface{}) {
	var (
		expectedTypes = make([]reflect.Type, 0, len(interceptors))
		matchingTypes = make([]reflect.Type, 0, len(interceptors))
	)
	for _, i := range interceptors {
		expectedTypes = append(expectedTypes, reflect.TypeOf(i))
	}
	for _, i := range c.interceptors {
		actualType := reflect.TypeOf(i)
		if slices.Contains(expectedTypes, actualType) {
			matchingTypes = append(matchingTypes, actualType)
		}
	}
	if len(matchingTypes) != 0 {
		a.Fail(fmt.Sprintf("interceptors found; expected to be not present: %v, found: %v", expectedTypes, matchingTypes))
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
	verifyContainsInterceptors(a, c, &CsrfInterceptor{}, &DefaultHeadersInterceptor{})
	verifyDoesNotContainInterceptors(a, c, &ApiKeyAuthInterceptor{})
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
	verifyContainsInterceptors(a, c, &ApiKeyAuthInterceptor{}, &DefaultHeadersInterceptor{})
	verifyDoesNotContainInterceptors(a, c, &CsrfInterceptor{})
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

func TestInterceptors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()

	// when
	err := c.Get(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.True(interceptor.IsRequestIntercepted(), "request interceptor not called")
	a.False(interceptor.IsResponseIntercepted(), "response interceptor called, but should not because of failed request")
}

func TestNoSendRequestWhenRequestInterceptorReturnsError(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()
	interceptor.failOnRequest = true

	// when
	err := c.Get(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.Contains(err.Error(), "request interceptor failed")
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
			// given
			c, interceptor := NewTestClientWithInterceptor()

			// when
			err := c.Get(context.Background(), tc.path, nil, nil)

			// then
			require.Error(t, err)
			a.EqualValues(tc.expected, interceptor.request.URL.String())
		})
	}
}

func TestApiKeyAddedToRequest(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()

	// when
	err := c.Get(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues("test-key", interceptor.RequestHeader(ApiKeyHeader))
}

func TestDefaultHeadersAddedToRequest(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()

	// when
	err := c.Get(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues("application/json", interceptor.RequestHeader(AcceptHeader))
	a.EqualValues("application/json; charset=utf-8", interceptor.RequestHeader(ContentTypeHeader))
	a.EqualValues(defaultUserAgent, interceptor.RequestHeader(UserAgentHeader))
}

type TestData struct {
	Data string `json:"data"`
}

func TestRequestSentWithJson(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()
	data := &TestData{
		Data: "test",
	}

	// when
	err := c.Get(context.Background(), "/", data, nil)

	// then
	require.Error(t, err)
	body := &TestData{}
	err = json.NewDecoder(interceptor.request.Body).Decode(body)
	require.NoError(t, err)
	a.Equal(data, body)
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
			// given
			c, interceptor := NewTestClientWithInterceptor()

			// when
			err := c.Do(context.Background(), tc, "", nil, nil)

			// then
			require.Error(t, err)
			a.EqualValues(tc, interceptor.Method())
		})
	}
}

func TestGetRequest(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()

	// when
	err := c.Get(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues(http.MethodGet, interceptor.Method())
}

func TestPostRequest(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()

	// when
	err := c.Post(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues(http.MethodPost, interceptor.Method())
}

func TestPutRequest(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()

	// when
	err := c.Put(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues(http.MethodPut, interceptor.Method())
}

func TestDeleteRequest(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	c, interceptor := NewTestClientWithInterceptor()

	// when
	err := c.Delete(context.Background(), "/", nil, nil)

	// then
	require.Error(t, err)
	a.EqualValues(http.MethodDelete, interceptor.Method())
}

func RunTestServer(path string, requestBody interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(CsrfHeader, "csrf-token")
		if !strings.EqualFold(r.URL.Path, path) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("error reading body:%v", err)
			return
		}
		err = json.Unmarshal(data, &requestBody)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("error decoding body: %s: %s", string(data), err)
			return
		}
		resp := TestData{
			Data: "test",
		}
		respData, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("error encoding response: %s", err)
			return
		}
		_, err = w.Write(respData)
		if err != nil {
			fmt.Printf("error writing response: %s", err)
		}
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
	srv := RunTestServer(NewStyleAPI.LoginPath, userPass{})
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
	srv := RunTestServer(NewStyleAPI.ApiPath+"/test", TestData{})
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
	srv := RunTestServer("", struct{}{})
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
