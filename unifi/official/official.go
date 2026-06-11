// Package official is the Official UniFi OpenAPI (integration/v1) client surface.
//
// It imports nothing from the parent unifi package: the controller transport is
// injected as a structural Doer, so the dependency is strictly one-way
// (unifi -> official). *unifi.client satisfies Doer through its public
// Get/Post/Put/Delete methods, and the capability check is injected as a Gate so
// the version/auth gating policy stays in the parent package.
//
// Errors: operations return errors from the injected Doer. When using the default
// unifi.Client transport, errors are *unifi.ServerError values — use
// errors.Is(err, unifi.ErrNotFound) to detect 404s and
// errors.As(err, &serverErr) for structured error details.
// A custom Doer is responsible for its own error types; this package makes no guarantee.
package official

import (
	"context"
	"sync"
)

// Doer is the transport seam: the subset of the UniFi client's public request
// methods the official wrappers need. Wrappers pass FULL controller paths
// (leading "/"), which the client resolves directly against its base URL,
// bypassing the legacy API-path prefix.
type Doer interface {
	Get(ctx context.Context, apiPath string, reqBody, respBody any) error
	Post(ctx context.Context, apiPath string, reqBody, respBody any) error
	Put(ctx context.Context, apiPath string, reqBody, respBody any) error
	Patch(ctx context.Context, apiPath string, reqBody, respBody any) error
	Delete(ctx context.Context, apiPath string, reqBody, respBody any) error
}

// Gate reports whether the official API may be used; it is evaluated before
// every operation and its error (unavailable/disabled) is returned verbatim.
type Gate func(ctx context.Context) error

// apiClient is the default Client implementation bound to an injected Doer.
type apiClient struct {
	doer     Doer
	basePath string // integration/v1 prefix, e.g. /proxy/network/integration/v1
	gate     Gate

	// siteIDs caches internalReference (legacy site name) -> Official-API UUID.
	mu      sync.RWMutex
	siteIDs map[string]string
}

// New constructs an official Client bound to doer, with basePath the
// integration/v1 prefix and gate the capability check run before each call.
func New(doer Doer, basePath string, gate Gate) Client {
	return &apiClient{doer: doer, basePath: basePath, gate: gate}
}

// check runs the capability gate (if any) before an operation.
func (c *apiClient) check(ctx context.Context) error {
	if c.gate == nil {
		return nil
	}
	return c.gate(ctx)
}

// path joins the integration/v1 base prefix with a leading-slash sub-path.
func (c *apiClient) path(sub string) string {
	return c.basePath + sub
}
