package unifi

// settingFactories maps a setting key to a constructor for its concrete fields
// type. It is populated exclusively by registerSetting calls emitted from the
// generated setting_*.generated.go files (one init() per setting), so the
// registry is a 1:1 reflection of the generated setting catalog and cannot drift
// from it by hand.
//
// Each registered factory MUST return a fresh pointer: the returned value is
// passed to json.Unmarshal, so sharing a single instance across calls would
// alias decoded state between callers.
var settingFactories = map[string]func() any{}

// registerSetting wires a generated setting key to its fields constructor. It is
// called from the generated per-setting init() functions; hand-written code must
// not call it directly. A duplicate key registration panics at init time so an
// accidental key collision in the generated catalog fails the build loudly
// instead of silently shadowing an earlier factory.
func registerSetting(key string, f func() any) {
	if _, exists := settingFactories[key]; exists {
		panic("unifi: duplicate setting key registration: " + key)
	}
	settingFactories[key] = f
}
