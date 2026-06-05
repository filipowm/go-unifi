package unifi //nolint: testpackage

import (
	"fmt"
	"reflect"
	"testing"
)

// driftSentinelKeys are the three setting keys whose factories were missing from
// the old hand-maintained settingFactories literal (the W0 fix). They are the
// concrete drift the registry-generation change exists to prevent: if codegen
// ever stops self-registering one of these, the registry test below must fail
// the build rather than silently shipping six broken Get/Update methods again.
//
// These are pinned by their generated *Key constants, so renaming a constant in
// codegen without updating this list is itself caught at compile time.
var driftSentinelKeys = []string{
	SettingMdnsKey,
	SettingRoamingAssistantKey,
	SettingTrafficFlowKey,
}

// TestSettingFactoriesSelfRegistered asserts the registry is populated purely
// from the generated per-setting init() functions. The truth is the generated
// catalog itself — there is no hand-maintained mirror to drift from. A new
// codegen setting that fails to self-register would simply be absent here, and
// its typed Get/Update methods would route through newFields -> "unexpected key";
// the per-key checks below guarantee every registered key is well-formed.
func TestSettingFactoriesSelfRegistered(t *testing.T) {
	t.Parallel()

	if len(settingFactories) == 0 {
		t.Fatal("settingFactories is empty: no setting self-registered via init(); " +
			"the generated registry is broken")
	}
}

// TestSettingDriftSentinelKeysRegistered locks in the exact regression the W0
// patch fixed. If any of the three previously-dropped keys is missing from the
// registry, this fails the BUILD — the codegen self-registration must keep them
// present. This is the guard that makes the old drift impossible to recur
// silently.
func TestSettingDriftSentinelKeysRegistered(t *testing.T) {
	t.Parallel()

	for _, key := range driftSentinelKeys {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			if _, ok := settingFactories[key]; !ok {
				t.Fatalf("setting key %q is not registered: the codegen self-registration "+
					"for this setting regressed (this is the exact W0 drift bug)", key)
			}
		})
	}
}

// TestSettingNewFieldsRegistryConstructs derives its truth from the generated
// registry: for every registered key, newFields must return a non-nil pointer of
// a concrete, key-specific type with no error. This replaces the old
// hand-mirrored expectedSettingTypes literal (which compared two hand-maintained
// mirrors and so could never catch codegen drift).
func TestSettingNewFieldsRegistryConstructs(t *testing.T) {
	t.Parallel()

	// seenTypes guards that distinct keys map to distinct concrete types, so two
	// keys can't accidentally share a factory.
	seenTypes := make(map[reflect.Type]string, len(settingFactories))

	for key := range settingFactories {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			s := &Setting{Key: key}
			got, err := s.newFields()
			if err != nil {
				t.Fatalf("newFields() returned error for registered key %q: %v", key, err)
			}
			if got == nil {
				t.Fatalf("newFields() returned nil for registered key %q", key)
			}
			v := reflect.ValueOf(got)
			if v.Kind() != reflect.Pointer || v.IsNil() {
				t.Fatalf("newFields() for key %q did not return a non-nil pointer: %#v", key, got)
			}
		})
	}

	// Distinct-type check runs independently of the parallel subtests; rebuild
	// its own values so it does not depend on subtest ordering.
	for key := range settingFactories {
		got, err := (&Setting{Key: key}).newFields()
		if err != nil {
			t.Fatalf("newFields() errored for key %q: %v", key, err)
		}
		typ := reflect.TypeOf(got)
		if other, dup := seenTypes[typ]; dup {
			t.Fatalf("keys %q and %q both construct %s; each setting key must map to a distinct concrete type", key, other, typ)
		}
		seenTypes[typ] = key
	}
}

// TestSettingNewFieldsFreshInstances locks in the critical invariant that each
// call yields a DISTINCT pointer. A registry of shared instances would alias
// decoded JSON state across callers — this guards against that regression.
func TestSettingNewFieldsFreshInstances(t *testing.T) {
	t.Parallel()

	for key := range settingFactories {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			s := &Setting{Key: key}
			a, err := s.newFields()
			if err != nil {
				t.Fatalf("first newFields() for key %q errored: %v", key, err)
			}
			b, err := s.newFields()
			if err != nil {
				t.Fatalf("second newFields() for key %q errored: %v", key, err)
			}
			if reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() {
				t.Fatalf("newFields() for key %q returned the same pointer on two calls; instances must be fresh", key)
			}
		})
	}
}

// TestSettingNewFieldsUnknownKey verifies the unknown-key error is preserved
// exactly as the original switch produced it.
func TestSettingNewFieldsUnknownKey(t *testing.T) {
	t.Parallel()

	const bogus = "definitely-not-a-real-setting-key"
	s := &Setting{Key: bogus}
	got, err := s.newFields()
	if got != nil {
		t.Fatalf("expected nil fields for unknown key, got %#v", got)
	}
	if err == nil {
		t.Fatal("expected an error for unknown key, got nil")
	}
	want := fmt.Sprintf("unexpected key %q", bogus)
	if err.Error() != want {
		t.Fatalf("error mismatch:\n got: %q\nwant: %q", err.Error(), want)
	}
}

// TestRegisterSettingDuplicatePanics asserts the registry rejects a duplicate
// key registration loudly, so a generated key collision fails at init() rather
// than silently shadowing an earlier factory.
func TestRegisterSettingDuplicatePanics(t *testing.T) {
	t.Parallel()

	// Use a key that is guaranteed to already be registered.
	const key = SettingMdnsKey
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("registerSetting did not panic on a duplicate key")
		}
	}()
	registerSetting(key, func() any { return &SettingMdns{} })
}
