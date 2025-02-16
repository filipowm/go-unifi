package unifi

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

// validationMode represents the mode for request validation.
// It may be set to "soft", "hard", or "disable". The default is "soft".
type validationMode string

const (
	// SoftValidation indicates that validation errors are logged as warnings but do not prevent the request from proceeding.
	SoftValidation validationMode = "soft"
	// HardValidation indicates that validation errors are treated as fatal and will cause the request to be rejected.
	HardValidation validationMode = "hard"
	// DisableValidation indicates that no validation is performed on the request body.
	DisableValidation validationMode = "disable"
	// DefaultValidation is the default validation mode used if none is specified.
	// Currently set to SoftValidation, but may change to HardValidation in a future major version.
	DefaultValidation validationMode = SoftValidation // TODO: change to hard in next major version
)

// HttpCustomizer is a function type for customizing the HTTP transport.
// It receives a pointer to an http.Transport and returns an error if customization fails.
type HttpCustomizer func(transport *http.Transport) error

// ResponseErrorHandler defines a method for handling HTTP response errors.
// HandleError processes the HTTP response and returns an error if the response indicates failure.
type ResponseErrorHandler interface {
	// HandleError processes the HTTP response and returns an error if the response signals a failure.
	HandleError(resp *http.Response) error
}

/*
ClientConfig holds configuration parameters for creating a UniFi client.

Fields:

	URL:           The base URL of the UniFi controller. Must be a valid URL and should not include the `/api` suffix.
	APIKey:        An API key used for authentication. Provide this if user/password credentials are not used.
	User:          The username for user/password authentication. Must be provided with Pass if APIKey is not used.
	Pass:          The password for user/password authentication. Must be provided with User if APIKey is not used.
	Timeout:       The maximum duration to wait for responses; default is no timeout.
	VerifySSL:     When false, disables SSL certificate verification.
	Interceptors:  A slice of ClientInterceptor implementations that can modify requests and responses.
	HttpCustomizer:An optional function to customize the HTTP transport (e.g., for custom TLS settings).
	UserAgent:     The User-Agent header string for outgoing HTTP requests.
	ErrorHandler:  A custom handler for processing HTTP response errors.
	UseLocking:    If true, enables internal locking for concurrent request processing.
	ValidationMode:The mode for validating request bodies. Can be "soft", "hard", or "disable".
*/
type ClientConfig struct {
	URL            string        `validate:"required,http_url"`
	APIKey         string        `validate:"required_without_all=User Pass"`
	User           string        `validate:"excluded_with=APIKey,required_with=Pass"`
	Pass           string        `validate:"excluded_with=APIKey,required_with=User"`
	Timeout        time.Duration // How long to wait for replies, default: forever.
	VerifySSL      bool
	Interceptors   []ClientInterceptor
	HttpCustomizer HttpCustomizer
	UserAgent      string
	ErrorHandler   ResponseErrorHandler
	UseLocking     bool
	ValidationMode validationMode `validate:"omitempty,oneof=soft hard disable"`
}

// Credentials abstracts authentication credentials.
// It defines methods to determine the type of credentials and retrieve the associated values.
type Credentials interface {
	// IsAPIKey returns true if the credentials represent an API key.
	IsAPIKey() bool
	// GetAPIKey returns the API key; returns an empty string if not applicable.
	GetAPIKey() string
	// GetUser returns the username for authentication; returns an empty string if not applicable.
	GetUser() string
	// GetPass returns the password for authentication; returns an empty string if not applicable.
	GetPass() string
}

// APIKeyCredentials holds API key authentication details.
type APIKeyCredentials struct {
	APIKey string
}

func (a APIKeyCredentials) IsAPIKey() bool    { return true }
func (a APIKeyCredentials) GetAPIKey() string { return a.APIKey }
func (a APIKeyCredentials) GetUser() string   { return "" }
func (a APIKeyCredentials) GetPass() string   { return "" }

// UserPassCredentials holds user/password authentication.
type UserPassCredentials struct {
	User string
	Pass string
}

func (u UserPassCredentials) IsAPIKey() bool    { return false }
func (u UserPassCredentials) GetAPIKey() string { return "" }
func (u UserPassCredentials) GetUser() string   { return u.User }
func (u UserPassCredentials) GetPass() string   { return u.Pass }

// client represents a UniFi client.
type client struct {
	baseURL        *url.URL
	sysInfo        *SysInfo
	apiPaths       *APIPaths
	timeout        time.Duration
	credentials    Credentials
	validationMode validationMode
	useLocking     bool

	http         *http.Client
	interceptors []ClientInterceptor
	errorHandler ResponseErrorHandler
	lock         sync.Mutex
	validator    *validator
}

var _ Client = &client{} // Ensure that client implements the Client interface. (compile-time check)

func (c *client) BaseURL() string {
	return c.baseURL.String()
}

//var _ client = &client{}

// AddInterceptor adds a ClientInterceptor to the client's interceptor list if it is not already present.
// It appends the interceptor only if it is not already included in the list.
func (c *client) AddInterceptor(interceptor *ClientInterceptor) {
	if !slices.Contains(c.interceptors, *interceptor) {
		c.interceptors = append(c.interceptors, *interceptor)
	}
}

func parseBaseURL(base string) (*url.URL, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	// Check if base URL's path is "/api" (deprecated usage now in api_paths.go)
	if strings.TrimSuffix(baseURL.Path, "/") == "/api" {
		return nil, fmt.Errorf("expected a base URL without the `/api`, got: %q", baseURL)
	}
	return baseURL, nil
}

func newClientFromConfig(config *ClientConfig, v *validator) (*client, error) {
	var err error
	config.URL = strings.TrimRight(config.URL, "/")
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !config.VerifySSL},
	}
	if config.HttpCustomizer != nil {
		if err = config.HttpCustomizer(transport); err != nil {
			return nil, fmt.Errorf("failed customizing HTTP transport: %w", err)
		}
	}
	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}
	if config.APIKey == "" {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, fmt.Errorf("failed creating cookiejar: %w", err)
		}
		httpClient.Jar = jar
	}
	baseURL, err := parseBaseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing base URL: %w", err)
	}
	var interceptors []ClientInterceptor
	var credentials Credentials

	if config.APIKey != "" {
		credentials = APIKeyCredentials{APIKey: config.APIKey}
		interceptors = append(interceptors, &APIKeyAuthInterceptor{apiKey: config.APIKey})
	} else {
		credentials = UserPassCredentials{User: config.User, Pass: config.Pass}
		interceptors = append(interceptors, &CSRFInterceptor{})
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
	u := &client{
		baseURL:        baseURL,
		timeout:        config.Timeout,
		credentials:    credentials,
		validationMode: config.ValidationMode,
		useLocking:     config.UseLocking,
		http:           httpClient,
		interceptors:   interceptors,
		errorHandler:   errorHandler,
		lock:           sync.Mutex{},
		validator:      v,
	}
	for _, interceptor := range config.Interceptors {
		u.AddInterceptor(&interceptor)
	}
	return u, nil
}

// NewClient creates and initializes a new UniFi client based on the provided ClientConfig.
// It validates the configuration, determines the API style, performs login if necessary,
// and retrieves system information from the UniFi controller.
// On success, it returns a pointer to a client; otherwise, it returns an error.
func NewClient(config *ClientConfig) (Client, error) {
	c, err := newBareClient(config)
	if err != nil {
		return c, err
	}
	if err = c.Login(); err != nil {
		return c, fmt.Errorf("failed logging in: %w", err)
	}
	if sysInfo, err := c.GetSystemInformation(); err != nil {
		return c, fmt.Errorf("failed getting server info: %w", err)
	} else {
		c.sysInfo = sysInfo
	}
	return c, nil
}

// NewBareClient creates a new UniFi client without performing login or system information retrieval.
// When user/pass authentication is used, you must call Login before making requests.
// It validates the configuration, determines the API style, and returns a pointer to the client on success.
func NewBareClient(config *ClientConfig) (Client, error) {
	return newBareClient(config)
}

func newBareClient(config *ClientConfig) (*client, error) {
	v, err := newValidator()
	if err != nil {
		return nil, fmt.Errorf("failed creating validator: %w", err)
	}
	if err = v.Validate(config); err != nil {
		return nil, fmt.Errorf("failed validating client configuration: %w", err)
	}
	c, err := newClientFromConfig(config, v)
	if err != nil {
		return nil, fmt.Errorf("failed creating unifi client: %w", err)
	}
	if err = c.determineApiStyle(); err != nil {
		return c, fmt.Errorf("failed determining API style: %w", err)
	}
	return c, nil
}

// Login authenticates the client using user/pass credentials.
// For API key authentication, Login does nothing.
// It returns an error if the authentication process fails.
func (c *client) Login() error {
	if c.credentials.IsAPIKey() {
		return nil
	}

	ctx, cancel := c.newRequestContext()
	defer cancel()

	err := c.Post(ctx, c.apiPaths.LoginPath, &struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: c.credentials.GetUser(),
		Password: c.credentials.GetPass(),
	}, nil)
	if err != nil {
		return err
	}
	return nil
}

// Logout terminates the client's session for user/pass authentication.
// For API key authentication, Logout does nothing.
// It returns an error if the logout process fails.
func (c *client) Logout() error {
	if c.credentials.IsAPIKey() {
		return nil
	}

	ctx, cancel := c.newRequestContext()
	defer cancel()

	err := c.Post(ctx, c.apiPaths.LogoutPath, nil, nil)
	return err
}
