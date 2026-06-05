package unifi //nolint: testpackage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentRequestsCSRFReplayNoRace is the ARCH-04 concurrency regression
// test. It fires N goroutines doing concurrent Get/Post on a single user/pass
// client. The CSRF token is read on every outgoing request (InterceptRequest)
// and written from every response (InterceptResponse); with the coarse
// per-request lock removed (O4) these now run truly concurrently, so the token
// must be guarded inside CSRFInterceptor. Run with -race for the data race on
// the token to bite.
//
// It also asserts correct CSRF replay: once the server has handed out a token,
// every subsequent request carries it back in the X-Csrf-Token header.
func TestConcurrentRequestsCSRFReplayNoRace(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 40
		csrfToken  = "concurrent-csrf-token" //nolint:gosec // not a credential; a CSRF test token
	)

	var (
		// replays counts requests that arrived carrying the CSRF token back.
		replays atomic.Int64
		// total counts data-endpoint requests (excludes the style-probe at "/").
		total atomic.Int64
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always hand out the CSRF token on the response so the interceptor
		// captures it; the controller does this on every reply.
		w.Header().Set(CsrfHeader, csrfToken)

		if r.URL.Path == "/" {
			// 200 at the root selects the new-style API during probing.
			w.WriteHeader(http.StatusOK)
			return
		}

		total.Add(1)
		if r.Header.Get(CsrfHeader) == csrfToken {
			replays.Add(1)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ok"}`))
	}))
	defer ts.Close()

	c, err := NewBareClient(&ClientConfig{
		URL:      ts.URL,
		User:     "test-user",
		Password: "test-pass",
	})
	require.NoError(t, err)

	// Prime the token with one request so the replay assertion is meaningful:
	// after this, the interceptor has captured the CSRF token.
	require.NoError(t, c.Get(context.Background(), "prime", nil, nil))

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			<-start // release together to maximize contention on the token
			var err error
			if idx%2 == 0 {
				err = c.Get(context.Background(), "resource", nil, nil)
			} else {
				err = c.Post(context.Background(), "resource", map[string]string{"k": "v"}, nil)
			}
			assert.NoError(t, err)
		}(i)
	}
	close(start)
	wg.Wait()

	// The prime request (the first one) ran before any token was captured, so it
	// does NOT replay; every one of the goroutines+1 requests reaches the server,
	// and all goroutines fired after priming must replay the captured token.
	assert.Equal(t, int64(goroutines+1), total.Load(), "all data requests should reach the server")
	assert.Equal(t, int64(goroutines), replays.Load(), "every post-prime request must replay the captured CSRF token")
}

// TestCSRFInterceptorConcurrentAccess directly hammers the CSRFInterceptor's
// read/write pair from many goroutines, isolating the token-guard data race
// independent of the HTTP layer. Run with -race.
func TestCSRFInterceptorConcurrentAccess(t *testing.T) {
	t.Parallel()

	interceptor := &CSRFInterceptor{}
	const goroutines = 50

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		// Writers: store a token via InterceptResponse.
		go func() {
			defer wg.Done()
			<-start
			resp := &http.Response{Header: http.Header{}}
			resp.Header.Set(CsrfHeader, "tok")
			assert.NoError(t, interceptor.InterceptResponse(resp))
		}()
		// Readers: read the token via InterceptRequest.
		go func() {
			defer wg.Done()
			<-start
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.invalid", nil)
			assert.NoError(t, interceptor.InterceptRequest(req))
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, "tok", interceptor.CSRFToken(), "token must be set after concurrent writes")
}
