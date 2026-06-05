package unifi

import (
	"net/http"
	"sync"
)

// ClientInterceptor defines the interface for interceptors.
// An interceptor can modify HTTP requests and responses.
type ClientInterceptor interface {
	InterceptRequest(req *http.Request) error
	InterceptResponse(resp *http.Response) error
}

// APIKeyAuthInterceptor adds an API key to outgoing requests.
// It implements the ClientInterceptor interface.
type APIKeyAuthInterceptor struct {
	apiKey string
}

// InterceptRequest sets the API key header on the given HTTP request.
// It adds the header defined by ApiKeyHeader with the stored API key and returns nil.
func (a *APIKeyAuthInterceptor) InterceptRequest(req *http.Request) error {
	req.Header.Set(ApiKeyHeader, a.apiKey)
	return nil
}

// InterceptResponse does not modify the HTTP response and always returns nil.
func (a *APIKeyAuthInterceptor) InterceptResponse(_ *http.Response) error {
	return nil
}

// CSRFInterceptor manages CSRF tokens when using user/pass authentication.
// It implements the ClientInterceptor interface.
//
// The CSRF token is read on every outgoing request (InterceptRequest) and
// updated from every response (InterceptResponse). Because a single client may
// fire concurrent requests from multiple goroutines, access to the token is
// guarded by an internal RWMutex so the read/write pair is data-race free
// regardless of the (now no-op) ClientConfig.UseLocking setting.
type CSRFInterceptor struct {
	mu        sync.RWMutex
	csrfToken string
}

// CSRFToken returns the most recently captured CSRF token in a data-race-safe way.
func (c *CSRFInterceptor) CSRFToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.csrfToken
}

// InterceptRequest adds the CSRF token to the HTTP request header if it is set.
// It returns nil on success.
func (c *CSRFInterceptor) InterceptRequest(req *http.Request) error {
	if token := c.CSRFToken(); token != "" {
		req.Header.Set(CsrfHeader, token)
	}
	return nil
}

// InterceptResponse extracts the CSRF token from the HTTP response header, if present, and stores it for future requests.
func (c *CSRFInterceptor) InterceptResponse(resp *http.Response) error {
	if token := resp.Header.Get(CsrfHeader); token != "" {
		c.mu.Lock()
		c.csrfToken = token
		c.mu.Unlock()
	}
	return nil
}

// DefaultHeadersInterceptor sets default HTTP headers for requests.
// It implements the ClientInterceptor interface.
type DefaultHeadersInterceptor struct {
	headers map[string]string
}

// InterceptRequest sets default HTTP headers on the request as specified in the interceptor's headers map.
// It returns nil on success.
func (d *DefaultHeadersInterceptor) InterceptRequest(req *http.Request) error {
	for key, value := range d.headers {
		req.Header.Set(key, value)
	}
	return nil
}

// InterceptResponse does not modify the HTTP response and always returns nil.
func (d *DefaultHeadersInterceptor) InterceptResponse(_ *http.Response) error {
	return nil
}
