package unifi //nolint: testpackage

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
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

func TestNewBareClient(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	c, err := newBareClient(&ClientConfig{
		URL:      localUrl,
		User:     "admin",
		Password: "password",
	})
	require.Error(t, err)
	a.Equal(localUrl, c.BaseURL())
	a.Contains(err.Error(), "connection refused", "an invalid destination should produce a connection error.")
	verifyInterceptorPresence(a, c, []any{&CSRFInterceptor{}, &DefaultHeadersInterceptor{}}, true)
	verifyInterceptorPresence(a, c, []any{&APIKeyAuthInterceptor{}}, false)
}

func TestNewClientWithApiKey(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// when
	c, err := newBareClient(&ClientConfig{
		URL:    localUrl,
		APIKey: "test",
	})

	// then
	require.Error(t, err)
	a.Equal(localUrl, c.BaseURL())
	a.Contains(err.Error(), "connection refused", "an invalid destination should produce a connection error.")
	verifyInterceptorPresence(a, c, []any{&APIKeyAuthInterceptor{}, &DefaultHeadersInterceptor{}}, true)
	verifyInterceptorPresence(a, c, []any{&CSRFInterceptor{}}, false)
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

func TestUnifiIntegrationUserPassInjected(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// given
	srv := runTestServer(NewStyleAPI.LoginPath)
	interceptor := NewTestInterceptor()
	c := newNewStyleClient(&ClientConfig{
		URL:          srv.URL,
		User:         "test-user",
		Password:     "test-pass",
		Interceptors: interceptor.AsList(),
	})

	// when
	err := c.Login()

	// then
	require.NoError(t, err, "user/pass login must not produce an error")
	a.Equal(http.MethodPost, interceptor.Method())
	a.Equal(http.StatusOK, interceptor.response.StatusCode)
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

func TestParseBaseUrl(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// Valid URL without /api in the path.
	base, err := parseBaseURL("http://localhost")
	require.NoError(t, err)
	a.Equal("http", base.Scheme)
	a.Empty(base.Path)

	// URL with trailing slash /api/
	_, err = parseBaseURL("http://localhost/api/")
	require.ErrorContains(t, err, "expected a base URL without the `/api`")

	// URL with /api in path (no trailing slash).
	_, err = parseBaseURL("http://localhost/api")
	require.ErrorContains(t, err, "expected a base URL without the `/api`")
}

func TestRegisterInterceptor(t *testing.T) {
	t.Parallel()
	// Create a manual client with an empty interceptor slice.
	client := &client{
		interceptors: []ClientInterceptor{},
	}
	// Create a dummy interceptor (using TestInterceptor already defined in the file).
	var dummy ClientInterceptor = &TestInterceptor{}
	initialCount := len(client.interceptors)
	client.AddInterceptor(&dummy)
	assert.Len(t, client.interceptors, initialCount+1)
	// Attempt to add the same interceptor again.
	client.AddInterceptor(&dummy)
	assert.Len(t, client.interceptors, initialCount+1)
}

func TestLoginWithAPIKeyDirect(t *testing.T) {
	t.Parallel()
	// Create a client manually with the APIKey set.

	c, err := newBareClient(&ClientConfig{
		APIKey: "abc",
		URL:    testUrl,
	})
	require.Error(t, err)
	err = c.Login()
	require.NoError(t, err)
}

func TestResolveCredentials(t *testing.T) {
	t.Parallel()

	t.Run("api key", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)
		creds, auth := resolveCredentials(&ClientConfig{APIKey: "abc"}, NewDefaultLogger(InfoLevel))
		a.True(creds.IsAPIKey())
		a.Equal("abc", creds.GetAPIKey())
		require.Len(t, auth, 1)
		apiKeyInterceptor, ok := auth[0].(*APIKeyAuthInterceptor)
		require.True(t, ok, "expected APIKeyAuthInterceptor")
		a.Equal("abc", apiKeyInterceptor.apiKey)
	})

	t.Run("user pass", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)
		creds, auth := resolveCredentials(&ClientConfig{User: "u", Password: "p", RememberMe: true}, NewDefaultLogger(InfoLevel))
		a.False(creds.IsAPIKey())
		a.Equal("u", creds.GetUser())
		a.Equal("p", creds.GetPass())
		a.True(creds.IsRememberMe())
		require.Len(t, auth, 1)
		_, ok := auth[0].(*CSRFInterceptor)
		require.True(t, ok, "expected CSRFInterceptor")
	})
}

func TestBuildInterceptorsDedup(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// A single interceptor instance supplied twice must only be added once.
	dup := NewTestInterceptor()
	config := &ClientConfig{
		Interceptors: []ClientInterceptor{dup, dup},
	}
	auth := []ClientInterceptor{&CSRFInterceptor{}}
	interceptors := buildInterceptors(config, NewDefaultLogger(InfoLevel), auth)

	count := 0
	for _, i := range interceptors {
		if i == ClientInterceptor(dup) {
			count++
		}
	}
	a.Equal(1, count, "duplicate interceptor must only be added once")
}

func TestBuildInterceptorsSetsDefaultUserAgent(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// Empty UserAgent must be defaulted on the config (mutation preserved).
	config := &ClientConfig{}
	buildInterceptors(config, NewDefaultLogger(InfoLevel), nil)
	a.Equal(defaultUserAgent, config.UserAgent)

	// Custom UserAgent must be left untouched.
	config = &ClientConfig{UserAgent: "custom-agent"}
	buildInterceptors(config, NewDefaultLogger(InfoLevel), nil)
	a.Equal("custom-agent", config.UserAgent)
}

func TestNewClientFromConfigTrimsURL(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	// Trailing slashes must be trimmed off the config URL (mutation preserved).
	config := &ClientConfig{
		URL:    testUrl + "///",
		APIKey: "test-key",
	}
	v, err := newValidator()
	require.NoError(t, err)
	c, err := newClientFromConfig(config, v)
	require.NoError(t, err)
	a.Equal(testUrl, config.URL)
	a.Equal(testUrl, c.BaseURL())
}

// TestVersionWithLockingNoDeadlock is a regression test for ARCH-01: Version()
// on a UseLocking:true client with an uncached sysInfo used to acquire c.lock and
// then re-enter it through executeRequest, self-deadlocking the goroutine. The
// fetch must now happen while holding no lock. A select + time.After makes a
// deadlock FAIL the test (timeout) rather than hang the whole suite.
func TestVersionWithLockingNoDeadlock(t *testing.T) {
	t.Parallel()

	const wantVersion = "9.9.9-test"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "", "/":
			// 200 at the root makes determineApiStyle pick the new-style API.
			w.WriteHeader(http.StatusOK)
		case "/proxy/network/api/s/default/stat/sysinfo":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"data": [{"version": "%s"}]}`, wantVersion)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// NewBareClient leaves sysInfo uncached, so Version() takes the fetch path —
	// the exact path that previously deadlocked under UseLocking.
	c, err := NewBareClient(&ClientConfig{
		URL:        ts.URL,
		APIKey:     "dummy",
		UseLocking: true,
	})
	require.NoError(t, err)

	done := make(chan string, 1)
	go func() {
		done <- c.Version()
	}()

	select {
	case got := <-done:
		assert.Equal(t, wantVersion, got, "Version() must return the controller version")
	case <-time.After(2 * time.Second):
		t.Fatal("Version() deadlocked: UseLocking:true client re-entered its own mutex (ARCH-01)")
	}
}

// TestVersionConcurrentCachedFetch is the load-bearing -race test for ARCH-01:
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

	var sysInfoHits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "", "/":
			// 200 at the root makes determineApiStyle pick the new-style API.
			w.WriteHeader(http.StatusOK)
		case "/proxy/network/api/s/default/stat/sysinfo":
			sysInfoHits.Add(1)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"data": [{"version": "%s"}]}`, wantVersion)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// NewBareClient leaves sysInfo uncached, so the first Version() fetches.
	c, err := NewBareClient(&ClientConfig{
		URL:        ts.URL,
		APIKey:     "dummy",
		UseLocking: true,
	})
	require.NoError(t, err)

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
	burstHits := sysInfoHits.Load()
	assert.GreaterOrEqual(t, burstHits, int32(1), "at least one sysInfo fetch must occur")

	assert.Equal(t, wantVersion, c.Version(), "post-burst Version() must serve the cached version")
	assert.Equal(t, burstHits, sysInfoHits.Load(), "cached Version() must not trigger another sysInfo fetch")
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
