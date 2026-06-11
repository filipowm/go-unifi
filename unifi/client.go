package unifi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/filipowm/go-unifi/v2/unifi/official"
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
	APIKey:        API key for authentication. Required. Obtain one from the UniFi Network controller.
	Timeout:       The maximum duration to wait for responses; default is no timeout. Consider 30s for most deployments — a zero Timeout allows requests to hang indefinitely against a slow or hostile controller, and a WARN is logged at build time when it is unset.
	SkipVerifySSL: Controls TLS certificate verification. SECURE BY DEFAULT: the zero value (false) verifies certificates. Set it to true (SkipVerifySSL: true) to DISABLE verification — required for the common case of a self-signed controller certificate. Disabling verification is logged at WARN level on every client build.
	Interceptors:  A slice of ClientInterceptor implementations that can modify requests and responses. Interceptors are deduplicated by concrete type; a duplicate triggers a WARN log.
	HttpTransportCustomizer:  An optional function to customize the HTTP transport (e.g., for custom TLS settings).
	HttpRoundTripperProvider: A function that returns a http.RoundTripper for customizing the HTTP client. If both HttpTransportCustomizer and HttpRoundTripperProvider are provided, HttpRoundTripperProvider takes precedence. TLS configuration is entirely the caller's responsibility; the client emits a WARN log at build time as a reminder.
	UserAgent:     The User-Agent header string for outgoing HTTP requests.
	ErrorHandler:  A custom handler for processing HTTP response errors.
	UseLocking:    DEPRECATED and a NO-OP since 1.11.0. net/http.Client is goroutine-safe and the client no longer serializes requests; the field is retained only for source compatibility and has no effect.
	APIStyle:      Optionally forces the controller API style (new vs old) instead of probing the controller over the network. The zero value (APIStyleAuto) keeps the auto-detection behavior. Set it to skip the network probe for offline construction.
	ValidationMode:The mode for validating request bodies. Can be "soft", "hard", or "disable".
	SkipSystemInfo: Skips the eager GetSystemInformation() round-trip in NewClient. Zero value (false) keeps fail-fast; true defers error surfacing to the first API call.
*/
type ClientConfig struct {
	URL    string `validate:"required,https_url"`
	APIKey string `validate:"required"`

	// Timeout is the maximum duration to wait for a controller response.
	// Zero (the default) means no timeout — requests can hang indefinitely if the
	// controller is unreachable or slow. Consider 30s for most deployments.
	// A WARN log is emitted at client build time when Timeout is not set.
	Timeout time.Duration
	// SkipVerifySSL controls TLS certificate verification. The zero value (false)
	// verifies certificates (secure by default); set it to true to disable
	// verification, e.g. SkipVerifySSL: true for a self-signed controller certificate.
	SkipVerifySSL            bool
	// Interceptors is a list of ClientInterceptor implementations applied to every
	// request and response. Interceptors are deduplicated by concrete type: if two
	// interceptors share the same concrete type, the second is silently dropped and
	// a WARN log is emitted.
	Interceptors             []ClientInterceptor
	HttpTransportCustomizer  HttpTransportCustomizer
	// HttpRoundTripperProvider, when set, replaces the entire HTTP transport with
	// the returned RoundTripper. TLS configuration is entirely the caller's
	// responsibility: the client performs no TLS validation and the SkipVerifySSL
	// flag is ignored. Setting this field causes a WARN log at client build time as
	// a reminder that TLS is caller-managed.
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
	// DisableOfficialAPI opts the client out of the Official UniFi OpenAPI:
	// Official() operations then fail fast with ErrOfficialAPIDisabled and the
	// capability probe is skipped entirely.
	DisableOfficialAPI bool
	// SkipSystemInfo skips the eager GetSystemInformation() call in NewClient.
	// Zero value (false) preserves the default fail-fast behavior: a bad API key
	// or unreachable controller surfaces at construction time. Set it to true to
	// defer that check to the first Version()/API call — required for fully-offline
	// construction when combined with a pinned APIStyle.
	SkipSystemInfo bool
}

// client represents a UniFi client.
//
// Concurrency contract: a *client is safe for concurrent use by multiple
// goroutines. Requests are dispatched through a goroutine-safe net/http.Client
// and are NOT serialized — they run concurrently. The only mutable shared state
// is the cached system information (guarded by sysInfoMu). No request-level lock
// is held across the network round-trip.
type client struct {
	Logger

	baseURL        *url.URL
	sysInfo        *SysInfo
	apiPaths       *APIPaths
	timeout        time.Duration
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

	// officialDisabled mirrors ClientConfig.DisableOfficialAPI: when set, the
	// capability gate fails fast with ErrOfficialAPIDisabled and never probes.
	officialDisabled bool
	// officialOnce lazily builds officialClient so its site-resolver cache
	// survives across Official() calls.
	officialOnce   sync.Once
	officialClient official.Client
	// officialReadyMu guards officialReady, which caches a successful capability
	// probe so /v1/info is fetched at most once.
	officialReadyMu sync.Mutex
	officialReady   bool
}

var _ Client = &client{} // Ensure that client implements the Client interface. (compile-time check)

func (c *client) BaseURL() string {
	return c.baseURL.String()
}

// AddInterceptor adds a ClientInterceptor to the client's interceptor list if no
// interceptor of the same concrete type is already present. Dedup is BY CONCRETE
// TYPE (reflect.TypeOf), not by value: this honors the "only one of a kind"
// intent and is panic-safe for interceptor types that are not comparable with ==
// (e.g. structs holding a slice/map/func).
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
	// Defence-in-depth: the validate tag already enforces https, but reject any
	// non-https scheme here too so a caller who bypasses validation can't silently
	// send an API key over plaintext.
	if baseURL.Scheme != "https" {
		return nil, fmt.Errorf("controller URL must use https://; got %q scheme", baseURL.Scheme)
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
// InsecureSkipVerify and transport customizer), and applies the timeout.
//
// TLS verification is secure by default: InsecureSkipVerify is only
// set when the caller explicitly disables verification via SkipVerifySSL, and that
// case is logged at WARN level.
func buildHTTPClient(config *ClientConfig, log Logger) (*http.Client, error) {
	var rt http.RoundTripper
	if config.HttpRoundTripperProvider != nil {
		log.Debug("Using custom HTTP round tripper provider")
		rt = config.HttpRoundTripperProvider()
		log.Warn("HttpRoundTripperProvider is set: TLS configuration is entirely caller-managed and cannot be validated by the client")
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
	return &http.Client{
		Timeout:   config.Timeout,
		Transport: rt,
	}, nil
}

// resolveAuthInterceptors returns the auth interceptor chain for the config.
func resolveAuthInterceptors(config *ClientConfig, log Logger) []ClientInterceptor {
	log.Debug("Using API key authentication")
	return []ClientInterceptor{&APIKeyAuthInterceptor{apiKey: config.APIKey}}
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
		} else {
			log.Warnf("interceptor of type %T skipped: an interceptor of the same concrete type is already registered; each concrete type may only appear once", interceptor)
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

	if cfg.Timeout == 0 {
		log.Warn("ClientConfig.Timeout is not set: requests can hang indefinitely; consider setting a timeout (e.g. 30s)")
	}
	httpClient, err := buildHTTPClient(&cfg, log)
	if err != nil {
		return nil, err
	}
	baseURL, err := parseBaseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing base URL: %w", err)
	}
	auth := resolveAuthInterceptors(&cfg, log)
	interceptors := buildInterceptors(&cfg, log, auth)
	errorHandler := resolveErrorHandler(&cfg, log)
	log.Tracef("Validation mode: %d", cfg.ValidationMode)
	return &client{
		baseURL:          baseURL,
		timeout:          cfg.Timeout,
		validationMode:   cfg.ValidationMode,
		http:             httpClient,
		interceptors:     interceptors,
		errorHandler:     errorHandler,
		validator:        v,
		Logger:           log,
		officialDisabled: cfg.DisableOfficialAPI,
	}, nil
}

// NewClient creates and initializes a new UniFi client based on the provided ClientConfig.
// It validates the configuration, determines the API style, and — unless SkipSystemInfo is true —
// eagerly fetches system information from the controller (fail-fast for bad credentials/unreachable host).
// On any error, nil is returned as the Client so callers can safely check err != nil without risking
// a nil-panic on the first API call.
func NewClient(config *ClientConfig) (Client, error) { //nolint: ireturn
	c, err := newClient(config)
	if err != nil {
		return nil, err
	}
	if !config.SkipSystemInfo {
		if sysInfo, err := c.GetSystemInformation(); err != nil {
			return nil, fmt.Errorf("failed getting server info: %w", err)
		} else {
			c.sysInfoMu.Lock()
			c.sysInfo = sysInfo
			c.sysInfoMu.Unlock()
			c.Debugf("Connected to UniFi controller\nversion: %s; name: %s; build: %s; hostname: %s", sysInfo.Version, sysInfo.Name, sysInfo.Build, sysInfo.Hostname)
		}
	}
	return c, nil
}

func newClient(config *ClientConfig) (*client, error) {
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
	// A pinned (non-auto) style skips the network probe so the client can be
	// constructed fully offline; APIStyleAuto probes the controller.
	switch config.APIStyle {
	case APIStyleAuto:
		if err = c.determineApiStyle(); err != nil {
			return nil, fmt.Errorf("failed determining API style: %w", err)
		}
	case APIStyleNew:
		c.apiPaths = &NewStyleAPI
		c.Debugf("Using explicitly configured API style (skipping probe): %d", config.APIStyle)
	case APIStyleOld:
		return nil, ErrOldStyleUnsupported
	default:
		return nil, fmt.Errorf("unsupported API style: %d", config.APIStyle)
	}
	return c, nil
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
