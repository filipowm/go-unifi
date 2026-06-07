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

// TestApiStyleFromStatus covers every branch of the pure decision function:
// 200 maps to the new-style API; 302 (classic/old-style) is rejected as
// unsupported; any other status is an error.
func TestApiStyleFromStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status    int
		wantPaths *APIPaths
		wantErr   string
	}{
		"200 -> new style": {
			status:    http.StatusOK,
			wantPaths: &NewStyleAPI,
		},
		"302 -> old-style unsupported": {
			status:  http.StatusFound,
			wantErr: "old-style (classic) controllers are unsupported",
		},
		"500 -> error": {
			status:  http.StatusInternalServerError,
			wantErr: "expected 200 or 302 status code, but got: 500",
		},
		"401 -> error": {
			status:  http.StatusUnauthorized,
			wantErr: "expected 200 or 302 status code, but got: 401",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			paths, err := apiStyleFromStatus(tc.status)
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
// offline-construction seam): when APIStyleNew is set, no network probe is made,
// so the client constructs against an unreachable URL without error and pins the
// requested paths.
func TestApiStyleOverrideSkipsProbe(t *testing.T) {
	t.Parallel()
	// localUrl points at an unreachable port: if the probe ran, construction
	// would fail with a connection error. APIStyle skips it entirely.
	c, err := newBareClient(&ClientConfig{
		URL:      localUrl,
		APIKey:   "test-key",
		APIStyle: APIStyleNew,
	})
	require.NoError(t, err)
	assert.Same(t, &NewStyleAPI, c.apiPaths)
}

// TestApiStyleOverrideOldStyleIsUnsupported ensures that pinning APIStyleOld
// fails immediately — classic controllers are unsupported after API-key-only auth.
func TestApiStyleOverrideOldStyleIsUnsupported(t *testing.T) {
	t.Parallel()
	_, err := newBareClient(&ClientConfig{
		URL:      localUrl,
		APIKey:   "test-key",
		APIStyle: APIStyleOld,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "old-style (classic) controllers are unsupported")
}

// TestApiStyleSetCopiesAreIsolated pins the value-returning seam:
// oldStyleAPI()/newStyleAPI() return fresh copies equal to the canonical package
// vars, and mutating a returned copy must NOT corrupt the shared OldStyleAPI /
// NewStyleAPI used for pointer-identity style detection.
func TestApiStyleSetCopiesAreIsolated(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// The copies equal the canonical sets by value.
	a.Equal(OldStyleAPI, oldStyleAPI())
	a.Equal(NewStyleAPI, newStyleAPI())

	// The two styles are genuinely different sets.
	a.NotEqual(oldStyleAPI(), newStyleAPI())

	// Mutating a local copy must not bleed into the package-level globals.
	cp := newStyleAPI()
	cp.ApiPath = "/corrupted"
	a.Equal(apiPathNew, NewStyleAPI.ApiPath, "mutating a copy must not corrupt the shared NewStyleAPI")
	a.Equal(apiPathNew, newStyleAPI().ApiPath, "each call returns a pristine copy")

	cpOld := oldStyleAPI()
	cpOld.ApiPath = "/corrupted"
	a.Equal(apiPath, OldStyleAPI.ApiPath, "mutating a copy must not corrupt the shared OldStyleAPI")
}

// TestApiPathsForStyle pins the pinned-style->paths mapping: the old
// style resolves to the &OldStyleAPI identity and the new style (and the auto
// default) resolve to &NewStyleAPI, preserving the pointer-identity contract the
// rest of the client relies on.
func TestApiPathsForStyle(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	a.Same(&OldStyleAPI, apiPathsForStyle(APIStyleOld))
	a.Same(&NewStyleAPI, apiPathsForStyle(APIStyleNew))
	a.Same(&NewStyleAPI, apiPathsForStyle(APIStyleAuto), "auto defaults to the new style set")
}

// TestDetermineApiStyle_OldStyleIsUnsupported exercises the 302 -> unsupported
// branch end to end against an httptest server that redirects at the root. The
// probe must NOT follow the redirect and must return an error.
func TestDetermineApiStyle_OldStyleIsUnsupported(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/manage", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_, err := newBareClient(&ClientConfig{
		URL:    ts.URL,
		APIKey: "test-key",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "old-style (classic) controllers are unsupported")
}
