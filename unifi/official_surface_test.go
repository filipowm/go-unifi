package unifi //nolint:testpackage

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	officialInfoPath  = integrationV1Path + "/info"
	officialSitesPath = integrationV1Path + "/sites"
)

// infoRoute serves GET /v1/info with the given application version.
func infoRoute(version string) route {
	return route{officialInfoPath, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"applicationVersion":"` + version + `"}`))
	}}
}

func TestOfficialGetInfo(t *testing.T) {
	t.Parallel()
	cs := newControllerServer(t, infoRoute("10.1.78"))
	c := cs.client()

	info, err := c.Official().GetInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "10.1.78", info.ApplicationVersion)
}

func TestOfficialResolveSiteID(t *testing.T) {
	t.Parallel()
	cs := newControllerServer(t,
		infoRoute("10.1.78"),
		route{officialSitesPath, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"offset":0,"limit":200,"count":2,"totalCount":2,"data":[` +
				`{"id":"uuid-default","internalReference":"default","name":"Default"},` +
				`{"id":"uuid-lab","internalReference":"lab","name":"Lab"}]}`))
		}},
	)
	c := cs.client()

	id, err := c.Official().ResolveSiteID(context.Background(), "lab")
	require.NoError(t, err)
	assert.Equal(t, "uuid-lab", id)
}

func TestOfficialGateUnavailableBelowVersionFloor(t *testing.T) {
	t.Parallel()
	cs := newControllerServer(t, infoRoute("10.1.67"))
	c := cs.client()

	_, err := c.Official().GetInfo(context.Background())
	require.ErrorIs(t, err, ErrOfficialAPIUnavailable)
}

func TestOfficialGateUnavailableOldStyle(t *testing.T) {
	t.Parallel()
	// Old-style controllers are unsupported for the Official API; user/pass auth
	// is required there since API-key + old-style is rejected at construction.
	c := newOfflineClient(t, &ClientConfig{
		URL:      testUrl,
		User:     "u",
		Password: "p",
		APIStyle: APIStyleOld,
	})

	_, err := c.Official().GetInfo(context.Background())
	require.ErrorIs(t, err, ErrOfficialAPIUnavailable)
}

func TestOfficialGateDisabled(t *testing.T) {
	t.Parallel()
	c := newOfflineClient(t, &ClientConfig{
		URL:                testUrl,
		APIKey:             "test-key",
		APIStyle:           APIStyleNew,
		DisableOfficialAPI: true,
	})

	_, err := c.Official().GetInfo(context.Background())
	require.ErrorIs(t, err, ErrOfficialAPIDisabled)
}

// TestOfficialGateProbeIsCached asserts the at-most-once /v1/info contract: the
// capability probe fires exactly once even across multiple Official() operations.
// GetInfo hits /v1/info twice on a cold cache (probe + real call); the second
// operation (ResolveSiteID) must reuse the cached gate result and not re-probe.
func TestOfficialGateProbeIsCached(t *testing.T) {
	t.Parallel()
	cs := newControllerServer(t,
		infoRoute("10.1.78"),
		route{officialSitesPath, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"offset":0,"limit":200,"count":1,"totalCount":1,"data":[` +
				`{"id":"uuid-default","internalReference":"default","name":"Default"}]}`))
		}},
	)
	c := cs.client()

	// First operation: gate probe + real GetInfo both hit /v1/info (two total).
	_, err := c.Official().GetInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, cs.countRequestsTo(officialInfoPath), "cold GetInfo: probe + real call = 2 hits on /v1/info")

	// Second operation: gate must short-circuit (officialReady=true), no new probe.
	_, err = c.Official().ResolveSiteID(context.Background(), "default")
	require.NoError(t, err)
	assert.Equal(t, 2, cs.countRequestsTo(officialInfoPath), "subsequent op must not re-probe /v1/info")
}

// TestOfficialGateUnavailableEndpointAbsent covers the realistic old-controller
// case where integration/v1 is entirely absent (404) — the probe fails and the
// error wraps ErrOfficialAPIUnavailable.
func TestOfficialGateUnavailableEndpointAbsent(t *testing.T) {
	t.Parallel()
	// No routes registered: unmatched /v1/info returns 404, exercising the probe-error path.
	cs := newControllerServer(t)
	c := cs.client()

	_, err := c.Official().GetInfo(context.Background())
	require.ErrorIs(t, err, ErrOfficialAPIUnavailable)
}

// TestVersionAtLeastUnparseable asserts that an empty or garbage version is treated
// as below the floor (fail-closed) so the gate rejects unknown controller builds.
func TestVersionAtLeastUnparseable(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"", "not-a-version", "???"} {
		assert.False(t, versionAtLeast(v, officialAPIMinVersion), "unparseable %q must be below floor", v)
	}
}

// TestOfficialGateUnavailableNewStyleUserPass asserts that a new-style controller
// with user/password auth (no API key) returns ErrOfficialAPIUnavailable.
func TestOfficialGateUnavailableNewStyleUserPass(t *testing.T) {
	t.Parallel()
	c := newOfflineClient(t, &ClientConfig{
		URL:      testUrl,
		User:     "u",
		Password: "p",
		APIStyle: APIStyleNew,
	})

	_, err := c.Official().GetInfo(context.Background())
	require.ErrorIs(t, err, ErrOfficialAPIUnavailable)
}

// TestInternalAccessorReturnsResourceSurface proves client.Internal() exposes the
// same resource CRUD the top-level client embeds.
func TestInternalAccessorReturnsResourceSurface(t *testing.T) {
	t.Parallel()
	cs := newControllerServer(t)
	c := cs.client()

	require.NotNil(t, c.Internal())
	// Compile-time proof the top-level client also satisfies InternalClient (the
	// embedded resource CRUD surface is reachable both ways).
	var _ InternalClient = c
}
