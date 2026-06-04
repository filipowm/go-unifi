package unifi //nolint: testpackage

import (
	"fmt"
	"reflect"
	"testing"
)

// expectedSettingTypes pins each registered setting key to the concrete pointer
// type its factory must produce. It locks in the contract migrated from the old
// switch statement so a future map edit can't silently change a mapping.
var expectedSettingTypes = map[string]any{
	SettingAutoSpeedtestKey:       &SettingAutoSpeedtest{},
	SettingBaresipKey:             &SettingBaresip{},
	SettingBroadcastKey:           &SettingBroadcast{},
	SettingConnectivityKey:        &SettingConnectivity{},
	SettingCountryKey:             &SettingCountry{},
	SettingDashboardKey:           &SettingDashboard{},
	SettingDohKey:                 &SettingDoh{},
	SettingDpiKey:                 &SettingDpi{},
	SettingElementAdoptKey:        &SettingElementAdopt{},
	SettingEtherLightingKey:       &SettingEtherLighting{},
	SettingEvaluationScoreKey:     &SettingEvaluationScore{},
	SettingGlobalApKey:            &SettingGlobalAp{},
	SettingGlobalNatKey:           &SettingGlobalNat{},
	SettingGlobalSwitchKey:        &SettingGlobalSwitch{},
	SettingGuestAccessKey:         &SettingGuestAccess{},
	SettingIpsKey:                 &SettingIps{},
	SettingLcmKey:                 &SettingLcm{},
	SettingLocaleKey:              &SettingLocale{},
	SettingMagicSiteToSiteVpnKey:  &SettingMagicSiteToSiteVpn{},
	SettingMgmtKey:                &SettingMgmt{},
	SettingNetflowKey:             &SettingNetflow{},
	SettingNetworkOptimizationKey: &SettingNetworkOptimization{},
	SettingNtpKey:                 &SettingNtp{},
	SettingPortaKey:               &SettingPorta{},
	SettingRadioAiKey:             &SettingRadioAi{},
	SettingRadiusKey:              &SettingRadius{},
	SettingRsyslogdKey:            &SettingRsyslogd{},
	SettingSnmpKey:                &SettingSnmp{},
	SettingSslInspectionKey:       &SettingSslInspection{},
	SettingSuperCloudaccessKey:    &SettingSuperCloudaccess{},
	SettingSuperEventsKey:         &SettingSuperEvents{},
	SettingSuperFwupdateKey:       &SettingSuperFwupdate{},
	SettingSuperIdentityKey:       &SettingSuperIdentity{},
	SettingSuperMailKey:           &SettingSuperMail{},
	SettingSuperMgmtKey:           &SettingSuperMgmt{},
	SettingSuperSdnKey:            &SettingSuperSdn{},
	SettingSuperSmtpKey:           &SettingSuperSmtp{},
	SettingTeleportKey:            &SettingTeleport{},
	SettingUsgKey:                 &SettingUsg{},
	SettingUswKey:                 &SettingUsw{},
}

// TestSettingNewFieldsRegistryCoverage ensures the factory registry and the
// expected-type pin stay in lockstep: no extra, no missing keys.
func TestSettingNewFieldsRegistryCoverage(t *testing.T) {
	if len(settingFactories) != len(expectedSettingTypes) {
		t.Fatalf("registry/expectation size mismatch: settingFactories=%d expected=%d",
			len(settingFactories), len(expectedSettingTypes))
	}
	for key := range settingFactories {
		if _, ok := expectedSettingTypes[key]; !ok {
			t.Errorf("registered key %q has no expected type pinned in the test", key)
		}
	}
	for key := range expectedSettingTypes {
		if _, ok := settingFactories[key]; !ok {
			t.Errorf("expected key %q is missing from settingFactories", key)
		}
	}
}

// TestSettingNewFieldsConcreteTypes verifies that for every registered key,
// newFields returns a non-nil pointer of the correct concrete type and no error.
func TestSettingNewFieldsConcreteTypes(t *testing.T) {
	for key, want := range expectedSettingTypes {
		t.Run(key, func(t *testing.T) {
			s := &Setting{Key: key}
			got, err := s.newFields()
			if err != nil {
				t.Fatalf("newFields() returned error for key %q: %v", key, err)
			}
			if got == nil {
				t.Fatalf("newFields() returned nil for key %q", key)
			}
			wantType := reflect.TypeOf(want)
			gotType := reflect.TypeOf(got)
			if gotType != wantType {
				t.Fatalf("newFields() for key %q returned %s, want %s", key, gotType, wantType)
			}
			// Returned value must be a non-nil pointer.
			if v := reflect.ValueOf(got); v.Kind() != reflect.Pointer || v.IsNil() {
				t.Fatalf("newFields() for key %q did not return a non-nil pointer: %#v", key, got)
			}
		})
	}
}

// TestSettingNewFieldsFreshInstances locks in the critical invariant that each
// call yields a DISTINCT pointer. A registry of shared instances would alias
// decoded JSON state across callers — this guards against that regression.
func TestSettingNewFieldsFreshInstances(t *testing.T) {
	for key := range settingFactories {
		t.Run(key, func(t *testing.T) {
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
