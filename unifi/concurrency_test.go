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

// TestConcurrentRequestsAPIKeyReplayNoRace fires N goroutines doing concurrent
// Get/Post on a single API-key client. The API key is set on every outgoing
// request (InterceptRequest); with truly concurrent requests the interceptor must
// be race-free. Run with -race.
func TestConcurrentRequestsAPIKeyReplayNoRace(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 40
		wantKey    = "test-api-key"
	)

	var (
		// authenticated counts requests that arrived carrying the API key.
		authenticated atomic.Int64
		// total counts data-endpoint requests (excludes the style-probe at "/").
		total atomic.Int64
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		total.Add(1)
		if r.Header.Get(ApiKeyHeader) == wantKey {
			authenticated.Add(1)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ok"}`))
	}))
	defer ts.Close()

	c, err := newClient(&ClientConfig{
		URL:    ts.URL,
		APIKey: wantKey,
	})
	require.NoError(t, err)

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			<-start // release together to maximize contention
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

	assert.Equal(t, int64(goroutines), total.Load(), "all data requests should reach the server")
	assert.Equal(t, int64(goroutines), authenticated.Load(), "every request must carry the API key header")
}

// TestAPIKeyAuthInterceptorConcurrentAccess directly hammers the
// APIKeyAuthInterceptor from many goroutines, isolating the header-set path
// independent of the HTTP layer. Run with -race.
func TestAPIKeyAuthInterceptorConcurrentAccess(t *testing.T) {
	t.Parallel()

	interceptor := &APIKeyAuthInterceptor{apiKey: "test-key"}
	const goroutines = 50

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			<-start
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.invalid", nil)
			assert.NoError(t, interceptor.InterceptRequest(req))
			assert.Equal(t, "test-key", req.Header.Get(ApiKeyHeader))
		}()
	}
	close(start)
	wg.Wait()
}
