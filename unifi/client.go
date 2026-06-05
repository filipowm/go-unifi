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

// ValidationMode represents the mode for request validation.
// It may be set to "soft", "hard", or "disable". The default is "soft".
type ValidationMode int

const (
	// SoftValidation indicates that validation errors are logged as warnings but do not prevent the request from proceeding.
	SoftValidation ValidationMode = iota
	// HardValidation indicates that validation errors are treated as fatal and will cause the request to be rejected.
	HardValidation
	// DisableValidation indicates that no validation is performed on the request body.
	DisableValidation
)

// HttpTransportCustomizer is a function type for customizing the HTTP transport.
// It receives a pointer to an http.Transport and returns an error if customization fails.
type HttpTransportCustomizer func(transport *http.Transport) (*http.Transport, error)

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
	User:          The username for user/password authentication. Must be provided with Password if APIKey is not used.
	Password:      The password for user/password authentication. Must be provided with User if APIKey is not used.
	RememberMe:    If true, the session is remembered for future requests. Useful for long-running processes. Default: false. Only used for user/password authentication.
	Timeout:       The maximum duration to wait for responses; default is no timeout.
	VerifySSL:     When false, disables SSL certificate verification.
	Interceptors:  A slice of ClientInterceptor implementations that can modify requests and responses.
	HttpTransportCustomizer:  An optional function to customize the HTTP transport (e.g., for custom TLS settings).
	HttpRoundTripperProvider: A function that returns a http.RoundTripper for customizing the HTTP client. If both HttpTransportCustomizer and HttpRoundTripperProvider are provided, HttpRoundTripperProvider takes precedence.
	UserAgent:     The User-Agent header string for outgoing HTTP requests.
	ErrorHandler:  A custom handler for processing HTTP response errors.
	UseLocking:    If true, enables internal locking for concurrent request processing.
	ValidationMode:The mode for validating request bodies. Can be "soft", "hard", or "disable".
*/
type ClientConfig struct {
	URL                      string        `validate:"required,http_url"`
	APIKey                   string        `validate:"required_without_all=User Password"`
	User                     string        `validate:"excluded_with=APIKey,required_with=Password"`
	Password                 string        `validate:"excluded_with=APIKey,required_with=User"`
	RememberMe               bool          `validate:"excluded_with=APIKey"`
	Timeout                  time.Duration // How long to wait for replies, default: forever.
	VerifySSL                bool
	Interceptors             []ClientInterceptor
	HttpTransportCustomizer  HttpTransportCustomizer
	HttpRoundTripperProvider func() http.RoundTripper
	UserAgent                string
	ErrorHandler             ResponseErrorHandler
	UseLocking               bool
	ValidationMode           ValidationMode
	Logger                   Logger
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
	IsRememberMe() bool
}

// APIKeyCredentials holds API key authentication details.
type APIKeyCredentials struct {
	APIKey string
}

func (a APIKeyCredentials) IsAPIKey() bool     { return true }
func (a APIKeyCredentials) GetAPIKey() string  { return a.APIKey }
func (a APIKeyCredentials) GetUser() string    { return "" }
func (a APIKeyCredentials) GetPass() string    { return "" }
func (a APIKeyCredentials) IsRememberMe() bool { return false }

// UserPassCredentials holds user/password authentication.
type UserPassCredentials struct {
	User     string
	Password string
	Remember bool
}

func (u UserPassCredentials) IsAPIKey() bool     { return false }
func (u UserPassCredentials) GetAPIKey() string  { return "" }
func (u UserPassCredentials) GetUser() string    { return u.User }
func (u UserPassCredentials) GetPass() string    { return u.Password }
func (u UserPassCredentials) IsRememberMe() bool { return u.Remember }

// client represents a UniFi client.
type client struct {
	Logger

	baseURL        *url.URL
	sysInfo        *SysInfo
	apiPaths       *APIPaths
	timeout        time.Duration
	credentials    Credentials
	validationMode ValidationMode
	useLocking     bool

	http         *http.Client
	interceptors []ClientInterceptor
	errorHandler ResponseErrorHandler
	lock         sync.Mutex
	// sysInfoMu guards the sysInfo cache. It is intentionally SEPARATE from lock
	// (the request-serialization mutex) so the two concerns never alias the same
	// mutex — aliasing them is what enabled the re-entrant Version() deadlock under
	// UseLocking:true (ARCH-01).
	sysInfoMu sync.RWMutex
	validator *validator
}

var _ Client = &client{} // Ensure that client implements the Client interface. (compile-time check)

func (c *client) BaseURL() string {
	return c.baseURL.String()
}

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

func (c *client) Version() string {
	// Fast path: read the cache under the dedicated read lock.
	c.sysInfoMu.RLock()
	if c.sysInfo != nil {
		v := c.sysInfo.Version
		c.sysInfoMu.RUnlock()
		return v
	}
	c.sysInfoMu.RUnlock()

	// Slow path: fetch over HTTP while holding NO lock. GetSystemInformation goes
	// through executeRequest, which re-acquires c.lock when UseLocking is set;
	// holding c.lock (or sysInfoMu) here would re-enter a non-reentrant mutex and
	// self-deadlock (ARCH-01).
	i, err := c.GetSystemInformation()
	if err != nil {
		return ""
	}

	// Store under the write lock, double-checking the cache in case a concurrent
	// caller populated it while we were fetching.
	c.sysInfoMu.Lock()
	defer c.sysInfoMu.Unlock()
	if c.sysInfo == nil {
		c.sysInfo = i
	}
	return c.sysInfo.Version
}

// resolveLogger returns the configured logger or a default info-level logger.
func resolveLogger(config *ClientConfig) Logger {
	if config.Logger != nil {
		return config.Logger
	}
	return NewDefaultLogger(InfoLevel)
}

// buildHTTPClient constructs the *http.Client from config: it resolves the
// round-tripper (custom provider or a default transport with optional
// InsecureSkipVerify and transport customizer), applies the timeout, and adds a
// cookiejar when no API key is used.
func buildHTTPClient(config *ClientConfig, log Logger) (*http.Client, error) {
	var rt http.RoundTripper
	if config.HttpRoundTripperProvider != nil {
		log.Debug("Using custom HTTP round tripper provider")
		rt = config.HttpRoundTripperProvider()
	}
	if rt == nil {
		//nolint:gosec // InsecureSkipVerify is configurable via ClientConfig.VerifySSL
		transport := &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !config.VerifySSL},
		}
		if config.HttpTransportCustomizer != nil {
			log.Debug("Customizing HTTP transport")
			var err error
			if transport, err = config.HttpTransportCustomizer(transport); err != nil {
				return nil, fmt.Errorf("failed customizing HTTP transport: %w", err)
			}
		}
		rt = transport
	}
	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: rt,
	}
	if config.APIKey == "" {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, fmt.Errorf("failed creating cookiejar: %w", err)
		}
		httpClient.Jar = jar
	}
	return httpClient, nil
}

// resolveCredentials selects API-key or user/pass credentials based on config and
// returns them together with the matching authentication interceptor.
func resolveCredentials(config *ClientConfig, log Logger) (Credentials, []ClientInterceptor) {
	if config.APIKey != "" {
		log.Debug("Using API key authentication")
		return APIKeyCredentials{APIKey: config.APIKey}, []ClientInterceptor{&APIKeyAuthInterceptor{apiKey: config.APIKey}}
	}
	log.Debug("Using user/pass authentication")
	return UserPassCredentials{User: config.User, Password: config.Password, Remember: config.RememberMe}, []ClientInterceptor{&CSRFInterceptor{}}
}

// buildInterceptors assembles the final interceptor chain: the provided auth
// interceptors, the default headers interceptor (with resolved User-Agent), and
// any user-supplied config.Interceptors. Note: this mutates config.UserAgent to
// the default when none is provided, preserving prior behavior. User-supplied
// interceptors are deduplicated using the same semantics as (*client).AddInterceptor.
func buildInterceptors(config *ClientConfig, log Logger, auth []ClientInterceptor) []ClientInterceptor {
	interceptors := auth
	if len(config.UserAgent) == 0 {
		config.UserAgent = defaultUserAgent
	} else {
		log.Debugf("Using custom User-Agent header: %s", config.UserAgent)
	}
	interceptors = append(interceptors, &DefaultHeadersInterceptor{headers: map[string]string{
		UserAgentHeader:   config.UserAgent,
		AcceptHeader:      "application/json",
		ContentTypeHeader: "application/json; charset=utf-8",
	}})
	for _, interceptor := range config.Interceptors {
		if !slices.Contains(interceptors, interceptor) {
			interceptors = append(interceptors, interceptor)
		}
	}
	return interceptors
}

// resolveErrorHandler returns the configured response error handler or the default one.
func resolveErrorHandler(config *ClientConfig, log Logger) ResponseErrorHandler {
	if config.ErrorHandler != nil {
		log.Debug("Using custom response error handler")
		return config.ErrorHandler
	}
	log.Debug("Using default response error handler")
	return &DefaultResponseErrorHandler{}
}

func newClientFromConfig(config *ClientConfig, v *validator) (*client, error) {
	log := resolveLogger(config)
	log.Info("Initializing new UniFi client")
	config.URL = strings.TrimRight(config.URL, "/")
	log.Debugf("Connecting to UniFi controller at %s", config.URL)

	httpClient, err := buildHTTPClient(config, log)
	if err != nil {
		return nil, err
	}
	baseURL, err := parseBaseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing base URL: %w", err)
	}
	credentials, auth := resolveCredentials(config, log)
	interceptors := buildInterceptors(config, log, auth)
	errorHandler := resolveErrorHandler(config, log)
	log.Tracef("Validation mode: %d", config.ValidationMode)
	return &client{
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
		Logger:         log,
	}, nil
}

// NewClient creates and initializes a new UniFi client based on the provided ClientConfig.
// It validates the configuration, determines the API style, performs login if necessary,
// and retrieves system information from the UniFi controller.
// On success, it returns a pointer to a client; otherwise, it returns an error.
func NewClient(config *ClientConfig) (Client, error) { //nolint: ireturn
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
		c.sysInfoMu.Lock()
		c.sysInfo = sysInfo
		c.sysInfoMu.Unlock()
		c.Debugf("Connected to UniFi controller\nversion: %s; name: %s; build: %s; hostname: %s", sysInfo.Version, sysInfo.Name, sysInfo.Build, sysInfo.Hostname)
	}
	return c, nil
}

// NewBareClient creates a new UniFi client without performing login or system information retrieval.
// When user/pass authentication is used, you must call Login before making requests.
// It validates the configuration, determines the API style, and returns a pointer to the client on success.
func NewBareClient(config *ClientConfig) (Client, error) { //nolint: ireturn
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
		c.Trace("API key authentication; skipping login")
		return nil
	}
	c.Trace("Logging in with user/pass credentials")

	ctx, cancel := c.newRequestContext()
	defer cancel()

	err := c.Post(ctx, c.apiPaths.LoginPath, &struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}{
		Username: c.credentials.GetUser(),
		Password: c.credentials.GetPass(),
		Remember: c.credentials.IsRememberMe(),
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
