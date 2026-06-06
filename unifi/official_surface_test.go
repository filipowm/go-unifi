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
	cs := newControllerServer(t, infoRoute("10.1.68"))
	c := cs.client()

	info, err := c.Official().GetInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "10.1.68", info.ApplicationVersion)
}

func TestOfficialResolveSiteID(t *testing.T) {
	t.Parallel()
	cs := newControllerServer(t,
		infoRoute("10.1.68"),
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
