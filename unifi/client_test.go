package unifi //nolint: testpackage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// verifyInterceptorPresence checks each expected interceptor type for presence or absence in the client.
func verifyInterceptorPresence(a *assert.Assertions, c *client, interceptors []any, shouldExist bool) {
	expectedTypes := make([]reflect.Type, 0, len(interceptors))
	for _, i := range interceptors {
		expectedTypes = append(expectedTypes, reflect.TypeOf(i))
	}
	for _, et := range expectedTypes {
		found := false
		for _, actual := range c.interceptors {
			if reflect.TypeOf(actual) == et {
				found = true
				break
			}
		}
		if shouldExist && !found {
			a.Fail(fmt.Sprintf("expected interceptor %v not found", et))
		}
		if !shouldExist && found {
			a.Fail(fmt.Sprintf("unexpected interceptor %v found", et))
		}
	}
}

func TestBareClientConstructor(t *testing.T) {
	t.Parallel()
	// A reachable URL that passes validation but fails at the API style probe
	// (connection refused) must return a nil client alongside the error.
	c, err := newClient(&ClientConfig{
		URL:    localUrl,
		APIKey: "test-key",
	})
	require.Error(t, err)
	require.Nil(t, c, "newClient must return nil client when construction fails")
	require.Contains(t, err.Error(), "connection refused", "an invalid destination should produce a connection error.")
}

func TestNewClientWithApiKey(t *testing.T) {
	t.Parallel()
	// when
	c, err := newClient(&ClientConfig{
		URL:    localUrl,
		APIKey: "test",
	})

	// then
	require.Error(t, err)
	require.Nil(t, c, "newClient must return nil client when construction fails")
	require.Contains(t, err.Error(), "connection refused", "an invalid destination should produce a connection error.")
}

func TestCustomizeHttpClient(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	called := false

	// when
	_, err := NewClient(&ClientConfig{
		URL:    localUrl,
		APIKey: "test-key",
		HttpTransportCustomizer: func(transport *http.Transport) (*http.Transport, error) {
			called = true
			return transport, nil
		},
	})

	// then
	require.Error(t, err)
	a.True(called, "http customizer not called")
}

func TestClientConfigValidationExecutedOnNewClient(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	cc := &ClientConfig{URL: "invalid URL"}
	// when
	c, err := NewClient(cc)
	// then
	require.ErrorContains(t, err, "validation failed")
	a.Nil(c)
}

// TestClientRejectsHTTPUrl verifies that NewClient rejects a plain http:// URL.
// An API key must never travel over plaintext; only https:// is accepted.
func TestClientRejectsHTTPUrl(t *testing.T) {
	t.Parallel()
	c, err := NewClient(&ClientConfig{
		URL:    "http://192.0.2.1",
		APIKey: "test-key",
	})
	require.Error(t, err)
	require.Nil(t, c)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestParseBaseUrl(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// Valid HTTPS URL without /api in the path.
	base, err := parseBaseURL("https://localhost")
	require.NoError(t, err)
	a.Equal("https", base.Scheme)
	a.Empty(base.Path)

	// http:// scheme is rejected (API key would be sent over plaintext).
	_, err = parseBaseURL("http://localhost")
	require.ErrorContains(t, err, "must use https://")

	// URL with trailing slash /api/
	_, err = parseBaseURL("https://localhost/api/")
	require.ErrorContains(t, err, "expected a base URL without the `/api`")

	// URL with /api in path (no trailing slash).
	_, err = parseBaseURL("https://localhost/api")
	require.ErrorContains(t, err, "expected a base URL without the `/api`")
}

// nonComparableInterceptor is a ClientInterceptor whose concrete type is NOT
// comparable with == (it holds a slice). Adding it must NOT panic the
// concrete-type dedup: containsInterceptorType compares reflect.TypeOf
// values, never the interface values themselves.
type nonComparableInterceptor struct {
	tags []string
}

func (nonComparableInterceptor) InterceptRequest(_ *http.Request) error   { return nil }
func (nonComparableInterceptor) InterceptResponse(_ *http.Response) error { return nil }

func TestRegisterInterceptor(t *testing.T) {
	t.Parallel()

	t.Run("dedup by concrete type", func(t *testing.T) {
		t.Parallel()
		// Two DISTINCT instances of the same concrete type collapse to one
		// (dedup is by concrete type, not by pointer identity).
		c := &client{interceptors: []ClientInterceptor{}}
		c.AddInterceptor(&TestInterceptor{})
		assert.Len(t, c.interceptors, 1)
		c.AddInterceptor(&TestInterceptor{})
		assert.Len(t, c.interceptors, 1, "a second instance of the same type must not be added")
	})

	t.Run("different types both kept", func(t *testing.T) {
		t.Parallel()
		c := &client{interceptors: []ClientInterceptor{}}
		c.AddInterceptor(&TestInterceptor{})
		c.AddInterceptor(&APIKeyAuthInterceptor{})
		assert.Len(t, c.interceptors, 2, "distinct concrete types must both be kept")
	})

	t.Run("nil is ignored", func(t *testing.T) {
		t.Parallel()
		c := &client{interceptors: []ClientInterceptor{}}
		c.AddInterceptor(nil)
		assert.Empty(t, c.interceptors, "a nil interceptor must be ignored")
	})

	t.Run("non-comparable type does not panic", func(t *testing.T) {
		t.Parallel()
		// A struct holding a slice is not comparable with ==; the old
		// slices.Contains dedup would panic. Concrete-type dedup must not.
		c := &client{interceptors: []ClientInterceptor{}}
		assert.NotPanics(t, func() {
			c.AddInterceptor(nonComparableInterceptor{tags: []string{"a"}})
			c.AddInterceptor(nonComparableInterceptor{tags: []string{"b"}})
		})
		assert.Len(t, c.interceptors, 1, "two non-comparable instances of one type collapse to one")
	})
}

func TestResolveAuthInterceptors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	auth := resolveAuthInterceptors(&ClientConfig{APIKey: "abc"}, NewDefaultLogger(InfoLevel))
	require.Len(t, auth, 1)
	apiKeyInterceptor, ok := auth[0].(*APIKeyAuthInterceptor)
	require.True(t, ok, "expected APIKeyAuthInterceptor")
	a.Equal("abc", apiKeyInterceptor.apiKey)
}

func TestBuildInterceptorsDedup(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// A single interceptor instance supplied twice must only be added once.
	dup := NewTestInterceptor()
	config := &ClientConfig{
		Interceptors: []ClientInterceptor{dup, dup},
	}
	auth := []ClientInterceptor{&APIKeyAuthInterceptor{apiKey: "test"}}
	interceptors := buildInterceptors(config, NewDefaultLogger(InfoLevel), auth)

	count := 0
	for _, i := range interceptors {
		if i == ClientInterceptor(dup) {
			count++
		}
	}
	a.Equal(1, count, "duplicate interceptor must only be added once")
}

// TestBuildInterceptorsDefaultsUserAgentWithoutMutatingConfig is the
// non-mutation guard for the User-Agent path: buildInterceptors must default an
// empty UserAgent into the produced DefaultHeadersInterceptor WITHOUT writing the
// default back through the caller-owned config.
func TestBuildInterceptorsDefaultsUserAgentWithoutMutatingConfig(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// Empty UserAgent: the default lands on the headers interceptor, but the
	// caller's config.UserAgent must stay empty (no write-back).
	config := &ClientConfig{}
	interceptors := buildInterceptors(config, NewDefaultLogger(InfoLevel), nil)
	a.Empty(config.UserAgent, "config.UserAgent must not be mutated")
	a.Equal(defaultUserAgent, headerUserAgent(t, interceptors), "default User-Agent must reach the headers interceptor")

	// Custom UserAgent: left untouched on the config and propagated to the headers.
	config = &ClientConfig{UserAgent: "custom-agent"}
	interceptors = buildInterceptors(config, NewDefaultLogger(InfoLevel), nil)
	a.Equal("custom-agent", config.UserAgent, "config.UserAgent must be left untouched")
	a.Equal("custom-agent", headerUserAgent(t, interceptors), "custom User-Agent must reach the headers interceptor")
}

// headerUserAgent extracts the User-Agent the DefaultHeadersInterceptor would set
// on a request, so the test asserts the resolved value without reaching through
// the caller's config.
func headerUserAgent(t *testing.T, interceptors []ClientInterceptor) string {
	t.Helper()
	for _, i := range interceptors {
		if dh, ok := i.(*DefaultHeadersInterceptor); ok {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test", http.NoBody)
			require.NoError(t, dh.InterceptRequest(req))
			return req.Header.Get(UserAgentHeader)
		}
	}
	t.Fatal("no DefaultHeadersInterceptor in chain")
	return ""
}

// TestNewClientFromConfigTrimsURLWithoutMutatingConfig is the non-mutation
// guard for the URL path: newClientFromConfig must normalize the trailing slash on
// the client's baseURL WITHOUT rewriting the caller-owned config.URL.
func TestNewClientFromConfigTrimsURLWithoutMutatingConfig(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// Trailing slashes are trimmed on the client's baseURL only; the caller's
	// config.URL must keep its trailing slashes (no write-back).
	config := &ClientConfig{
		URL:    testUrl + "///",
		APIKey: "test-key",
	}
	v, err := newValidator()
	require.NoError(t, err)
	c, err := newClientFromConfig(config, v)
	require.NoError(t, err)
	a.Equal(testUrl+"///", config.URL, "config.URL must not be mutated")
	a.Equal(testUrl, c.BaseURL(), "client baseURL must be normalized (trailing slash trimmed)")
}

// TestVersionWithLockingNoDeadlock is a regression test: Version()
// on a UseLocking:true client with an uncached sysInfo used to acquire c.lock and
// then re-enter it through executeRequest, self-deadlocking the goroutine. The
// fetch must now happen while holding no lock. A select + time.After makes a
// deadlock FAIL the test (timeout) rather than hang the whole suite.
func TestVersionWithLockingNoDeadlock(t *testing.T) {
	t.Parallel()

	const wantVersion = "9.9.9-test"
	cs := newControllerServer(t, sysinfoRoute(wantVersion))

	// The offline client leaves sysInfo uncached, so Version() takes the fetch
	// path — the exact path that previously deadlocked under UseLocking.
	c := cs.clientWith(func(cfg *ClientConfig) { cfg.UseLocking = true })

	done := make(chan string, 1)
	go func() {
		done <- c.Version()
	}()

	select {
	case got := <-done:
		assert.Equal(t, wantVersion, got, "Version() must return the controller version")
	case <-time.After(2 * time.Second):
		t.Fatal("Version() deadlocked: UseLocking:true client re-entered its own mutex")
	}
}

// TestVersionConcurrentCachedFetch is the load-bearing -race test:
// it hammers Version() from many goroutines at once so the race detector
// actually exercises the sysInfoMu RWMutex + double-checked locking that guards
// c.sysInfo. The single-goroutine TestVersionWithLockingNoDeadlock only proves
// no self-deadlock; only concurrent callers can surface a data race or torn read
// on c.sysInfo (e.g. if a refactor drops sysInfoMu or aliases it back to c.lock).
// Run with -race for this to bite.
func TestVersionConcurrentCachedFetch(t *testing.T) {
	t.Parallel()

	const (
		wantVersion = "9.9.9-test"
		goroutines  = 50
	)

	sysinfoPath := apiV1Path("s/default/stat/sysinfo")
	cs := newControllerServer(t, sysinfoRoute(wantVersion))

	// The offline client leaves sysInfo uncached, so the first Version() fetches.
	c := cs.clientWith(func(cfg *ClientConfig) { cfg.UseLocking = true })

	results := make([]string, goroutines)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			<-start // release all goroutines together to maximize contention
			results[idx] = c.Version()
		}(i)
	}
	close(start)
	wg.Wait()

	// Every concurrent caller must observe the full, untorn version string.
	for i, got := range results {
		assert.Equalf(t, wantVersion, got, "goroutine %d got a wrong/torn Version()", i)
	}

	// The burst against an initially-uncached client may legitimately race
	// several fetches (the HTTP fetch happens under NO lock; the double-check
	// only de-dupes the cache WRITE, not the fetch), so we only require >=1.
	// What MUST hold once the dust settles is that the cache is populated: a
	// post-burst Version() must serve from c.sysInfo without any new round-trip.
	burstHits := cs.countRequestsTo(sysinfoPath)
	assert.GreaterOrEqual(t, burstHits, 1, "at least one sysInfo fetch must occur")

	assert.Equal(t, wantVersion, c.Version(), "post-burst Version() must serve the cached version")
	assert.Equal(t, burstHits, cs.countRequestsTo(sysinfoPath), "cached Version() must not trigger another sysInfo fetch")
}

// TestBareClientDoesNotMutateConfig is the end-to-end guard: building a
// client from a config that carries a trailing-slash URL and an empty UserAgent
// must leave the CALLER's config byte-for-byte intact, while the constructed
// client behaves normalized — requests land on the trimmed URL. The APIStyle
// override keeps construction fully offline (no network probe).
func TestBareClientDoesNotMutateConfig(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	cs := newControllerServer(t, sysinfoRoute("9.9.9"))

	// Trailing slash on the URL and an EMPTY UserAgent: both are the fields
	// newClientFromConfig/buildInterceptors used to rewrite through the pointer.
	// The config itself is the subject under test, so build the client directly
	// from it rather than via the cs helper.
	srvTransport := cs.srv.Client().Transport
	config := &ClientConfig{
		URL:                      cs.srv.URL + "/",
		APIKey:                   "test-key",
		APIStyle:                 APIStyleNew, // offline: skip the network probe
		HttpRoundTripperProvider: func() http.RoundTripper { return srvTransport },
	}
	origURL := config.URL
	origUserAgent := config.UserAgent

	c, err := newClient(config)
	require.NoError(t, err)

	// The caller's struct must be untouched.
	a.Equal(origURL, config.URL, "config.URL must retain its trailing slash (no write-back)")
	a.Empty(origUserAgent, "precondition: UserAgent started empty")
	a.Empty(config.UserAgent, "config.UserAgent must stay empty (no default written back)")

	// The client itself is normalized: baseURL has no trailing slash and requests
	// reach the trimmed URL.
	a.Equal(cs.srv.URL, c.BaseURL(), "client baseURL must be the trimmed URL")
	_, err = c.GetSystemInformation()
	require.NoError(t, err)
	a.Equal(apiV1Path("s/default/stat/sysinfo"), cs.lastRequest().Path, "request must reach the normalized (trimmed) URL path")
}

// TestVersion pins the two Version() branches: the cached fast path
// returns the stored sysInfo version without any HTTP round-trip, and the
// error path swallows a failing sysinfo fetch into an empty string.
func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("cached fast path serves without a round-trip", func(t *testing.T) {
		t.Parallel()
		cs := newControllerServer(t) // no routes: any HTTP call would 404
		c := cs.client()
		c.sysInfo = &SysInfo{Version: "9.1.2-cached"}

		assert.Equal(t, "9.1.2-cached", c.Version())
		assert.Zero(t, cs.requestCount(), "a cached Version() must not issue any HTTP request")
	})

	t.Run("fetch error returns empty string", func(t *testing.T) {
		t.Parallel()
		// The sysinfo endpoint 500s, so the fetch fails and Version() swallows the
		// error into "". Any non-sysinfo path also 404s.
		cs := newControllerServer(t, route{
			path: apiV1Path("s/default/stat/sysinfo"),
			fn: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		})
		c := cs.client()

		assert.Empty(t, c.Version(), "a failing sysinfo fetch must make Version() return an empty string")
	})
}

// TestNewClientSkipSystemInfoNoRoundTrip asserts that NewClient with SkipSystemInfo:true
// issues zero sysinfo requests during construction. Only the style-probe (if APIStyleAuto)
// or no requests at all (pinned APIStyle) should occur.
func TestNewClientSkipSystemInfoNoRoundTrip(t *testing.T) {
	t.Parallel()

	cs := newControllerServer(t, sysinfoRoute("9.9.9"))

	_, err := cs.newClientWith(func(cfg *ClientConfig) { cfg.SkipSystemInfo = true })
	require.NoError(t, err)
	assert.Zero(t, cs.countRequestsTo(apiV1Path("s/default/stat/sysinfo")), "NewClient with SkipSystemInfo:true must not fetch sysinfo at construction")
}

// TestNewClientSkipSystemInfoDeferredError verifies the core SkipSystemInfo contract:
// construction must succeed even when the controller is unreachable or returns an auth
// error, and the failure must then surface on the first Version/API call.
func TestNewClientSkipSystemInfoDeferredError(t *testing.T) {
	t.Parallel()

	t.Run("unreachable controller defers error to first API call", func(t *testing.T) {
		t.Parallel()
		// localUrl points at an unreachable port, so every connection attempt gets
		// "connection refused".
		c, err := NewClient(&ClientConfig{
			URL:            localUrl,
			APIKey:         "dummy",
			APIStyle:       APIStyleNew, // pinned: no style probe
			SkipSystemInfo: true,
		})
		require.NoError(t, err, "NewClient with SkipSystemInfo:true must succeed even when controller is unreachable")

		_, err = c.VersionContext(context.Background())
		require.Error(t, err, "VersionContext must surface the deferred connection error")
	})

	t.Run("bad API key defers auth error to first API call", func(t *testing.T) {
		t.Parallel()
		// The sysinfo endpoint 401s, so the deferred fetch on the first API call fails.
		cs := newControllerServer(t, route{apiV1Path("s/default/stat/sysinfo"), func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}})

		c, err := cs.newClientWith(func(cfg *ClientConfig) {
			cfg.APIKey = "bad-key"
			cfg.SkipSystemInfo = true
		})
		require.NoError(t, err, "NewClient with SkipSystemInfo:true must succeed even with a bad API key")

		_, err = c.VersionContext(context.Background())
		require.Error(t, err, "VersionContext must surface the deferred auth error")
	})
}

func TestHttpTransportCustomizerError(t *testing.T) {
	t.Parallel()
	customizer := func(transport *http.Transport) (*http.Transport, error) {
		return nil, errors.New("customization failed")
	}
	_, err := NewClient(&ClientConfig{
		URL:                     testUrl,
		APIKey:                  "test-key",
		HttpTransportCustomizer: customizer,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed customizing HTTP transport")
}
