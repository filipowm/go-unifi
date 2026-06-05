package unifi //nolint: testpackage

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineApiStyle_InvalidStatus(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return an unexpected status code.
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := NewClient(&ClientConfig{
		URL:    ts.URL,
		APIKey: "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected 200 or 302 status code")
}

// TestApiStyleFromStatus covers every branch of the pure decision function
// extracted for offline testability (TEST-09): 200/302/other x apikey/userpass,
// including the 'cannot use API key with old-style API' guard.
func TestApiStyleFromStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status    int
		isAPIKey  bool
		wantPaths *APIPaths
		wantErr   string
	}{
		"200 user/pass -> new style": {
			status:    http.StatusOK,
			isAPIKey:  false,
			wantPaths: &NewStyleAPI,
		},
		"200 api key -> new style": {
			status:    http.StatusOK,
			isAPIKey:  true,
			wantPaths: &NewStyleAPI,
		},
		"302 user/pass -> old style": {
			status:    http.StatusFound,
			isAPIKey:  false,
			wantPaths: &OldStyleAPI,
		},
		"302 api key -> rejected": {
			status:   http.StatusFound,
			isAPIKey: true,
			wantErr:  "unable to use API key authentication with old style API",
		},
		"500 user/pass -> error": {
			status:   http.StatusInternalServerError,
			isAPIKey: false,
			wantErr:  "expected 200 or 302 status code, but got: 500",
		},
		"500 api key -> error": {
			status:   http.StatusInternalServerError,
			isAPIKey: true,
			wantErr:  "expected 200 or 302 status code, but got: 500",
		},
		"401 user/pass -> error": {
			status:   http.StatusUnauthorized,
			isAPIKey: false,
			wantErr:  "expected 200 or 302 status code, but got: 401",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			paths, err := apiStyleFromStatus(tc.status, tc.isAPIKey)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Nil(t, paths)
				return
			}
			require.NoError(t, err)
			assert.Same(t, tc.wantPaths, paths)
		})
	}
}

// TestApiStyleOverrideSkipsProbe proves the ClientConfig.APIStyle override (the
// offline-construction seam, TEST-09): when set, no network probe is made, so
// the client constructs against an unreachable URL without error and pins the
// requested paths.
func TestApiStyleOverrideSkipsProbe(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		style     APIStyle
		wantPaths *APIPaths
	}{
		"new style override": {style: APIStyleNew, wantPaths: &NewStyleAPI},
		"old style override": {style: APIStyleOld, wantPaths: &OldStyleAPI},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// localUrl points at an unreachable port: if the probe ran, construction
			// would fail with a connection error. APIStyle skips it entirely.
			c, err := newBareClient(&ClientConfig{
				URL:      localUrl,
				User:     "admin",
				Password: "password",
				APIStyle: tc.style,
			})
			require.NoError(t, err)
			assert.Same(t, tc.wantPaths, c.apiPaths)
		})
	}
}

// TestApiStyleOverrideOldStyleRejectsAPIKey ensures the API-key-vs-old-style
// guard still applies when the style is pinned offline rather than probed.
func TestApiStyleOverrideOldStyleRejectsAPIKey(t *testing.T) {
	t.Parallel()
	_, err := newBareClient(&ClientConfig{
		URL:      localUrl,
		APIKey:   "test-key",
		APIStyle: APIStyleOld,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to use API key authentication with old style API")
}

// TestDetermineApiStyle_OldStyle exercises the 302 -> old-style branch end to
// end against an httptest server that redirects at the root, proving the probe
// (now routed through a clone of c.http) does NOT follow the redirect.
func TestDetermineApiStyle_OldStyle(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/manage", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := newBareClient(&ClientConfig{
		URL:      ts.URL,
		User:     "admin",
		Password: "password",
	})
	require.NoError(t, err)
	assert.Same(t, &OldStyleAPI, c.apiPaths)
}
