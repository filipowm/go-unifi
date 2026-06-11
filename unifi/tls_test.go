package unifi //nolint: testpackage

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingLogger records every Warn message so tests can assert that disabling
// TLS verification emits a warning. It embeds the noop logger so only
// the methods under test need overriding.
type capturingLogger struct {
	noopLogger

	warns []string
}

func (l *capturingLogger) Warn(msg string) {
	l.warns = append(l.warns, msg)
}

func (l *capturingLogger) Warnf(format string, args ...any) {
	l.warns = append(l.warns, format)
}

// transportInsecureSkipVerify extracts the effective InsecureSkipVerify flag
// from the built client's transport.
func transportInsecureSkipVerify(t *testing.T, c *http.Client) bool {
	t.Helper()
	tr, ok := c.Transport.(*http.Transport)
	require.True(t, ok, "expected *http.Transport")
	require.NotNil(t, tr.TLSClientConfig, "expected a TLSClientConfig")
	return tr.TLSClientConfig.InsecureSkipVerify
}

// TestBuildHTTPClientTLSDefaults is the secure-by-default contract: the
// zero-value SkipVerifySSL (false) MUST verify certificates; only an explicit
// SkipVerifySSL=true disables verification, and that case MUST emit a Warn log.
func TestBuildHTTPClientTLSDefaults(t *testing.T) {
	t.Parallel()

	// SkipVerifySSL maps to the transport's InsecureSkipVerify by identity, and a
	// disabled verification (true) is the only case that warns — so each input row
	// fully determines both expectations.
	cases := map[string]struct {
		skipVerifySSL bool
	}{
		"zero value (false) -> verification ON, no warn": {skipVerifySSL: false},
		"true -> verification OFF, warn emitted":         {skipVerifySSL: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			log := &capturingLogger{}
			httpClient, err := buildHTTPClient(&ClientConfig{
				URL:           testUrl,
				APIKey:        "test-key",
				SkipVerifySSL: tc.skipVerifySSL,
			}, log)
			require.NoError(t, err)

			assert.Equal(t, tc.skipVerifySSL, transportInsecureSkipVerify(t, httpClient),
				"SkipVerifySSL must pass through to the transport's InsecureSkipVerify")

			if tc.skipVerifySSL {
				require.NotEmpty(t, log.warns, "disabling verification must emit a Warn log")
				assert.Truef(t, containsAny(log.warns, "verification is DISABLED"),
					"warn must mention disabled verification, got: %v", log.warns)
			} else {
				assert.Empty(t, log.warns, "verification ON must not warn")
			}
		})
	}
}

// TestTLSVerificationDisabled directly pins the secure-by-default predicate.
func TestTLSVerificationDisabled(t *testing.T) {
	t.Parallel()
	assert.False(t, tlsVerificationDisabled(&ClientConfig{}), "zero value verifies")
	assert.False(t, tlsVerificationDisabled(&ClientConfig{SkipVerifySSL: false}), "false verifies")
	assert.True(t, tlsVerificationDisabled(&ClientConfig{SkipVerifySSL: true}), "true disables")
}

// TestBuildHTTPClientCustomRoundTripperBypassesTLS ensures that a custom
// round-tripper provider takes precedence and the default TLS transport (and
// its SkipVerifySSL warning) is not constructed. The provider itself emits its
// own warning reminding the caller that TLS is entirely caller-managed.
func TestBuildHTTPClientCustomRoundTripperBypassesTLS(t *testing.T) {
	t.Parallel()
	log := &capturingLogger{}
	httpClient, err := buildHTTPClient(&ClientConfig{
		URL:           testUrl,
		APIKey:        "test-key",
		SkipVerifySSL: true, // would normally warn, but the custom RT wins
		HttpRoundTripperProvider: func() http.RoundTripper {
			return &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13}}
		},
	}, log)
	require.NoError(t, err)
	// The provided transport is used verbatim; no insecure default applied.
	assert.False(t, transportInsecureSkipVerify(t, httpClient))
	// A warn IS emitted — but it is the "caller-managed TLS" notice, NOT the
	// "InsecureSkipVerify disabled" warn that SkipVerifySSL would produce.
	require.NotEmpty(t, log.warns, "HttpRoundTripperProvider must emit a TLS-ownership warning")
	assert.True(t, containsAny(log.warns, "caller-managed"), "warn must mention caller-managed TLS")
	assert.False(t, containsAny(log.warns, "verification is DISABLED"), "insecure-default warn must NOT appear when custom RT is used")
}

func containsAny(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}
