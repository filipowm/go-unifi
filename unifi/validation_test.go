package unifi //nolint: testpackage

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vd "github.com/go-playground/validator/v10"
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

// TestValidationErrorUnwrap asserts that ValidationError exposes its underlying
// validator error via Unwrap so errors.Is/errors.As can reach it (ARCH-22).
func TestValidationErrorUnwrap(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	root := errors.New("root cause")
	ve := &ValidationError{Root: root}

	require.ErrorIs(t, ve, root, "errors.Is must reach the wrapped Root error")

	var asValErr vd.ValidationErrors
	// Validate a struct that genuinely fails so Root is a vd.ValidationErrors and
	// errors.As can extract it through Unwrap.
	v, err := newValidator()
	require.NoError(t, err)
	verr := v.Validate(&ClientConfig{})
	require.Error(t, verr)
	a.ErrorAs(verr, &asValErr, "errors.As must extract the underlying vd.ValidationErrors")
}

// TestValidationErrorDeterministicOutput asserts that Error() sorts field keys so
// the message is stable across runs (ARCH-22).
func TestValidationErrorDeterministicOutput(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	ve := &ValidationError{
		Messages: map[string]string{
			"Zeta":  "must be set",
			"Alpha": "must be set",
			"Mu":    "must be set",
		},
	}

	want := "validation failed: \nAlpha: must be set\nMu: must be set\nZeta: must be set\n"
	// Build repeatedly; a map-iteration implementation would eventually diverge.
	for range 50 {
		a.Equal(want, ve.Error())
	}
}

// TestValidateNonStructFallback asserts that Validate does not panic when the
// validator returns a non-vd.ValidationErrors error (e.g. a nil/non-struct
// argument); it must fall back to wrapping the raw error (ARCH-22).
func TestValidateNonStructFallback(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	v, err := newValidator()
	require.NoError(t, err)

	// Passing a non-struct, non-pointer value makes validator.Struct return an
	// *InvalidValidationError, which is NOT a vd.ValidationErrors.
	var verr error
	a.NotPanics(func() {
		verr = v.Validate(42)
	})
	require.Error(t, verr)

	var ve *ValidationError
	require.ErrorAs(t, verr, &ve)
	require.Error(t, ve.Root, "the raw validator error must be preserved as Root")
	a.Nil(ve.Messages, "no translated messages exist for a non-struct validation failure")
}

// TestNewValidatorExtraValidators pins the optional-extra-validators seam
// (TEST-13): a one-off CustomValidator passed to newValidator is registered on
// that instance only and must NOT leak into the shared customValidators global, so
// a freshly built plain validator does not know the throwaway tag.
func TestNewValidatorExtraValidators(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	type onlyDigits struct {
		Code string `validate:"only_digits"`
	}

	extra := NewCustomRegexValidator("only_digits", `^[0-9]+$`)

	// A validator that knows the one-off tag.
	withExtra, err := newValidator(extra)
	require.NoError(t, err)

	a.NoError(withExtra.Validate(&onlyDigits{Code: "12345"}), "matching value must pass the extra validator")
	require.ErrorContains(t, withExtra.Validate(&onlyDigits{Code: "12x45"}), "validation failed", "non-matching value must fail")

	// A plain validator built WITHOUT the extra must not know the tag — registering
	// it on a per-instance basis must not have mutated the shared customValidators
	// global. validator.Struct panics on an unknown tag, so a vanilla validator
	// validating the only_digits-tagged struct must panic, proving the tag never
	// leaked.
	plain, err := newValidator()
	require.NoError(t, err)
	a.Panics(func() {
		_ = plain.Validate(&onlyDigits{Code: "123"})
	}, "the one-off tag must not leak into a freshly built validator (no global mutation)")
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
			c, interceptor := newInterceptedClient(t, func(cfg *ClientConfig) {
				cfg.ValidationMode = tc.validationMode
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
