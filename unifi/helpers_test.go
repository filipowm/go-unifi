package unifi //nolint: testpackage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
)

const (
	localUrl = "http://127.0.0.1:64431"
	testUrl  = "http://test.url"
)

type TestData struct {
	Data string `json:"data"`
}

// newNewStyleClient builds a bare client from cfg and pins it to the new-style
// API paths. The construction error is intentionally swallowed: callers that
// exercise request behavior rely on the later request failing, not on the
// connection-time error. Tests asserting on the construction error must call
// newBareClient directly instead.
func newNewStyleClient(cfg *ClientConfig) *client {
	c, _ := newBareClient(cfg)
	c.apiPaths = &NewStyleAPI
	return c
}

func runTestServer(path string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always set the CSRF header on the response.
		w.Header().Set(CsrfHeader, "csrf-token")
		if !strings.EqualFold(r.URL.Path, path) {
			http.NotFound(w, r)
			return
		}

		// Return a JSON response
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TestData{Data: "test"})
	}))
}
