package main

import (
	"os"
	"path/filepath"
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
	// Not parallel: inspects the package-level logger via a local hook.
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

	hook := test.NewLocal(log)
	defer hook.Reset()

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
