package unifi

import (
	"net/http"
)

// ClientInterceptor defines the interface for interceptors.
// An interceptor can modify HTTP requests and responses.
//
// Execution order: built-in auth interceptor, built-in default-headers interceptor,
// then user-supplied interceptors in registration order.
// Response interceptors run before error handling and response decoding.
//
// IMPORTANT: Do not read or consume resp.Body in InterceptResponse — the body
// is decoded after interceptors run; consuming it will cause silent decode failures
// (the caller receives a zero-valued response with nil error).
// For timing and tracing use ClientConfig.HttpRoundTripperProvider which wraps
// the transport and can properly read and replace the response body.
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
