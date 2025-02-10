package unifi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	apiPath   = "/api"
	apiV2Path = "/v2/api"

	apiPathNew   = "/proxy/network/api"
	apiV2PathNew = "/proxy/network/v2/api"

	loginPath    = "/api/login"
	loginPathNew = "/api/auth/login"

	statusPath    = "/status"
	statusPathNew = "/proxy/network/status"

	logoutPath = "/api/logout"

	defaultUserAgent = "go-unifi/0.0.1"

	ApiKeyHeader      = "X-Api-Key"
	CsrfHeader        = "X-Csrf-Token"
	UserAgentHeader   = "User-Agent"
	AcceptHeader      = "Accept"
	ContentTypeHeader = "Content-Type"

	SoftValidation    validationMode = "soft"
	HardValidation    validationMode = "hard"
	DisableValidation validationMode = "disable"
	DefaultValidation validationMode = SoftValidation // TODO: change to hard in next major version
)

type validationMode string

type ClientConfig struct {
	URL            string        `validate:"required,http_url"`
	APIKey         string        `validate:"required_without_all=User Pass"`
	User           string        `validate:"excluded_with=APIKey,required_with=Pass"`
	Pass           string        `validate:"excluded_with=APIKey,required_with=User"`
	Timeout        time.Duration // how long to wait for replies, default: forever.
	VerifySSL      bool
	Interceptors   []ClientInterceptor
	HttpCustomizer HttpCustomizer
	UserAgent      string
	ErrorHandler   ResponseErrorHandler
	UseLocking     bool
	ValidationMode validationMode `validate:"omitempty,oneof=soft hard disable"`
}

type Client struct {
	BaseURL      *url.URL
	SysInfo      *SysInfo
	apiPaths     *ApiPaths
	config       *ClientConfig
	http         *http.Client
	interceptors []ClientInterceptor
	errorHandler ResponseErrorHandler
	lock         sync.Mutex
	validator    *validator
}

type ApiPaths struct {
	ApiPath    string
	ApiV2Path  string
	LoginPath  string
	StatusPath string
	LogoutPath string
}

var (
	OldStyleAPI = ApiPaths{
		ApiPath:    apiPath,
		ApiV2Path:  apiV2Path,
		LoginPath:  loginPath,
		StatusPath: statusPath,
		LogoutPath: logoutPath,
	}
	NewStyleAPI = ApiPaths{
		ApiPath:    apiPathNew,
		ApiV2Path:  apiV2PathNew,
		LoginPath:  loginPathNew,
		StatusPath: statusPathNew,
		LogoutPath: logoutPath,
	}
)

type ServerInfo struct {
	Up            bool   `json:"up"`
	ServerVersion string `json:"server_version"`
	UUID          string `json:"uuid"`
}

type HttpCustomizer func(transport *http.Transport) error

type ClientInterceptor interface {
	InterceptRequest(req *http.Request) error
	InterceptResponse(resp *http.Response) error
}
type ApiKeyAuthInterceptor struct {
	apiKey string
}

func (a *ApiKeyAuthInterceptor) InterceptRequest(req *http.Request) error {
	req.Header.Set(ApiKeyHeader, a.apiKey)
	return nil
}

func (a *ApiKeyAuthInterceptor) InterceptResponse(_ *http.Response) error {
	return nil
}

type CsrfInterceptor struct {
	csrfToken string
}

func (c *CsrfInterceptor) InterceptRequest(req *http.Request) error {
	if c.csrfToken != "" {
		req.Header.Set(CsrfHeader, c.csrfToken)
	}
	return nil
}

func (c *CsrfInterceptor) InterceptResponse(resp *http.Response) error {
	if csrf := resp.Header.Get(CsrfHeader); csrf != "" {
		c.csrfToken = csrf
	}
	return nil
}

type DefaultHeadersInterceptor struct {
	headers map[string]string
}

func (d *DefaultHeadersInterceptor) InterceptRequest(req *http.Request) error {
	for key, value := range d.headers {
		req.Header.Set(key, value)
	}
	return nil
}

func (d *DefaultHeadersInterceptor) InterceptResponse(_ *http.Response) error {
	return nil
}

func (c *Client) RegisterInterceptor(interceptor *ClientInterceptor) {
	// ensure no duplicate interceptors
	if !slices.Contains(c.interceptors, *interceptor) {
		c.interceptors = append(c.interceptors, *interceptor)
	}
}

type ResponseErrorHandler interface {
	HandleError(resp *http.Response) error
}

// NewClient creates a http.Client with authenticated cookies.
// Used to make additional, authenticated requests to the APIs.
// Start here.
func NewClient(config *ClientConfig) (*Client, error) {
	v, err := newValidator()
	if err != nil {
		return nil, fmt.Errorf("failed creating validator: %w", err)
	}

	if err := v.Validate(config); err != nil {
		return nil, fmt.Errorf("failed validating config: %w", err)
	}

	u, err := newUnifi(config, v)
	if err != nil {
		return nil, fmt.Errorf("failed creating unifi client: %w", err)
	}

	if err = u.determineApiStyle(); err != nil {
		return u, fmt.Errorf("failed determining API style: %w", err)
	}

	if err = u.Login(); err != nil {
		return u, fmt.Errorf("failed logging in: %w", err)
	}

	if sysInfo, err := u.getSystemInformation(); err != nil {
		return u, fmt.Errorf("failed getting server info: %w", err)
	} else {
		u.SysInfo = sysInfo
	}
	return u, nil
}

func parseBaseUrl(base string) (*url.URL, error) {
	var err error
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, err
	}

	// error for people who are still passing hard coded old paths
	if path := strings.TrimSuffix(baseURL.Path, "/"); path == apiPath {
		return nil, fmt.Errorf("expected a base URL without the `/api`, got: %q", baseURL)
	}

	return baseURL, nil
}

func newUnifi(config *ClientConfig, v *validator) (*Client, error) {
	var err error

	config.URL = strings.TrimRight(config.URL, "/")
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !config.VerifySSL}, //nolint: gosec
	}

	if config.HttpCustomizer != nil {
		if err = config.HttpCustomizer(transport); err != nil {
			return nil, fmt.Errorf("failed customizing HTTP transport: %w", err)
		}
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	if config.APIKey == "" {
		// old user/pass style use the cookie jar
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, fmt.Errorf("failed creating cookiejar: %w", err)
		}
		client.Jar = jar
	}
	baseURL, err := parseBaseUrl(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing base URL: %w", err)
	}
	var interceptors []ClientInterceptor

	if config.APIKey != "" {
		interceptors = append(interceptors, &ApiKeyAuthInterceptor{apiKey: config.APIKey})
	} else {
		// CSRF is only needed for user/pass auth
		interceptors = append(interceptors, &CsrfInterceptor{})
	}
	if len(config.UserAgent) == 0 {
		config.UserAgent = defaultUserAgent
	}
	interceptors = append(interceptors, &DefaultHeadersInterceptor{headers: map[string]string{
		UserAgentHeader:   config.UserAgent,
		AcceptHeader:      "application/json",
		ContentTypeHeader: "application/json; charset=utf-8",
	}})

	var errorHandler ResponseErrorHandler
	if config.ErrorHandler != nil {
		errorHandler = config.ErrorHandler
	} else {
		errorHandler = &DefaultResponseErrorHandler{}
	}
	if config.ValidationMode == "" {
		config.ValidationMode = DefaultValidation
	}
	u := &Client{
		BaseURL:      baseURL,
		config:       config,
		http:         client,
		interceptors: interceptors,
		errorHandler: errorHandler,
		lock:         sync.Mutex{},
		validator:    v,
	}
	for _, interceptor := range config.Interceptors {
		// add any custom interceptors and ensure no duplicates
		u.RegisterInterceptor(&interceptor)
	}

	return u, nil
}

// Login is a helper method. It can be called to grab a new authentication cookie.
// Only useful if you are using user/pass auth.
func (c *Client) Login() error {
	if c.config.APIKey != "" {
		// no need to login on api-key auth
		return nil
	}

	ctx, cancel := c.createRequestContext()
	defer cancel()

	err := c.Post(ctx, c.apiPaths.LoginPath, &struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: c.config.User,
		Password: c.config.Pass,
	}, nil)
	if err != nil {
		return err
	}
	return nil
}

// Logout closes the current session. Only useful if you are using user/pass auth.
func (c *Client) Logout() error {
	if c.config.APIKey != "" {
		// no need to logout on api-key auth
		return nil
	}
	ctx, cancel := c.createRequestContext()
	defer cancel()

	// a post is needed for logout
	err := c.Post(ctx, c.apiPaths.LogoutPath, nil, nil)

	return err
}

func (c *Client) createRequestContext() (context.Context, context.CancelFunc) {
	var (
		ctx    = context.Background()
		cancel = func() {}
	)
	if c.config.Timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, c.config.Timeout)
	}
	return ctx, cancel
}

// with the release of controller version 5.12.55 on UDM in Jan 2020 the api paths
// changed and broke this library. This function runs when `NewClient()` is called to
// check if this is a newer controller or not. If it is, we set new to true.
// Setting new to true makes the path() method return different (new) paths.
func (c *Client) determineApiStyle() error {
	ctx, cancel := c.createRequestContext()
	defer cancel()

	// c.DebugLog("Requesting %s/ to determine API paths", c.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL.String(), nil)
	if err != nil {
		return err
	}

	// We can't share these cookies with other requests, so make a new client.
	// Checking the return code on the first request so don't follow a redirect.
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: c.http.Transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()               // we need no data here.
	_, _ = io.Copy(io.Discard, resp.Body) // avoid leaking.

	switch resp.StatusCode {
	case http.StatusOK:
		c.apiPaths = &NewStyleAPI // The new version returns a "200" for a / request.
	case http.StatusFound:
		c.apiPaths = &OldStyleAPI // The old version returns a "302" (to /manage) for a / request.
	default:
		return fmt.Errorf("expected 200 or 302 status code, but got: %d", resp.StatusCode)
	}
	if c.apiPaths == &OldStyleAPI && c.config.APIKey != "" {
		return errors.New("unable to use API key authentication with old style API. Switch to user/pass authentication or update controller to latest version")
	}
	return nil
}

func (c *Client) getOldSysInfo(ctx context.Context) (*SysInfo, error) {
	var response struct {
		Data ServerInfo `json:"Meta"`
	}

	err := c.Get(ctx, c.apiPaths.StatusPath, nil, &response)
	if err != nil {
		return nil, err
	}
	data := response.Data
	return &SysInfo{
		Version: data.ServerVersion,
	}, nil
}

// getSystemInformation reads the controller's version and UUID. Only call this if you
// previously called Login and suspect the controller version has changed.
func (c *Client) getSystemInformation() (*SysInfo, error) {
	ctx, cancel := c.createRequestContext()
	defer cancel()

	var resultingError error
	info, err := c.GetSystemInfo(ctx, "default") // get for default site which must exist
	if err != nil {
		resultingError = err
	} else if info == nil || info.Version == "" {
		resultingError = errors.New("new API returned empty server info")
	}

	if resultingError != nil {
		info, err = c.getOldSysInfo(ctx)
		if err != nil {
			resultingError = errors.Join(resultingError, err)
		} else if info == nil || info.Version == "" {
			resultingError = errors.Join(resultingError, errors.New("old API returned empty server info"))
		} else {
			resultingError = nil
		}
	}

	if resultingError != nil {
		return nil, resultingError
	}
	return info, nil
}

func marshalRequest(reqBody interface{}) (io.Reader, error) {
	if reqBody == nil {
		return nil, nil //nolint: nilnil
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(reqBytes), nil
}

func (c *Client) createRequestURL(apiPath string) (*url.URL, error) {
	reqURL, err := url.Parse(apiPath)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(apiPath, "/") && !reqURL.IsAbs() {
		reqURL.Path = path.Join(c.apiPaths.ApiPath, reqURL.Path)
	}

	return c.BaseURL.ResolveReference(reqURL), nil
}

func (c *Client) validateRequestBody(reqBody interface{}) error {
	if reqBody != nil && c.config.ValidationMode != DisableValidation {
		if err := c.validator.Validate(reqBody); err != nil {
			err = fmt.Errorf("failed validating request body: %w", err)
			if c.config.ValidationMode == HardValidation {
				return err
			} else {
				fmt.Println(err)
			}
		}
	}
	return nil
}

// Do performs a request to the given API path with the given method.
func (c *Client) Do(ctx context.Context, method, apiPath string, reqBody interface{}, respBody interface{}) error {
	if err := c.validateRequestBody(reqBody); err != nil {
		return err
	}
	reqReader, err := marshalRequest(reqBody)
	if err != nil {
		return fmt.Errorf("unable to marshal request: %w", err)
	}

	url, err := c.createRequestURL(apiPath)
	if err != nil {
		return fmt.Errorf("unable to create request URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, url.String(), reqReader)
	if err != nil {
		return fmt.Errorf("unable to create request: %s %s %w", method, apiPath, err)
	}
	if c.config.UseLocking {
		c.lock.Lock()
		defer c.lock.Unlock()
	}

	for _, interceptor := range c.interceptors {
		if err := interceptor.InterceptRequest(req); err != nil {
			return err
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("unable to perform request: %s %s %w", method, apiPath, err)
	}
	defer resp.Body.Close()

	for _, interceptor := range c.interceptors {
		if err := interceptor.InterceptResponse(resp); err != nil {
			return err
		}
	}
	if err := c.errorHandler.HandleError(resp); err != nil {
		return err
	}
	if respBody == nil || resp.ContentLength == 0 {
		return nil
	}

	err = json.NewDecoder(resp.Body).Decode(respBody)
	if err != nil {
		return fmt.Errorf("unable to decode body: %s %s %w", method, apiPath, err)
	}

	return nil
}

// Get performs a GET request to the given API path.
func (c *Client) Get(context context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(context, http.MethodGet, apiPath, reqBody, respBody)
}

// Post performs a POST request to the given API path.
func (c *Client) Post(context context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(context, http.MethodPost, apiPath, reqBody, respBody)
}

// Put performs a PUT request to the given API path.
func (c *Client) Put(context context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(context, http.MethodPut, apiPath, reqBody, respBody)
}

// Delete performs a DELETE request to the given API path.
func (c *Client) Delete(context context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(context, http.MethodDelete, apiPath, reqBody, respBody)
}
