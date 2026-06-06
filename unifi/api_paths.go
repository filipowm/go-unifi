package unifi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	apiPath   = "/api"
	apiV2Path = "/v2/api"

	apiPathNew   = "/proxy/network/api"
	apiV2PathNew = "/proxy/network/v2/api"

	// integrationV1Path is the base prefix for the Official UniFi OpenAPI
	// (integration/v1). It is a capability layered on the new-style API, not a
	// fourth APIStyle: full leading-slash paths under it bypass APIPaths.ApiPath.
	integrationV1Path = "/proxy/network/integration/v1"

	loginPath    = "/api/login"
	loginPathNew = "/api/auth/login"

	statusPath    = "/status"
	statusPathNew = "/proxy/network/status"

	uploadPath    = "/upload"
	uploadPathNew = "/proxy/network/upload"

	logoutPath = "/api/logout"

	defaultUserAgent = "go-unifi/0.0.1"

	ApiKeyHeader      = "X-Api-Key" //nolint:gosec
	CsrfHeader        = "X-Csrf-Token"
	UserAgentHeader   = "User-Agent"
	AcceptHeader      = "Accept"
	ContentTypeHeader = "Content-Type"
)

// APIPaths defines the URL paths used by the client.
type APIPaths struct {
	ApiPath    string
	ApiV2Path  string
	LoginPath  string
	StatusPath string
	LogoutPath string
	UploadPath string
}

// OldStyleAPI and NewStyleAPI are the canonical path sets for the two controller
// API styles. They are compared by POINTER IDENTITY on the package-level addresses
// (&OldStyleAPI / &NewStyleAPI) by apiStyleFromStatus, apiPathsForStyle and
// determineApiStyle, so a *client's apiPaths can be identified back to a style.
//
// IMMUTABLE: treat these as read-only. Mutating a field of either corrupts every
// client and every parallel test that shares the pointer. Code (and tests) that
// needs an independent, mutable copy must call oldStyleAPI()/newStyleAPI(), which
// return fresh value copies.
var (
	OldStyleAPI = oldStyleAPI()
	NewStyleAPI = newStyleAPI()
)

// oldStyleAPI returns a fresh copy of the legacy (classic controller) API path
// set. Returning a value (not a shared pointer) lets callers/tests hold an
// independent APIPaths that can be mutated without corrupting the package-level
// OldStyleAPI used for style identity.
func oldStyleAPI() APIPaths {
	return APIPaths{
		ApiPath:    apiPath,
		ApiV2Path:  apiV2Path,
		LoginPath:  loginPath,
		StatusPath: statusPath,
		LogoutPath: logoutPath,
		UploadPath: uploadPath,
	}
}

// newStyleAPI returns a fresh copy of the new (UniFi OS / proxy) API path set.
// See oldStyleAPI for why this returns a value rather than a shared pointer.
func newStyleAPI() APIPaths {
	return APIPaths{
		ApiPath:    apiPathNew,
		ApiV2Path:  apiV2PathNew,
		LoginPath:  loginPathNew,
		StatusPath: statusPathNew,
		LogoutPath: logoutPath,
		UploadPath: uploadPathNew,
	}
}

// APIStyle selects which UniFi controller API style the client should use.
//
// The zero value, APIStyleAuto, preserves the historical behavior: the client
// probes the controller over the network at construction time to detect the
// style. APIStyleNew and APIStyleOld pin the style explicitly and SKIP the
// network probe, enabling fully offline client construction (the seam the
// hand-written wrapper tests rely on).
type APIStyle int

const (
	// APIStyleAuto auto-detects the API style by probing the controller (default).
	APIStyleAuto APIStyle = iota
	// APIStyleNew forces the new (UniFi OS / proxy) API style without probing.
	APIStyleNew
	// APIStyleOld forces the legacy (classic controller) API style without probing.
	APIStyleOld
)

// apiStyleFromStatus is the pure decision function behind determineApiStyle: it
// maps the controller's probe HTTP status (and whether API-key auth is in use)
// to the matching APIPaths, with zero network I/O so it can be unit-tested in
// isolation. A 200 means the new style; a 302 means the old style; anything else
// is an error. API-key auth is rejected against the old style because the
// classic controller does not support it.
func apiStyleFromStatus(status int, isAPIKey bool) (*APIPaths, error) {
	var paths *APIPaths
	switch status {
	case http.StatusOK:
		paths = &NewStyleAPI
	case http.StatusFound:
		paths = &OldStyleAPI
	default:
		return nil, fmt.Errorf("expected 200 or 302 status code, but got: %d", status)
	}

	if paths == &OldStyleAPI && isAPIKey {
		return nil, errors.New("unable to use API key authentication with old style API. Switch to user/pass authentication or update controller to latest version")
	}
	return paths, nil
}

// apiPathsForStyle returns the explicit APIPaths for a pinned (non-auto) style.
func apiPathsForStyle(style APIStyle) *APIPaths {
	if style == APIStyleOld {
		return &OldStyleAPI
	}
	return &NewStyleAPI
}

// determineApiStyle checks the base URL to decide which API style to use and sets the apiPaths accordingly.
//
// The probe is routed through c.http (cloned with a per-call CheckRedirect
// override) so it shares the same transport, timeout and cookie jar as every
// other request, rather than a throwaway *http.Client.
func (c *client) determineApiStyle() error {
	c.Debug("Determining API style")
	ctx, cancel := c.newRequestContext()
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL.String(), nil)
	if err != nil {
		return err
	}

	// Clone c.http so the probe shares transport/timeout/jar but does not follow
	// the controller's redirect (a 302 is the signal for the old-style API).
	probe := *c.http
	probe.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := probe.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Discard response body to avoid leaks
	_, _ = io.Copy(io.Discard, resp.Body)

	paths, err := apiStyleFromStatus(resp.StatusCode, c.credentials.IsAPIKey())
	if err != nil {
		return err
	}
	if paths == &NewStyleAPI {
		c.Debug("Using new style API")
	} else {
		c.Debug("Using old style API")
	}
	c.apiPaths = paths
	return nil
}
