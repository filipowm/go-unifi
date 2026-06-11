package unifi //nolint: testpackage

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Common test server setup for system information tests.
type sysInfoTestCase struct {
	name           string
	newAPIVersion  string
	oldAPIVersion  string
	expectedError  string
	expectedResult string
}

func setupSysInfoTestServer(tc sysInfoTestCase) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "", "/":
			w.WriteHeader(http.StatusOK)
		case "/proxy/network/api/s/default/stat/sysinfo":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"data": [{"version": "%s"}]}`, tc.newAPIVersion)
		case "/proxy/network/status":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"Meta": {"server_version": "%s"}}`, tc.oldAPIVersion)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestGetSystemInformation(t *testing.T) {
	t.Parallel()

	testCases := []sysInfoTestCase{
		{
			name:           "New API Success",
			newAPIVersion:  "v2-success",
			oldAPIVersion:  "",
			expectedResult: "v2-success",
		},
		{
			name:           "Fallback to Old API",
			newAPIVersion:  "",
			oldAPIVersion:  "old-success",
			expectedResult: "old-success",
		},
		{
			name:          "Both APIs Failure",
			newAPIVersion: "",
			oldAPIVersion: "",
			expectedError: "new API returned empty server info",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			ts := setupSysInfoTestServer(tc)
			defer ts.Close()

			// Use the TLS server's own client transport so the self-signed cert is trusted.
			srvTransport := ts.Client().Transport
			// Use SkipSystemInfo so NewClient's eager fetch does not mask the test-case
			// assertion — the test itself drives GetSystemInformation.
			c, err := NewClient(&ClientConfig{
				URL:                      ts.URL,
				APIKey:                   "dummy",
				SkipSystemInfo:           true,
				HttpRoundTripperProvider: func() http.RoundTripper { return srvTransport },
			})
			require.NoError(t, err)

			sysInfo, err := c.GetSystemInformation()

			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				a.Nil(sysInfo)
			} else {
				require.NoError(t, err)
				a.Equal(tc.expectedResult, sysInfo.Version)
			}
		})
	}
}
