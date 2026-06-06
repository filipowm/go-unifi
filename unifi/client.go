package unifi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
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

Concurrency: the resulting Client is safe for concurrent use by multiple
goroutines (requests run concurrently; see the Client concurrency contract).

Fields:

	URL:           The base URL of the UniFi controller. Must be a valid URL and should not include the `/api` suffix.
	APIKey:        An API key used for authentication. Provide this if user/password credentials are not used.
	User:          The username for user/password authentication. Must be provided with Password if APIKey is not used.
	Password:      The password for user/password authentication. Must be provided with User if APIKey is not used.
	RememberMe:    If true, the session is remembered for future requests. Useful for long-running processes. Default: false. Only used for user/password authentication.
	Timeout:       The maximum duration to wait for responses; default is no timeout.
	SkipVerifySSL: Controls TLS certificate verification. SECURE BY DEFAULT: the zero value (false) verifies certificates. Set it to true (SkipVerifySSL: true) to DISABLE verification — required for the common case of a self-signed controller certificate. Disabling verification is logged at WARN level on every client build.
	Interceptors:  A slice of ClientInterceptor implementations that can modify requests and responses.
	HttpTransportCustomizer:  An optional function to customize the HTTP transport (e.g., for custom TLS settings).
	HttpRoundTripperProvider: A function that returns a http.RoundTripper for customizing the HTTP client. If both HttpTransportCustomizer and HttpRoundTripperProvider are provided, HttpRoundTripperProvider takes precedence.
	UserAgent:     The User-Agent header string for outgoing HTTP requests.
	ErrorHandler:  A custom handler for processing HTTP response errors.
	UseLocking:    DEPRECATED and a NO-OP since 1.11.0. net/http.Client is goroutine-safe and the client no longer serializes requests; the field is retained only for source compatibility and has no effect.
	APIStyle:      Optionally forces the controller API style (new vs old) instead of probing the controller over the network. The zero value (APIStyleAuto) keeps the auto-detection behavior. Set it to skip the network probe for offline construction.
	ValidationMode:The mode for validating request bodies. Can be "soft", "hard", or "disable".
*/
type ClientConfig struct {
	URL        string        `validate:"required,http_url"`
	APIKey     string        `validate:"required_without_all=User Password"`
	User       string        `validate:"excluded_with=APIKey,required_with=Password"`
	Password   string        `validate:"excluded_with=APIKey,required_with=User"`
	RememberMe bool          `validate:"excluded_with=APIKey"`
	Timeout    time.Duration // How long to wait for replies, default: forever.
	// SkipVerifySSL controls TLS certificate verification. The zero value (false)
	// verifies certificates (secure by default); set it to true to disable
	// verification, e.g. SkipVerifySSL: true for a self-signed controller certificate.
	SkipVerifySSL            bool
	Interceptors             []ClientInterceptor
	HttpTransportCustomizer  HttpTransportCustomizer
	HttpRoundTripperProvider func() http.RoundTripper
	UserAgent                string
	ErrorHandler             ResponseErrorHandler
	// Deprecated: UseLocking is a no-op since 1.11.0 and has no effect.
	// Requests are dispatched through a goroutine-safe net/http.Client and are no
	// longer serialized. The field is retained only for source compatibility.
	UseLocking     bool
	APIStyle       APIStyle
	ValidationMode ValidationMode
	Logger         Logger
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
//
// Concurrency contract: a *client is safe for concurrent use by multiple
// goroutines. Requests are dispatched through a goroutine-safe net/http.Client
// and are NOT serialized — they run concurrently. The only mutable shared state
// is the cached system information (guarded by sysInfoMu) and, for user/pass
// auth, the CSRF token (guarded inside CSRFInterceptor). No request-level lock
// is held across the network round-trip.
type client struct {
	Logger

	baseURL        *url.URL
	sysInfo        *SysInfo
	apiPaths       *APIPaths
	timeout        time.Duration
	credentials    Credentials
	validationMode ValidationMode

	http         *http.Client
	interceptors []ClientInterceptor
	errorHandler ResponseErrorHandler
	// sysInfoMu guards the sysInfo cache. Reads take the read lock; the slow-path
	// fetch happens while holding NO lock, then the result is stored under the
	// write lock (double-checked). Holding it across the HTTP fetch would re-enter
	// a non-reentrant mutex and self-deadlock.
	sysInfoMu sync.RWMutex
	validator *validator
}

var _ Client = &client{} // Ensure that client implements the Client interface. (compile-time check)

func (c *client) BaseURL() string {
	return c.baseURL.String()
}

// AddInterceptor adds a ClientInterceptor to the client's interceptor list if no
// interceptor of the same concrete type is already present. Dedup is BY CONCRETE
// TYPE (reflect.TypeOf), not by value: this honors the "only one of a kind"
// intent (a single CSRF / API-key interceptor) and is panic-safe for interceptor
// types that are not comparable with == (e.g. structs holding a slice/map/func).
func (c *client) AddInterceptor(interceptor ClientInterceptor) {
	if interceptor == nil {
		return
	}
	if !containsInterceptorType(c.interceptors, interceptor) {
		c.interceptors = append(c.interceptors, interceptor)
	}
}

// containsInterceptorType reports whether interceptors already holds one whose
// concrete dynamic type matches the candidate's. Comparing reflect.TypeOf values
// (which are themselves always comparable) avoids the == panic that slices.Contains
// over potentially-non-comparable interface values would trigger.
func containsInterceptorType(interceptors []ClientInterceptor, candidate ClientInterceptor) bool {
	if candidate == nil {
		return false
	}
	t := reflect.TypeOf(candidate)
	for _, existing := range interceptors {
		if reflect.TypeOf(existing) == t {
			return true
		}
	}
	return false
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

// Version returns the cached controller version, fetching it once if the cache
// is empty. It swallows any fetch error (returning "") for source compatibility;
// callers that need to observe the error should use VersionContext. It derives a
// fresh request context honoring the client-wide timeout.
func (c *client) Version() string {
	ctx, cancel := c.newRequestContext()
	defer cancel()
	v, _ := c.VersionContext(ctx)
	return v
}

// VersionContext returns the version of the UniFi Controller API using the
// supplied context for cancellation/deadline. Unlike Version(), it surfaces the
// fetch error rather than swallowing it. It uses the same sysInfo cache and
// double-checked locking as Version().
func (c *client) VersionContext(ctx context.Context) (string, error) {
	// Fast path: read the cache under the dedicated read lock.
	if v, ok := c.cachedVersion(); ok {
		return v, nil
	}

	// Slow path: fetch over HTTP while holding NO lock. Holding sysInfoMu across
	// the round-trip would block every concurrent reader for the duration of the
	// fetch; it is taken again only to store the result below.
	i, err := c.GetSystemInformationContext(ctx)
	if err != nil {
		return "", err
	}

	// Store under the write lock, double-checking the cache in case a concurrent
	// caller populated it while we were fetching.
	c.sysInfoMu.Lock()
	defer c.sysInfoMu.Unlock()
	if c.sysInfo == nil {
		c.sysInfo = i
	}
	return c.sysInfo.Version, nil
}

// resolveLogger returns the configured logger or a default info-level logger.
func resolveLogger(config *ClientConfig) Logger {
	if config.Logger != nil {
		return config.Logger
	}
	return NewDefaultLogger(InfoLevel)
}

// tlsVerificationDisabled reports whether TLS certificate verification is
// explicitly disabled by the caller. Verification is SECURE BY DEFAULT: only an
// explicit SkipVerifySSL set to true turns it off; the zero value verifies.
func tlsVerificationDisabled(config *ClientConfig) bool {
	return config.SkipVerifySSL
}

// buildHTTPClient constructs the *http.Client from config: it resolves the
// round-tripper (custom provider or a default transport with optional
// InsecureSkipVerify and transport customizer), applies the timeout, and adds a
// cookiejar when no API key is used.
//
// TLS verification is secure by default: InsecureSkipVerify is only
// set when the caller explicitly disables verification via SkipVerifySSL, and that
// case is logged at WARN level.
func buildHTTPClient(config *ClientConfig, log Logger) (*http.Client, error) {
	var rt http.RoundTripper
	if config.HttpRoundTripperProvider != nil {
		log.Debug("Using custom HTTP round tripper provider")
		rt = config.HttpRoundTripperProvider()
	}
	if rt == nil {
		insecure := tlsVerificationDisabled(config)
		if insecure {
			log.Warn("TLS certificate verification is DISABLED (SkipVerifySSL set to true); connections are vulnerable to man-in-the-middle attacks")
		}
		//nolint:gosec // InsecureSkipVerify is opt-in via ClientConfig.SkipVerifySSL (secure by default)
		transport := &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
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
// any user-supplied config.Interceptors. The User-Agent is resolved into a local
// (the default when none is provided); nothing is written back
// through config. User-supplied interceptors are deduplicated by concrete type
// using the same semantics as (*client).AddInterceptor.
func buildInterceptors(config *ClientConfig, log Logger, auth []ClientInterceptor) []ClientInterceptor {
	interceptors := auth
	userAgent := config.UserAgent
	if len(userAgent) == 0 {
		userAgent = defaultUserAgent
	} else {
		log.Debugf("Using custom User-Agent header: %s", userAgent)
	}
	interceptors = append(interceptors, &DefaultHeadersInterceptor{headers: map[string]string{
		UserAgentHeader:   userAgent,
		AcceptHeader:      "application/json",
		ContentTypeHeader: "application/json; charset=utf-8",
	}})
	for _, interceptor := range config.Interceptors {
		if !containsInterceptorType(interceptors, interceptor) {
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
	// Operate on a shallow copy so we never write back through the
	// caller-owned *ClientConfig. URL normalization (trailing-slash trim) and the
	// default User-Agent are applied to this local copy only; the caller's struct
	// is left byte-for-byte intact.
	cfg := *config
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	log.Debugf("Connecting to UniFi controller at %s", cfg.URL)

	httpClient, err := buildHTTPClient(&cfg, log)
	if err != nil {
		return nil, err
	}
	baseURL, err := parseBaseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing base URL: %w", err)
	}
	credentials, auth := resolveCredentials(&cfg, log)
	interceptors := buildInterceptors(&cfg, log, auth)
	errorHandler := resolveErrorHandler(&cfg, log)
	log.Tracef("Validation mode: %d", cfg.ValidationMode)
	return &client{
		baseURL:        baseURL,
		timeout:        cfg.Timeout,
		credentials:    credentials,
		validationMode: cfg.ValidationMode,
		http:           httpClient,
		interceptors:   interceptors,
		errorHandler:   errorHandler,
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
	// APIStyle override: when the caller pins the style, skip the
	// network probe entirely so the client can be constructed fully offline.
	if config.APIStyle != APIStyleAuto {
		paths := apiPathsForStyle(config.APIStyle)
		if paths == &OldStyleAPI && c.credentials.IsAPIKey() {
			return c, errors.New("unable to use API key authentication with old style API. Switch to user/pass authentication or update controller to latest version")
		}
		c.Debugf("Using explicitly configured API style (skipping probe): %d", config.APIStyle)
		c.apiPaths = paths
		return c, nil
	}
	if err = c.determineApiStyle(); err != nil {
		return c, fmt.Errorf("failed determining API style: %w", err)
	}
	return c, nil
}

// Login authenticates the client using user/pass credentials.
// For API key authentication, Login does nothing.
// It returns an error if the authentication process fails.
// It derives a fresh request context honoring the client-wide timeout and
// delegates to LoginContext.
func (c *client) Login() error {
	ctx, cancel := c.newRequestContext()
	defer cancel()
	return c.LoginContext(ctx)
}

// LoginContext authenticates the client using user/pass credentials, using the
// supplied context for cancellation/deadline. For API key authentication it does
// nothing. The passed ctx is threaded through to the underlying HTTP call so a
// cancelled or expired context aborts the request.
func (c *client) LoginContext(ctx context.Context) error {
	if c.credentials.IsAPIKey() {
		c.Trace("API key authentication; skipping login")
		return nil
	}
	c.Trace("Logging in with user/pass credentials")

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
// It derives a fresh request context honoring the client-wide timeout and
// delegates to LogoutContext.
func (c *client) Logout() error {
	ctx, cancel := c.newRequestContext()
	defer cancel()
	return c.LogoutContext(ctx)
}

// LogoutContext terminates the client's session for user/pass authentication,
// using the supplied context for cancellation/deadline. For API key
// authentication it does nothing. The passed ctx is threaded through to the
// underlying HTTP call so a cancelled or expired context aborts the request.
func (c *client) LogoutContext(ctx context.Context) error {
	if c.credentials.IsAPIKey() {
		return nil
	}

	return c.Post(ctx, c.apiPaths.LogoutPath, nil, nil)
}

// cachedVersion returns the cached version and whether the cache was populated.
// It is the pure cache-decision half of VersionContext, split out from the IO so
// the cached-vs-fetch branch is testable without timing hacks. The read
// is performed under the dedicated read lock.
func (c *client) cachedVersion() (string, bool) {
	c.sysInfoMu.RLock()
	defer c.sysInfoMu.RUnlock()
	if c.sysInfo != nil {
		return c.sysInfo.Version, true
	}
	return "", false
}
