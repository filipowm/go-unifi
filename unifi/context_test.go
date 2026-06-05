package unifi //nolint: testpackage

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cancelledContext returns a context that has already been cancelled, so any
// HTTP round-trip threaded with it must abort immediately.
func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// TestContextVariantsAbortOnCancelledContext proves that the four ctx-accepting
// variants thread the supplied context through to the HTTP layer: a pre-cancelled
// context aborts the request before it completes (TEST-15). Each subtest asserts
// errors.Is(err, context.Canceled) through the wrapped error chain.
func TestContextVariantsAbortOnCancelledContext(t *testing.T) {
	t.Parallel()

	// A handler that should never be reached when the context is already
	// cancelled — the abort must happen client-side before the round-trip.
	okHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"data": [{"version": "9.9.9-test"}]}`)
	}

	tests := map[string]struct {
		// call exercises one ctx-accepting variant with the supplied context and
		// returns only the error (value results are irrelevant for the abort path).
		call func(c *client, ctx context.Context) error
		// apiKey selects API-key auth; user/pass auth is needed so Login/Logout do
		// not short-circuit before the round-trip.
		apiKey bool
	}{
		"VersionContext": {
			call: func(c *client, ctx context.Context) error {
				_, err := c.VersionContext(ctx)
				return err
			},
			apiKey: true,
		},
		"GetSystemInformationContext": {
			call: func(c *client, ctx context.Context) error {
				_, err := c.GetSystemInformationContext(ctx)
				return err
			},
			apiKey: true,
		},
		"LoginContext": {
			call: func(c *client, ctx context.Context) error {
				return c.LoginContext(ctx)
			},
		},
		"LogoutContext": {
			call: func(c *client, ctx context.Context) error {
				return c.LogoutContext(ctx)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Each parallel subtest gets its own server+client: controllerServer
			// records requests on an unsynchronized slice (testhelpers_test.go), so
			// it must not be shared across concurrently-running subtests.
			cs := newControllerServer(t,
				route{path: apiV1Path("s/default/stat/sysinfo"), fn: okHandler},
				route{path: NewStyleAPI.LoginPath, fn: okHandler},
				route{path: NewStyleAPI.LogoutPath, fn: okHandler},
			)

			var c *client
			if tc.apiKey {
				c = cs.client()
			} else {
				c = cs.clientUserPass()
			}

			err := tc.call(c, cancelledContext())
			require.Error(t, err, "a pre-cancelled context must abort the request")
			assert.ErrorIs(t, err, context.Canceled, "the cancellation must surface through the wrapped error chain")
		})
	}
}

// TestVersionContextHappyPath proves VersionContext fetches and returns the
// controller version when the cache is empty and a valid context is supplied.
func TestVersionContextHappyPath(t *testing.T) {
	t.Parallel()

	const wantVersion = "9.5.21-test"
	cs := newControllerServer(t,
		route{path: apiV1Path("s/default/stat/sysinfo"), fn: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"data": [{"version": "%s"}]}`, wantVersion)
		}},
	)
	c := cs.client()

	got, err := c.VersionContext(context.Background())
	require.NoError(t, err, "VersionContext must surface no error on the happy path")
	assert.Equal(t, wantVersion, got)
}

// TestVersionContextFetchErrorSurfaces proves the NON-cancellation fetch-error
// slow path (TEST-15): with an empty cache and a sysinfo endpoint that 500s,
// GetSystemInformationContext errors and VersionContext must surface the empty
// string AND the error (unlike Version(), which swallows it). The surfaced error
// is the *ServerError carrying the 500 status.
func TestVersionContextFetchErrorSurfaces(t *testing.T) {
	t.Parallel()

	cs := newControllerServer(t,
		route{path: apiV1Path("s/default/stat/sysinfo"), fn: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}},
	)
	c := cs.client()

	got, err := c.VersionContext(context.Background())
	assert.Empty(t, got, "a failing sysinfo fetch must yield an empty version string")
	require.Error(t, err, "VersionContext must surface the fetch error rather than swallow it")

	var serverErr *ServerError
	require.ErrorAs(t, err, &serverErr)
	assert.Equal(t, http.StatusInternalServerError, serverErr.StatusCode)
}

// TestVersionContextCachedFastPath proves the pure cache-decision half
// (cachedVersion) short-circuits VersionContext: a pre-populated sysInfo cache is
// returned without any HTTP round-trip, and importantly without consulting the
// supplied context — so even a cancelled context yields the cached value
// (TEST-15: cached-vs-fetch branch is testable without timing hacks).
func TestVersionContextCachedFastPath(t *testing.T) {
	t.Parallel()

	const cachedVersion = "1.2.3-cached"
	var hits atomic.Int32
	cs := newControllerServer(t,
		route{path: apiV1Path("s/default/stat/sysinfo"), fn: func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"data": [{"version": "should-not-be-fetched"}]}`)
		}},
	)
	c := cs.client()

	// Pre-populate the cache under the same write lock VersionContext uses.
	c.sysInfoMu.Lock()
	c.sysInfo = &SysInfo{Version: cachedVersion}
	c.sysInfoMu.Unlock()

	// The pure cache-decision half reports the populated cache directly.
	v, ok := c.cachedVersion()
	require.True(t, ok, "cachedVersion must report the cache as populated")
	assert.Equal(t, cachedVersion, v)

	// Even a cancelled context must not trigger a fetch when the cache is warm.
	got, err := c.VersionContext(cancelledContext())
	require.NoError(t, err, "the cached fast path must not consult the context or hit the network")
	assert.Equal(t, cachedVersion, got)
	assert.Zero(t, hits.Load(), "the cached fast path must perform no sysInfo round-trip")
}

// TestGetSystemInformationContextHappyPath proves GetSystemInformationContext
// returns the parsed SysInfo on the new-API path with a valid context.
func TestGetSystemInformationContextHappyPath(t *testing.T) {
	t.Parallel()

	cs := newControllerServer(t,
		route{path: apiV1Path("s/default/stat/sysinfo"), fn: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"data": [{"version": "9.5.21-test", "name": "Dream Machine"}]}`)
		}},
	)
	c := cs.client()

	info, err := c.GetSystemInformationContext(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "9.5.21-test", info.Version)
	assert.Equal(t, "Dream Machine", info.Name)
}

// TestLoginContextHappyPath proves LoginContext performs the user/pass login POST
// successfully with a valid context.
func TestLoginContextHappyPath(t *testing.T) {
	t.Parallel()

	cs := newControllerServer(t,
		route{path: NewStyleAPI.LoginPath, fn: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		}},
	)
	c := cs.clientUserPass()

	require.NoError(t, c.LoginContext(context.Background()))
	assert.Equal(t, NewStyleAPI.LoginPath, cs.lastRequest().Path)
}

// TestLogoutContextHappyPath proves LogoutContext performs the user/pass logout
// POST successfully with a valid context.
func TestLogoutContextHappyPath(t *testing.T) {
	t.Parallel()

	cs := newControllerServer(t,
		route{path: NewStyleAPI.LogoutPath, fn: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		}},
	)
	c := cs.clientUserPass()

	require.NoError(t, c.LogoutContext(context.Background()))
	assert.Equal(t, NewStyleAPI.LogoutPath, cs.lastRequest().Path)
}

// TestLoginLogoutContextAPIKeyShortCircuit proves the ctx variants skip the
// round-trip entirely for API-key auth (mirroring the legacy Login/Logout), so a
// cancelled context is irrelevant when there is nothing to send.
func TestLoginLogoutContextAPIKeyShortCircuit(t *testing.T) {
	t.Parallel()

	cs := newControllerServer(t)
	c := cs.client() // API-key auth

	require.NoError(t, c.LoginContext(cancelledContext()), "API-key LoginContext must short-circuit")
	require.NoError(t, c.LogoutContext(cancelledContext()), "API-key LogoutContext must short-circuit")
	assert.Zero(t, cs.requestCount(), "API-key auth must perform no login/logout round-trip")
}
