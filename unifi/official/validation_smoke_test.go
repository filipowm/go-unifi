package official //nolint:testpackage

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidationTagsGoPlaygroundParseable feeds representative generated structs
// to go-playground's validator, asserting that emitted tags are parseable and
// type-compatible at runtime. This guards the quirk fields (float-string-enum,
// int-string-enum) that could produce panic-inducing tags if the codegen ever
// emits a type-incompatible bound. Validates on both a valid and an out-of-range
// value to confirm the rule fires when expected.
func TestValidationTagsGoPlaygroundParseable(t *testing.T) {
	t.Parallel()
	v := validator.New()

	// Float-typed enum: FrequencyGHz is float32. go-playground's oneof panics on
	// float types, so the codegen emits no oneof for "number" fields — the field
	// carries no validate tag. Both valid and out-of-range values must pass
	// validate.Struct without panic or InvalidValidationError.
	t.Run("float_enum_valid", func(t *testing.T) {
		t.Parallel()
		s := LatestStatisticsForWirelessRadio{FrequencyGHz: LatestStatisticsForWirelessRadioFrequencyGHzN24}
		assertTagParseable(t, v, s)
		require.NoError(t, v.Struct(s))
	})
	t.Run("float_enum_out_of_range_no_panic", func(t *testing.T) {
		t.Parallel()
		// No oneof tag emitted for float fields; any float value must pass without panic.
		s := LatestStatisticsForWirelessRadio{FrequencyGHz: 99.9}
		assertTagParseable(t, v, s)
		require.NoError(t, v.Struct(s), "no validate tag on float enum field; any value passes")
	})

	// Int-string-enum quirk: WifiBasicDataRateConfiguration24 is int32 with
	// string-formatted oneof values (1000 2000 5500 …).
	t.Run("int_enum_valid", func(t *testing.T) {
		t.Parallel()
		s := WifiBasicDataRateConfiguration{
			N24: WifiBasicDataRateConfiguration24N1000,
			N5:  WifiBasicDataRateConfiguration5N6000,
		}
		assertTagParseable(t, v, s)
		require.NoError(t, v.Struct(s))
	})
	t.Run("int_enum_invalid", func(t *testing.T) {
		t.Parallel()
		s := WifiBasicDataRateConfiguration{N24: 9999}
		assertTagParseable(t, v, s)
		err := v.Struct(s)
		require.Error(t, err, "out-of-enum value must fail oneof")
		assertValidationErrors(t, err)
	})

	// Numeric range: gte/lte bounds must parse as integers without error.
	t.Run("numeric_range_valid", func(t *testing.T) {
		t.Parallel()
		s := DHCPConfigurationForIPv6Network{LeaseTimeSeconds: 3600}
		assertTagParseable(t, v, s)
		require.NoError(t, v.Struct(s))
	})
	t.Run("numeric_range_invalid", func(t *testing.T) {
		t.Parallel()
		s := DHCPConfigurationForIPv6Network{LeaseTimeSeconds: -1}
		assertTagParseable(t, v, s)
		err := v.Struct(s)
		require.Error(t, err, "negative value must fail gte=0")
		assertValidationErrors(t, err)
	})

	// Array-dive: each slice element validated against oneof via dive.
	t.Run("array_dive_valid", func(t *testing.T) {
		t.Parallel()
		s := AdoptedDeviceOverview{
			Features: []AdoptedDeviceOverviewFeatures{AdoptedDeviceOverviewFeaturesSwitching},
		}
		assertTagParseable(t, v, s)
		require.NoError(t, v.Struct(s))
	})
	t.Run("array_dive_invalid", func(t *testing.T) {
		t.Parallel()
		s := AdoptedDeviceOverview{
			Features: []AdoptedDeviceOverviewFeatures{"bogus"},
		}
		assertTagParseable(t, v, s)
		err := v.Struct(s)
		require.Error(t, err, "unknown enum value must fail dive+oneof")
		assertValidationErrors(t, err)
	})
}

// assertTagParseable asserts validate.Struct does not panic and does not return
// an InvalidValidationError (a tag-parse failure). Whether s is valid or not is
// irrelevant — this only checks the tag is structurally acceptable to go-playground.
func assertTagParseable(t *testing.T, v *validator.Validate, s any) {
	t.Helper()
	require.NotPanics(t, func() {
		err := v.Struct(s)
		var inv *validator.InvalidValidationError
		if errors.As(err, &inv) {
			t.Errorf("validate.Struct returned InvalidValidationError (tag-parse failure): %v", inv)
		}
	})
}

// assertValidationErrors asserts err is a ValidationErrors (normal rule
// violation), not an InvalidValidationError (tag-parse / structural failure).
func assertValidationErrors(t *testing.T, err error) {
	t.Helper()
	var verr validator.ValidationErrors
	assert.ErrorAs(t, err, &verr, "expected ValidationErrors, got %T: %v", err, err)
}
