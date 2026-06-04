package unifi //nolint: testpackage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthConfigurationValidation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		User, Pass, APIKey string
		shouldFail         bool
	}{
		{"", "", "", true},
		{"", "", "test", false},
		{"", "test", "", true},
		{"", "test", "test", true},
		{"test", "", "", true},
		{"test", "", "test", true},
		{"test", "test", "", false},
		{"test", "test", "test", true},
	}

	v, err := newValidator()
	require.NoError(t, err)
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("user:%s-pass:%s-apikey:%s", tc.User, tc.Pass, tc.APIKey), func(t *testing.T) {
			t.Parallel()
			// given
			cc := &ClientConfig{
				URL:      testUrl,
				User:     tc.User,
				Password: tc.Pass,
				APIKey:   tc.APIKey,
			}

			// when
			err := v.Validate(cc)
			// then
			if tc.shouldFail {
				require.ErrorContains(t, err, "validation failed")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUrlValidation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		URL         string
		shouldFail  bool
		errorString string
	}{
		{"", true, "required"},
		{"http://test.url", false, ""},
		{"http://test.url:3999", false, ""},
		{"https://test.url:3999", false, ""},
		{"ftp://test.url", true, "http"},
		{"test.url", true, "http"},
		{"http://127.0.0.1", false, ""},
		{"http://127.0.0.1:3999", false, ""},
		{"test", true, "http"},
	}

	for _, tc := range testCases {
		t.Run(tc.URL, func(t *testing.T) {
			t.Parallel()
			// given
			cc := &ClientConfig{
				URL:    tc.URL,
				APIKey: "test-key",
			}
			v, err := newValidator()
			require.NoError(t, err)

			// when
			err = v.Validate(cc)

			// then
			if tc.shouldFail {
				require.ErrorContains(t, err, "validation failed")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidationModeValidation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		validationMode ValidationMode
	}{
		{SoftValidation},
		{HardValidation},
		{DisableValidation},
		{99},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d", tc.validationMode), func(t *testing.T) {
			t.Parallel()
			// given
			cc := &ClientConfig{
				URL:            testUrl,
				APIKey:         "test-key",
				ValidationMode: tc.validationMode,
			}
			v, err := newValidator()
			require.NoError(t, err)

			// when
			err = v.Validate(cc)
			require.NoError(t, err)
		})
	}
}

type validateableBody struct {
	Data string `json:"data" validate:"required"`
}

func TestValidationModes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		validationMode ValidationMode
		expectedError  string
		expectRequest  bool
	}{
		{SoftValidation, "dial tcp", true},
		{HardValidation, "validation failed", false},
		{DisableValidation, "dial tcp", true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d", tc.validationMode), func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			// given
			interceptor := NewTestInterceptor()
			c := newNewStyleClient(&ClientConfig{
				URL:            testUrl,
				APIKey:         "test-key",
				Interceptors:   []ClientInterceptor{interceptor},
				ValidationMode: tc.validationMode,
			})
			// when
			err := c.Get(context.Background(), "", validateableBody{}, nil)

			// then
			require.ErrorContains(t, err, tc.expectedError)
			if tc.expectRequest {
				a.NotNil(interceptor.request)
			} else {
				a.Nil(interceptor.request)
			}
		})
	}
}
