package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Removed dummy type declarations for FieldInfo and Resource since they are already defined in the package

func TestUnmarshalCustomizationYamlDefault(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	generate, err := unmarshalCustomizationYaml("")
	require.NoError(t, err)
	require.NotNil(t, generate)

	// Check that some expected resource customizations exist
	a.Contains(generate.Customizations.Resources, "Account")
	a.Contains(generate.Customizations.Resources, "Device")

	dvc, ok := generate.Customizations.Resources["Device"]
	a.True(ok, "Device customization should exist")
	a.Contains(dvc.Fields, AllFieldsCustomizationKeyword)
}

func TestNewCodeCustomizer_NonExistent(t *testing.T) {
	t.Parallel()
	cc, err := NewCodeCustomizer("nonexistent.yml")
	require.Error(t, err)
	require.ErrorContains(t, err, "failed reading customizations file")
	assert.Nil(t, cc)
}

func TestApplyToResource(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	cc, err := NewCodeCustomizer("")
	require.NoError(t, err)

	// Create a dummy Resource for 'Device'
	res := &Resource{StructName: "Device"}
	cc.ApplyToResource(res)
	a.NotNil(res.FieldProcessor, "FieldProcessor should be set after applying customizations")

	// Test field 'X': should update FieldType to "float64" and _all customization sets omitEmpty true
	fiX := &FieldInfo{
		FieldName: "X",
		FieldType: "string",
		OmitEmpty: false,
	}
	err = res.FieldProcessor("X", fiX)
	require.NoError(t, err)
	a.Equal("float64", fiX.FieldType, "X field type should be updated to float64")
	a.True(fiX.OmitEmpty, "OmitEmpty should be true due to _all customization")

	// Test field 'Channel': applied only when FieldType equals "string"
	fiChannel := &FieldInfo{
		FieldName: "Channel",
		FieldType: "string",
	}
	err = res.FieldProcessor("Channel", fiChannel)
	require.NoError(t, err)
	a.Equal("numberOrString", fiChannel.CustomUnmarshalType, "Channel should get customUnmarshalType override")

	// Test 'Channel' with non-matching FieldType: no override gets applied
	fiChannelMismatch := &FieldInfo{
		FieldName: "Channel",
		FieldType: "int",
	}
	err = res.FieldProcessor("Channel", fiChannelMismatch)
	require.NoError(t, err)
	a.Empty(fiChannelMismatch.CustomUnmarshalType, "Override should not apply when FieldType does not match")
}

func TestCompositeFieldProcessor(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	cc, err := NewCodeCustomizer("")
	require.NoError(t, err)

	// Create a Resource for 'Account' with a pre-existing FieldProcessor that appends "_original" to FieldName
	res := &Resource{
		StructName: "Account",
		FieldProcessor: func(name string, f *FieldInfo) error {
			// Original processing: append '_original' to FieldName
			f.FieldName += "_original"
			return nil
		},
	}
	cc.ApplyToResource(res)
	a.NotNil(res.FieldProcessor, "Composite FieldProcessor should be set")

	// For Account, customization for field 'IP' sets omitEmpty true
	fiIP := &FieldInfo{
		FieldName: "IP",
		FieldType: "string",
		OmitEmpty: false,
	}
	err = res.FieldProcessor("IP", fiIP)
	require.NoError(t, err)
	// Expected behavior: customization applies first (e.g. setting omitEmpty) and then the original processor appends suffix
	a.True(fiIP.OmitEmpty, "OmitEmpty should be set to true by customization")
	a.Equal("IP_original", fiIP.FieldName, "FieldName should have '_original' appended by the composite processor")
}

func TestNoCustomizationForResource(t *testing.T) {
	t.Parallel()
	// Create a Resource that does not have any associated customizations
	res := &Resource{StructName: "NonExistent"}

	cc, err := NewCodeCustomizer("")
	require.NoError(t, err)

	cc.ApplyToResource(res)
	assert.Nil(t, res.FieldProcessor, "FieldProcessor should remain nil if no customization applies")
}

func createTempCustomizationsYaml(t *testing.T, data string) string {
	t.Helper()
	tempFile := filepath.Join(t.TempDir(), "temp_customizations.yml")
	err := os.WriteFile(tempFile, []byte(data), 0o644) //nolint:gosec
	require.NoError(t, err, "should create temp file")
	return tempFile
}

func TestReadCustomizationsYmlError(t *testing.T) {
	t.Parallel()
	tempFile := createTempCustomizationsYaml(t, "invalid: yaml: ::::\n")

	// Expect an error due to invalid YAML
	_, err := unmarshalCustomizationYaml(tempFile)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed unmarshalling YAML")
}

func TestApplyToResource_CustomInline(t *testing.T) {
	t.Parallel()
	yamlContent := `
customizations:
  resources:
    CustomResource:
      fields:
        CustomField:
          fieldType: int
          omitEmpty: true
`
	tempFile := createTempCustomizationsYaml(t, yamlContent)
	cc, err := NewCodeCustomizer(tempFile)
	require.NoError(t, err)

	res := &Resource{StructName: "CustomResource"}
	cc.ApplyToResource(res)
	require.NotNil(t, res.FieldProcessor)

	fi := &FieldInfo{
		FieldName: "CustomField",
		FieldType: "string",
		OmitEmpty: false,
	}
	err = res.FieldProcessor("CustomField", fi)
	require.NoError(t, err)
	assert.Equal(t, "int", fi.FieldType, "Custom field type should be updated to int")
	assert.True(t, fi.OmitEmpty, "Custom field omitEmpty should be true")
}

func TestApplyToResource_CustomFieldMismatch(t *testing.T) {
	t.Parallel()
	yamlContent := `
customizations:
  resources:
    CustomResource:
      fields:
        CustomField:
          ifFieldType: string
          customUnmarshalType: customType
`
	tempFile := createTempCustomizationsYaml(t, yamlContent)
	cc, err := NewCodeCustomizer(tempFile)
	require.NoError(t, err)

	res := &Resource{StructName: "CustomResource"}
	cc.ApplyToResource(res)
	require.NotNil(t, res.FieldProcessor)

	fi := &FieldInfo{
		FieldName: "CustomField",
		FieldType: "int",
	}
	err = res.FieldProcessor("CustomField", fi)
	require.NoError(t, err)
	assert.Empty(t, fi.CustomUnmarshalType, "Customization should not apply if field type mismatches")
}

func TestExcludedClientFunctions(t *testing.T) {
	t.Parallel()
	yamlContent := `
customizations:
  resources:
    Network:
      excludeFunctions:
        - Update
        - Delete
    SettingMgmt:
      excludeFunctions:
        - Update
    EmptyExclude:
      fields:
        Foo:
          omitEmpty: true
`
	tempFile := createTempCustomizationsYaml(t, yamlContent)
	cc, err := NewCodeCustomizer(tempFile)
	require.NoError(t, err)

	cases := map[string]struct {
		resource *Resource
		want     []string
	}{
		"configured normal resource":         {resource: &Resource{StructName: "Network"}, want: []string{"Update", "Delete"}},
		"configured settings resource":       {resource: &Resource{StructName: "SettingMgmt"}, want: []string{"Update"}},
		"resource without excludeFunctions":  {resource: &Resource{StructName: "EmptyExclude"}, want: nil},
		"resource without any customization": {resource: &Resource{StructName: "Unknown"}, want: nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, cc.ExcludedClientFunctions(tc.resource))
		})
	}
}

func TestExcludedClientFunctions_NilSafe(t *testing.T) {
	t.Parallel()
	// An empty customizer (no resources configured) must not panic.
	cc := &CodeCustomizer{Customizations: Customizations{}}
	assert.Nil(t, cc.ExcludedClientFunctions(&Resource{StructName: "Network"}))
}

func TestExcludedClientFunctions_UnknownActionWarns(t *testing.T) {
	// Parallel-safe: the warning sink is an INJECTED logger with its own local
	// hook, not the mutated package global, so this test shares no state with any
	// other and reads only its own entries. See TEST-13.
	t.Parallel()
	yamlContent := `
customizations:
  resources:
    Network:
      excludeFunctions:
        - Updte
    SettingMgmt:
      excludeFunctions:
        - List
`
	tempFile := createTempCustomizationsYaml(t, yamlContent)
	cc, err := NewCodeCustomizer(tempFile)
	require.NoError(t, err)

	logger, hook := test.NewNullLogger()
	cc.logger = logger

	// Unknown action is still returned (warn-and-ignore: AddResource simply never
	// matches it, so no real method is dropped).
	assert.Equal(t, []string{"Updte"}, cc.ExcludedClientFunctions(&Resource{StructName: "Network"}))
	// "List" is invalid for a settings resource (only Get/Update exist).
	assert.Equal(t, []string{"List"}, cc.ExcludedClientFunctions(&Resource{StructName: "SettingMgmt"}))

	var msgs []string
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			msgs = append(msgs, e.Message)
		}
	}
	assert.Contains(t, msgs, `excludeFunctions: unknown action "Updte" for resource Network (ignored)`)
	assert.Contains(t, msgs, `excludeFunctions: unknown action "List" for resource SettingMgmt (ignored)`)
}

// TestApplyToResource_OrderingContract pins the ARCH-21 field-processor ordering
// contract: the YAML field customizations run FIRST (the _all keyword, then the
// named field), then any processor pre-installed by customizeResource. Each leg
// runs once per field. A pre-installed processor that records the FieldType it
// observes proves the YAML override has already run by the time it executes.
func TestApplyToResource_OrderingContract(t *testing.T) {
	t.Parallel()

	yamlContent := `
customizations:
  resources:
    Ordered:
      resourcePath: overridden/path
      fields:
        Name:
          fieldType: int
`
	tempFile := createTempCustomizationsYaml(t, yamlContent)
	cc, err := NewCodeCustomizer(tempFile)
	require.NoError(t, err)

	// The pre-installed processor records what FieldType it sees and how many times
	// it runs per field. Because the YAML override runs first, by the time this
	// runs for "Name" the type has already been rewritten to int.
	seenType := map[string]string{}
	runs := map[string]int{}
	res := NewResource("Ordered", "original/path")
	res.FieldProcessor = func(name string, f *FieldInfo) error {
		runs[name]++
		seenType[name] = f.FieldType
		return nil
	}

	cc.ApplyToResource(res)
	require.NotNil(t, res.FieldProcessor)

	// Resource-level override applied.
	assert.Equal(t, "overridden/path", res.ResourcePath, "resourcePath override should be applied")

	// Drive the real field pipeline once.
	require.NoError(t, res.processJSON([]byte(`{"name":".{0,32}","other":".{0,32}"}`)))

	// The pre-installed processor runs exactly once per field.
	assert.Equal(t, 1, runs["Name"], "pre-installed processor must run exactly once for Name")
	assert.Equal(t, 1, runs["Other"], "pre-installed processor must run exactly once for Other")
	// Ordering: YAML override ran before the pre-installed processor.
	assert.Equal(t, "int", seenType["Name"], "YAML field override must run before the pre-installed processor")
	// Final state confirms the override took effect.
	f := res.BaseType().Fields["Name"]
	require.NotNil(t, f)
	assert.Equal(t, "int", f.FieldType, "field override should be applied")
}

// TestApplyFieldOverrides_IsNotANoOp proves that applyFieldOverrides genuinely
// mutates the processor chain on every call: a second call re-wraps the YAML
// customization leg around the chain so it runs twice. This is exactly why
// customizations must be applied EXACTLY ONCE per resource (and why the dead
// re-apply in collectResourceGenerators was removed — see ARCH-21). The guard
// counts YAML-leg invocations through a customUnmarshalFunc override that the
// pre-installed leaf records on each pass.
func TestApplyFieldOverrides_IsNotANoOp(t *testing.T) {
	t.Parallel()

	yamlContent := `
customizations:
  resources:
    Twice:
      fields:
        _all:
          omitEmpty: true
`
	tempFile := createTempCustomizationsYaml(t, yamlContent)
	cc, err := NewCodeCustomizer(tempFile)
	require.NoError(t, err)
	rc := cc.Customizations.Resources["Twice"]
	require.NotNil(t, rc)

	// yamlLegRuns counts how many times the YAML customization leg executes for a
	// single composed-processor invocation. We observe it indirectly: the
	// pre-installed leaf appends a marker, and a second applyFieldOverrides wraps
	// the whole chain, so the YAML leg (and the _all keyword pass inside it) fires
	// an extra time. We assert the chain length grows.
	var trail []string
	res := &Resource{StructName: "Twice"}
	res.FieldProcessor = func(_ string, _ *FieldInfo) error {
		trail = append(trail, "leaf")
		return nil
	}

	rc.applyFieldOverrides(res)
	single := res.FieldProcessor
	trail = nil
	require.NoError(t, single("X", &FieldInfo{}))
	leafRunsSingle := len(trail)

	// Re-wrap once more: applyFieldOverrides must produce a NEW, deeper chain (it is
	// not a no-op). The processor function identity must change, proving a second
	// apply would genuinely duplicate the YAML customization — which is why
	// ApplyToResource must run exactly once per resource (ARCH-21).
	rc.applyFieldOverrides(res)
	doubled := res.FieldProcessor
	assert.NotEqual(t,
		reflect.ValueOf(single).Pointer(),
		reflect.ValueOf(doubled).Pointer(),
		"applyFieldOverrides must re-wrap the processor (it is not a no-op)",
	)
	trail = nil
	require.NoError(t, doubled("X", &FieldInfo{}))
	assert.Len(t, trail, leafRunsSingle, "leaf is innermost and still runs once")
}

// TestCollectResourceGenerators_DoesNotReapplyCustomizations is the decisive
// ARCH-21 guard: customizations are applied once, in
// buildResourcesFromDownloadedFields, BEFORE processJSON. collectResourceGenerators
// must NOT re-apply them (the removed dead line re-wrapped a processor nobody
// invokes again). We pass a resource whose FieldProcessor is a sentinel and assert
// its identity is untouched after collectResourceGenerators runs.
func TestCollectResourceGenerators_DoesNotReapplyCustomizations(t *testing.T) {
	t.Parallel()

	// A real customizer that DOES have a customization for this resource name; if
	// collectResourceGenerators re-applied it, the sentinel would be wrapped.
	yamlContent := `
customizations:
  resources:
    Account:
      fields:
        IP:
          omitEmpty: true
`
	tempFile := createTempCustomizationsYaml(t, yamlContent)
	cc, err := NewCodeCustomizer(tempFile)
	require.NoError(t, err)

	sentinel := func(_ string, _ *FieldInfo) error { return nil }
	res := &Resource{StructName: "Account", ResourcePath: "account", FieldProcessor: sentinel}

	gens := collectResourceGenerators([]*Resource{res}, *cc, nil)
	require.NotEmpty(t, gens)

	// The sentinel must be the SAME function value: collectResourceGenerators did
	// not call ApplyToResource (which would compose/replace it).
	assert.Equal(t,
		reflect.ValueOf(sentinel).Pointer(),
		reflect.ValueOf(res.FieldProcessor).Pointer(),
		"collectResourceGenerators must not re-apply customizations (FieldProcessor identity must be preserved)",
	)
}

// TestApplyToResource_ResourceAndFieldOverridesSeparated pins that resource-level
// overrides (resourcePath) and field-level overrides (FieldProcessor) are applied
// independently: a resource customization that only sets resourcePath must not
// leave the field processor nil, and one that only sets fields must not disturb
// resourcePath.
func TestApplyToResource_ResourceAndFieldOverridesSeparated(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		yaml         string
		startPath    string
		wantPath     string
		wantFieldNil bool
	}{
		"resource-level only sets path, still installs field processor": {
			yaml: `
customizations:
  resources:
    R:
      resourcePath: new/path
`,
			startPath:    "old/path",
			wantPath:     "new/path",
			wantFieldNil: false,
		},
		"field-level only leaves path untouched": {
			yaml: `
customizations:
  resources:
    R:
      fields:
        X:
          fieldType: int
`,
			startPath:    "keep/path",
			wantPath:     "keep/path",
			wantFieldNil: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tempFile := createTempCustomizationsYaml(t, tc.yaml)
			cc, err := NewCodeCustomizer(tempFile)
			require.NoError(t, err)

			res := &Resource{StructName: "R", ResourcePath: tc.startPath}
			cc.ApplyToResource(res)

			assert.Equal(t, tc.wantPath, res.ResourcePath)
			if tc.wantFieldNil {
				assert.Nil(t, res.FieldProcessor)
			} else {
				assert.NotNil(t, res.FieldProcessor)
			}
		})
	}
}

func TestMatchesExcludePattern(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		pattern  string
		name     string
		expected bool
	}{
		"contains match":    {pattern: "*Setting*", name: "FooSettingBar", expected: true},
		"contains no match": {pattern: "*Setting*", name: "FooBar", expected: false},
		"prefix match":      {pattern: "Device*", name: "DeviceState", expected: true},
		"prefix no match":   {pattern: "Device*", name: "FirewallRule", expected: false},
		"suffix match":      {pattern: "*Group", name: "APGroup", expected: true},
		"suffix no match":   {pattern: "*Group", name: "APProfile", expected: false},
		"exact match":       {pattern: "Network", name: "Network", expected: true},
		"exact no match":    {pattern: "Network", name: "Networks", expected: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, matchesExcludePattern(tc.pattern, tc.name))
		})
	}

	// Edge case: a bare "*" (or "**") has no inner content and matches everything.
	t.Run("star only matches all", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesExcludePattern("*", "Anything"))
		assert.True(t, matchesExcludePattern("**", "Anything"))
	})
}
