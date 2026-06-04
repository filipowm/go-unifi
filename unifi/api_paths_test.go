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
		URL:       ts.URL,
		APIKey:    "test",
		VerifySSL: false,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected 200 or 302 status code")
}
