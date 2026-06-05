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
// TLS verification emits a warning (ARCH-06). It embeds the noop logger so only
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

// TestBuildHTTPClientTLSDefaults is the ARCH-06 secure-by-default contract: a
// nil VerifySSL (the zero value) MUST verify certificates; only an explicit
// pointer-to-false disables verification, and that case MUST emit a Warn log.
func TestBuildHTTPClientTLSDefaults(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		verifySSL       *bool
		wantSkipVerify  bool
		wantWarnEmitted bool
	}{
		"nil (zero value) -> verification ON, no warn": {
			verifySSL:       nil,
			wantSkipVerify:  false,
			wantWarnEmitted: false,
		},
		"explicit true -> verification ON, no warn": {
			verifySSL:       new(true),
			wantSkipVerify:  false,
			wantWarnEmitted: false,
		},
		"explicit false -> verification OFF, warn emitted": {
			verifySSL:       new(false),
			wantSkipVerify:  true,
			wantWarnEmitted: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			log := &capturingLogger{}
			httpClient, err := buildHTTPClient(&ClientConfig{
				URL:       testUrl,
				APIKey:    "test-key",
				VerifySSL: tc.verifySSL,
			}, log)
			require.NoError(t, err)

			assert.Equal(t, tc.wantSkipVerify, transportInsecureSkipVerify(t, httpClient),
				"InsecureSkipVerify must match the secure-by-default contract")

			if tc.wantWarnEmitted {
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
	assert.False(t, tlsVerificationDisabled(&ClientConfig{VerifySSL: nil}), "nil verifies")
	assert.False(t, tlsVerificationDisabled(&ClientConfig{VerifySSL: new(true)}), "true verifies")
	assert.True(t, tlsVerificationDisabled(&ClientConfig{VerifySSL: new(false)}), "false disables")
}

// TestBuildHTTPClientCustomRoundTripperBypassesTLS ensures that a custom
// round-tripper provider takes precedence and the default TLS transport (and
// its warning) is not constructed.
func TestBuildHTTPClientCustomRoundTripperBypassesTLS(t *testing.T) {
	t.Parallel()
	log := &capturingLogger{}
	httpClient, err := buildHTTPClient(&ClientConfig{
		URL:       testUrl,
		APIKey:    "test-key",
		VerifySSL: new(false), // would normally warn, but the custom RT wins
		HttpRoundTripperProvider: func() http.RoundTripper {
			return &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13}}
		},
	}, log)
	require.NoError(t, err)
	// The provided transport is used verbatim; no insecure default applied.
	assert.False(t, transportInsecureSkipVerify(t, httpClient))
	assert.Empty(t, log.warns, "custom round tripper must not trigger the insecure-default warning")
}

func containsAny(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}
