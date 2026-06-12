package unifi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
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

	statusPath    = "/status"
	statusPathNew = "/proxy/network/status"

	uploadPath    = "/upload"
	uploadPathNew = "/proxy/network/upload"

	ApiKeyHeader      = "X-Api-Key" //nolint:gosec
	UserAgentHeader   = "User-Agent"
	AcceptHeader      = "Accept"
	ContentTypeHeader = "Content-Type"
)

var defaultUserAgent = buildUserAgent()

// buildUserAgent returns a User-Agent string derived from the module version
// reported by debug.ReadBuildInfo. When used as a dependency, it reads the
// version from the dep entry; when built as the main module during development
// it reads info.Main.Version. Falls back to "go-unifi/2" when no version is
// available.
func buildUserAgent() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/filipowm/go-unifi/v2" {
				return "go-unifi/" + dep.Version
			}
		}
		// When built as the main module (e.g. during development), use the main module version.
		if info.Main.Path == "github.com/filipowm/go-unifi/v2" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			return "go-unifi/" + info.Main.Version
		}
	}
	return "go-unifi/2"
}

// APIPaths defines the URL paths used by the client.
type APIPaths struct {
	ApiPath    string
	ApiV2Path  string
	StatusPath string
	UploadPath string
}

// OldStyleAPI and NewStyleAPI are the canonical path sets for the two controller
// API styles. They are compared by POINTER IDENTITY on the package-level addresses
// (&OldStyleAPI / &NewStyleAPI) by apiStyleFromStatus and determineApiStyle, so a
// *client's apiPaths can be identified back to a style.
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
		StatusPath: statusPath,
		UploadPath: uploadPath,
	}
}

// newStyleAPI returns a fresh copy of the new (UniFi OS / proxy) API path set.
// See oldStyleAPI for why this returns a value rather than a shared pointer.
func newStyleAPI() APIPaths {
	return APIPaths{
		ApiPath:    apiPathNew,
		ApiV2Path:  apiV2PathNew,
		StatusPath: statusPathNew,
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

// ErrOldStyleUnsupported is returned when a client targets an old-style (classic)
// controller. API-key authentication — the only supported auth as of 2.0.0 —
// requires a controller new enough to expose the new-style API (UniFi Network
// 9.0.114 or newer). Callers can match it with errors.Is.
var ErrOldStyleUnsupported = errors.New("old-style (classic) controllers are unsupported; API-key authentication requires UniFi Network 9.0.114 or newer")

// apiStyleFromStatus is the pure decision function behind determineApiStyle: it
// maps the controller's probe HTTP status to the matching APIPaths, with zero
// network I/O so it can be unit-tested in isolation. A 200 means the new style;
// a 302 indicates a classic (old-style) controller which is no longer supported.
func apiStyleFromStatus(status int) (*APIPaths, error) {
	switch status {
	case http.StatusOK:
		return &NewStyleAPI, nil
	case http.StatusFound:
		return nil, ErrOldStyleUnsupported
	default:
		return nil, fmt.Errorf("expected 200 or 302 status code, but got: %d", status)
	}
}

// determineApiStyle checks the base URL to decide which API style to use and sets the apiPaths accordingly.
//
// The probe is routed through c.http (cloned with a per-call CheckRedirect
// override) so it shares the same transport, timeout and cookie jar as every
// other request, rather than a throwaway *http.Client.
func (c *client) determineApiStyle() error {
	c.log.Debug("Determining API style")
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

	paths, err := apiStyleFromStatus(resp.StatusCode)
	if err != nil {
		return err
	}
	if paths == &NewStyleAPI {
		c.log.Debug("Using new style API")
	} else {
		c.log.Debug("Using old style API")
	}
	c.apiPaths = paths
	return nil
}
