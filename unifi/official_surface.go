package unifi

import (
	"context"
	"fmt"

	"github.com/filipowm/go-unifi/unifi/official"
	goversion "github.com/hashicorp/go-version"
)

// officialAPIMinVersion is the first controller version exposing the Official
// UniFi OpenAPI (integration/v1).
const officialAPIMinVersion = "10.1.68"

// Internal returns the client itself as the InternalClient (legacy UniFi Network
// API) surface. In 2.0.0 this is the canonical client; calling it documents
// intent ahead of the 3.0.0 flip to the Official client as the default.
func (c *client) Internal() InternalClient { //nolint:ireturn
	return c
}

// Official returns the Official UniFi OpenAPI client bound to this client's
// transport and gated on controller capability. The instance (and its
// site-resolver cache) is built once and reused.
func (c *client) Official() official.Client { //nolint:ireturn
	c.officialOnce.Do(func() {
		c.officialClient = official.New(c, integrationV1Path, c.officialAvailable)
	})
	return c.officialClient
}

// officialAvailable is the capability gate for the Official API. It fails fast
// when opted out, requires a new-style controller with API-key auth, and
// otherwise probes GET /v1/info once to confirm the version floor. A successful
// probe is cached so subsequent operations skip it.
func (c *client) officialAvailable(ctx context.Context) error {
	if c.officialDisabled {
		return ErrOfficialAPIDisabled
	}
	if c.apiPaths != &NewStyleAPI || !c.credentials.IsAPIKey() {
		return fmt.Errorf("%w: requires a new-style controller with API-key authentication", ErrOfficialAPIUnavailable)
	}

	c.officialReadyMu.Lock()
	defer c.officialReadyMu.Unlock()
	if c.officialReady {
		return nil
	}

	var info official.Info
	if err := c.Get(ctx, integrationV1Path+"/info", nil, &info); err != nil {
		return fmt.Errorf("%w: %w", ErrOfficialAPIUnavailable, err)
	}
	if !versionAtLeast(info.ApplicationVersion, officialAPIMinVersion) {
		return fmt.Errorf("%w: controller version %q is below the required %s", ErrOfficialAPIUnavailable, info.ApplicationVersion, officialAPIMinVersion)
	}
	c.officialReady = true
	return nil
}

// versionAtLeast reports whether have is a parseable version >= minVersion. An
// unparseable have is treated as below the floor.
func versionAtLeast(have, minVersion string) bool {
	h, err := goversion.NewVersion(have)
	if err != nil {
		return false
	}
	return h.GreaterThanOrEqual(goversion.Must(goversion.NewVersion(minVersion)))
}
