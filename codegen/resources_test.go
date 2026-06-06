package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceWarningsUseInjectedLogger pins that a Resource routes its
// dropped-field and CamelCase-collision WARNs through its INJECTED logger, not
// the package global, so the assertions read only this test's own hook and the
// test is fully parallel-safe. buildResourcesFromDownloadedFields wires the
// injected logger onto every resource it builds.
func TestResourceWarningsUseInjectedLogger(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// 123 is not an inferable validation -> dropped field WARN.
	// "id" and "i_d" both normalize to Go field "ID" -> collision WARN.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dropper.json"), []byte(`{"bad": 123}`), 0o644))                  //nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Collider.json"), []byte(`{"id": "", "i_d": ".{0,32}"}`), 0o644)) //nolint:gosec

	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	resources, err := buildResourcesFromDownloadedFields(tmpDir, CodeCustomizer{}, false, logger)
	require.NoError(t, err)
	require.NotEmpty(t, resources)

	var warns []string
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			warns = append(warns, e.Message)
		}
	}
	require.NotEmpty(t, warns, "injected logger must capture resource WARNs")
	assertContainsSubstr(t, warns, "dropping field")
	assertContainsSubstr(t, warns, "CamelCase collision on Go field")
}

// assertContainsSubstr asserts at least one string in msgs contains substr.
func assertContainsSubstr(t *testing.T, msgs []string, substr string) {
	t.Helper()
	for _, m := range msgs {
		if strings.Contains(m, substr) {
			return
		}
	}
	t.Errorf("no logged message contained %q; got %v", substr, msgs)
}

func TestFieldInfoFromValidation(t *testing.T) {
	t.Parallel()

	for i, c := range []struct {
		expectedType      string
		expectedComment   string
		expectedOmitEmpty bool
		validation        any
	}{
		{"string", "", true, ""},
		{"string", "default|custom", true, "default|custom"},
		{"string", ".{0,32}", true, ".{0,32}"},
		{"string", "^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$", false, "^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$"},

		{"int", "^([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$", true, "^([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$"},
		{"int", "", true, "^[0-9]*$"},

		{"float64", "", true, "[-+]?[0-9]*\\.?[0-9]+"},
		// this one is really an error as the . is not escaped
		{"float64", "", true, "^([-]?[\\d]+[.]?[\\d]*)$"},
		{"float64", "", true, "^([\\d]+[.]?[\\d]*)$"},

		{"bool", "", false, "false|true"},
		{"bool", "", false, "true|false"},
	} {
		t.Run(fmt.Sprintf("%d %s %s", i, c.expectedType, c.validation), func(t *testing.T) {
			t.Parallel()

			resource := &Resource{
				StructName:     "TestType",
				Types:          make(map[string]*FieldInfo),
				FieldProcessor: func(name string, f *FieldInfo) error { return nil },
			}

			fieldInfo, err := resource.fieldInfoFromValidation("fieldName", c.validation, false)
			// actualType, actualComment, actualOmitEmpty, err := fieldInfoFromValidation(c.validation)
			if err != nil {
				t.Fatal(err)
			}
			if fieldInfo.FieldType != c.expectedType {
				t.Fatalf("expected type %q got %q", c.expectedType, fieldInfo.FieldType)
			}
			if fieldInfo.FieldValidationComment != c.expectedComment {
				t.Fatalf("expected comment %q got %q", c.expectedComment, fieldInfo.FieldValidationComment)
			}
			if fieldInfo.OmitEmpty != c.expectedOmitEmpty {
				t.Fatalf("expected omitempty %t got %t", c.expectedOmitEmpty, fieldInfo.OmitEmpty)
			}
		})
	}
}

func TestFieldInfoFromValidationDetails(t *testing.T) {
	t.Parallel()

	// These cases lock in the tricky numeric/float/int/IP-octet/bool branches
	// of fieldInfoFromValidation that a refactor could silently break.
	cases := map[string]struct {
		fieldName     string
		validation    any
		isArray       bool
		expectedType  string
		expectedField string // FieldName after cleanName/ToCamel
		expectComment string
		expectOmit    bool
		expectUnmarsh string
		expectIsArray bool
	}{
		"int field uses emptyStringInt unmarshal": {
			fieldName:     "max_value",
			validation:    "^[0-9]*$",
			expectedType:  "int",
			expectedField: "MaxValue",
			expectComment: "", // normalized "09" blanks the comment
			expectOmit:    true,
			expectUnmarsh: "emptyStringInt",
		},
		"int field keeps non-09 comment": {
			fieldName:     "octet",
			validation:    "^([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$",
			expectedType:  "int",
			expectedField: "Octet",
			expectComment: "^([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$",
			expectOmit:    true,
			expectUnmarsh: "emptyStringInt",
		},
		"float64 field blanks 09.09 comment": {
			fieldName:     "ratio",
			validation:    "[-+]?[0-9]*\\.?[0-9]+",
			expectedType:  "float64",
			expectedField: "Ratio",
			expectComment: "",
			expectOmit:    true,
			expectUnmarsh: "",
		},
		"IP-octet pattern falls through to string with original comment": {
			fieldName:     "gateway_ip",
			validation:    "^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$",
			expectedType:  "string",
			expectedField: "GatewayIP",
			expectComment: "^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$",
			expectOmit:    false, // contains "^$" -> omitempty false
			expectUnmarsh: "",
		},
		"bool from false|true": {
			fieldName:     "enabled",
			validation:    "false|true",
			expectedType:  "bool",
			expectedField: "Enabled",
			expectComment: "",
			expectOmit:    false,
			expectUnmarsh: "",
		},
		"bool from true|false": {
			fieldName:     "active",
			validation:    "true|false",
			expectedType:  "bool",
			expectedField: "Active",
			expectComment: "",
			expectOmit:    false,
			expectUnmarsh: "",
		},
		"plain string field omits empty when no ^$ and not ID suffix": {
			fieldName:     "note",
			validation:    ".{0,32}",
			expectedType:  "string",
			expectedField: "Note",
			expectComment: ".{0,32}",
			expectOmit:    true,
			expectUnmarsh: "",
		},
		"string field with ID suffix does not omit empty": {
			fieldName:     "site_id",
			validation:    "",
			expectedType:  "string",
			expectedField: "SiteID",
			expectComment: "",
			expectOmit:    false,
			expectUnmarsh: "",
		},
		"single-element array sets IsArray and OmitEmpty": {
			fieldName:     "tags",
			validation:    []any{".{0,32}"},
			expectedType:  "string",
			expectedField: "Tags",
			expectComment: ".{0,32}",
			expectOmit:    true,
			expectIsArray: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			resource := &Resource{
				StructName:     "TestType",
				Types:          make(map[string]*FieldInfo),
				FieldProcessor: func(_ string, _ *FieldInfo) error { return nil },
			}

			fieldInfo, err := resource.fieldInfoFromValidation(tc.fieldName, tc.validation, tc.isArray)
			require.NoError(t, err)
			require.NotNil(t, fieldInfo)

			a.Equal(tc.expectedType, fieldInfo.FieldType, "FieldType")
			a.Equal(tc.expectedField, fieldInfo.FieldName, "FieldName")
			a.Equal(tc.expectComment, fieldInfo.FieldValidationComment, "FieldValidationComment")
			a.Equal(tc.expectOmit, fieldInfo.OmitEmpty, "OmitEmpty")
			a.Equal(tc.expectUnmarsh, fieldInfo.CustomUnmarshalType, "CustomUnmarshalType")
			a.Equal(tc.expectIsArray, fieldInfo.IsArray, "IsArray")
		})
	}
}

func TestResourceTypes(t *testing.T) {
	t.Parallel()

	testData := `
{
  "note": ".{0,1024}",
  "date": "^$|^(20[0-9]{2}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9])Z?$",
  "mac": "^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$",
  "number": "\\d+",
  "boolean": "true|false",
	"nested_type": {
    "nested_field": "^$"
  },
  "nested_type_array": [{
    "nested_field": "^$"
  }]
}
	`
	expectedFields := map[string]*FieldInfo{
		"Note":    NewFieldInfo("Note", "note", "string", "validate:\"omitempty,gte=0,lte=1024\"", ".{0,1024}", true, false, ""),
		"Date":    NewFieldInfo("Date", "date", "string", "", "^$|^(20[0-9]{2}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9])Z?$", false, false, ""),
		"MAC":     NewFieldInfo("MAC", "mac", "string", "validate:\"omitempty,mac\"", "^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$", true, false, ""),
		"Number":  NewFieldInfo("Number", "number", "int", "", "", true, false, "emptyStringInt"),
		"Boolean": NewFieldInfo("Boolean", "boolean", "bool", "", "", false, false, ""),
		"NestedType": {
			FieldName:              "NestedType",
			JSONName:               "nested_type",
			FieldType:              "StructNestedType",
			FieldValidationComment: "",
			OmitEmpty:              true,
			IsArray:                false,
			Fields: map[string]*FieldInfo{
				"NestedFieldModified": NewFieldInfo("NestedFieldModified", "nested_field", "string", "", "^$", false, false, ""),
			},
		},
		"NestedTypeArray": {
			FieldName:              "NestedTypeArray",
			JSONName:               "nested_type_array",
			FieldType:              "StructNestedTypeArray",
			FieldValidationComment: "",
			OmitEmpty:              true,
			IsArray:                true,
			Fields: map[string]*FieldInfo{
				"NestedFieldModified": NewFieldInfo("NestedFieldModified", "nested_field", "string", "", "^$", false, false, ""),
			},
		},
	}

	expectedStruct := map[string]*FieldInfo{
		"Struct": {
			FieldName:              "Struct",
			JSONName:               "path",
			FieldType:              "struct",
			FieldValidationComment: "",
			OmitEmpty:              false,
			IsArray:                false,
			Fields: map[string]*FieldInfo{
				"   ID":      NewFieldInfo("ID", "_id", "string", "", "", true, false, ""),
				"   SiteID":  NewFieldInfo("SiteID", "site_id", "string", "", "", true, false, ""),
				"   _Spacer": nil,
				"  Hidden":   NewFieldInfo("Hidden", "attr_hidden", "bool", "", "", true, false, ""),
				"  HiddenID": NewFieldInfo("HiddenID", "attr_hidden_id", "string", "", "", true, false, ""),
				"  NoDelete": NewFieldInfo("NoDelete", "attr_no_delete", "bool", "", "", true, false, ""),
				"  NoEdit":   NewFieldInfo("NoEdit", "attr_no_edit", "bool", "", "", true, false, ""),
				"  _Spacer":  nil,
				" _Spacer":   nil,
			},
		},
	}

	maps.Copy(expectedStruct["Struct"].Fields, expectedFields)

	expectation := &Resource{
		StructName:   "Struct",
		ResourcePath: "path",

		Types: map[string]*FieldInfo{
			"Struct":                expectedStruct["Struct"],
			"StructNestedType":      expectedStruct["Struct"].Fields["NestedType"],
			"StructNestedTypeArray": expectedStruct["Struct"].Fields["NestedTypeArray"],
		},

		FieldProcessor: func(name string, f *FieldInfo) error {
			if name == "NestedField" {
				f.FieldName = "NestedFieldModified"
			}
			return nil
		},
	}

	t.Run("structural test", func(t *testing.T) {
		t.Parallel()

		resource := NewResource("Struct", "path")
		resource.FieldProcessor = expectation.FieldProcessor

		err := resource.processJSON(([]byte)(testData))

		require.NoError(t, err, "No error processing JSON")
		assert.Equal(t, expectation.StructName, resource.StructName)
		assert.Equal(t, expectation.ResourcePath, resource.ResourcePath)
		assert.Equal(t, expectation.Types, resource.Types)
	})
}

func TestNormalizeValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"\\d+", "09"},
		{"[-+]?[0-9]*\\.?[0-9]+", "09.09"},
		{"^([0-9]|[1-9][0-9]|25[0-5])$", "0919092505"},
		{"^(([0-9]\\.[0-9]{2})\\.){3}([0-9]\\.[0-9])$", "09.09.09.09"},
		{"[+-]?[0-9]*\\.?[0-9]+", "09.09"},
		{"[-]?[\\d]+[.]?[\\d]*", "09.09"},
		{"^$|^(20[0-9]{2}T([01][0-9]):[1-5]:[0-9])Z?$", "2009T0109:15:09Z"},
		{"false|true", "falsetrue"},
		{"true|false", "truefalse"},
		{".{0,32}", "."},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			actual := normalizeValidation(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

var testReps = []replacement{
	{"dhcpd", "DHCPD"},
	{"ip", "IP"},
}

func TestCleanName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		reps     []replacement
		expected string
	}{
		{"field replacements basic", "dhcpd_enabled", testReps, "DHCPD_enabled"},
		{"field replacements multiple", "dhcpd_ip_mac", testReps, "DHCPD_IP_mac"},
		{"field replacements no match", "something_else", testReps, "something_else"},
		{"empty string", "", fieldReps, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			actual := cleanName(tc.input, tc.reps)
			a.Equal(tc.expected, actual)
		})
	}
}

func TestIsSetting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		structName string
		expected   bool
	}{
		{"Setting", true},
		{"SettingUsg", true},
		{"SettingGlobalAp", true},
		{"Settings", true},
		{"Device", false},
		{"Network", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.structName, func(t *testing.T) {
			t.Parallel()
			resource := &Resource{StructName: tc.structName}
			assert.Equal(t, tc.expected, resource.IsSetting())
		})
	}
}

func TestFieldInfoFromValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fieldName     string
		validation    any
		errorContains string
	}{
		{
			"invalid validation type",
			"field",
			123,
			"unable to determine type from validation",
		},
		{
			"empty array",
			"field",
			[]any{},
			"",
		},
		{
			"array with multiple items",
			"field",
			[]any{"item1", "item2"},
			"unknown validation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			resource := NewResource("Test", "test")
			fieldInfo, err := resource.fieldInfoFromValidation(tc.fieldName, tc.validation, false)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
				a.NotNil(fieldInfo)
				a.Equal(&FieldInfo{}, fieldInfo)
			} else {
				require.NoError(t, err)
				a.NotNil(fieldInfo)
			}
		})
	}
}

// TestFieldInfoFromMapSkipsFailingChild pins the nested-struct robustness:
// a single un-inferable nested child is dropped (not the whole struct + its
// siblings). In the default (non-strict) mode the nested struct is still
// produced, carrying every well-formed sibling, and only the bad child is
// missing. This is the nested mirror of processFields' per-field skip.
func TestFieldInfoFromMapSkipsFailingChild(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	resource := NewResource("Test", "test")
	fieldInfo, err := resource.fieldInfoFromValidation("outer", map[string]any{
		"good":    ".{0,32}",
		"bad":     123, // un-inferable -> dropped
		"also_ok": "true|false",
	}, false)

	require.NoError(t, err, "non-strict mode must not error on a single bad nested child")
	require.NotNil(t, fieldInfo)
	a.Equal("TestOuter", fieldInfo.FieldType)

	// The two well-formed siblings survive; the bad child is dropped.
	a.NotNil(fieldByJSONName(fieldInfo.Fields, "good"), "good sibling retained")
	a.NotNil(fieldByJSONName(fieldInfo.Fields, "also_ok"), "also_ok sibling retained")
	a.Nil(fieldByJSONName(fieldInfo.Fields, "bad"), "bad child dropped")
	a.Len(fieldInfo.Fields, 2, "only the two inferable children remain")
}

// TestStrictModeDroppedField pins the strict-mode opt-in: with
// UNIFI_CODEGEN_STRICT set, an un-inferable field is a HARD error rather than a
// silently-dropped WARN, both at the top level (processFields via processJSON)
// and inside a nested struct (fieldInfoFromMap). NOT parallel: mutates a process
// env var that strictMode() reads globally.
func TestStrictModeDroppedField(t *testing.T) {
	t.Setenv(strictEnvVar, "1")

	t.Run("top-level field", func(t *testing.T) {
		resource := NewResource("Test", "test")
		err := resource.processJSON([]byte(`{"bad": 123}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `dropping field "bad"`)
	})

	t.Run("nested child", func(t *testing.T) {
		resource := NewResource("Test", "test")
		_, err := resource.fieldInfoFromValidation("outer", map[string]any{"bad": 123}, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `dropping field "TestOuter.bad"`)
	})
}

// TestStrictModeAbortsGeneration pins that under strict mode a field-drop in a
// single resource file aborts the whole buildResourcesFromDownloadedFields run
// (returns an error) instead of skipping that file, while an ordinary
// malformed-JSON file is still skipped (parse errors are not strict violations).
// NOT parallel: sets a process env var.
func TestStrictModeAbortsGeneration(t *testing.T) {
	t.Setenv(strictEnvVar, "1")

	tmpDir := t.TempDir()
	// Drops a field (123 is not an inferable validation) -> strict violation.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dropper.json"), []byte(`{"bad": 123}`), 0o644)) //nolint:gosec

	_, err := buildResourcesFromDownloadedFields(tmpDir, CodeCustomizer{}, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict mode")
	assert.Contains(t, err.Error(), "Dropper.json")
}

// TestStrictModeSkipsMalformedJSON confirms a malformed-JSON file is still a
// per-file skip (not a strict abort) even in strict mode: a JSON parse error is
// not a field-drop/collision strict violation. NOT parallel: env var.
func TestStrictModeSkipsMalformedJSON(t *testing.T) {
	t.Setenv(strictEnvVar, "1")

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Bad.json"), []byte(`not json`), 0o644))             //nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Good.json"), []byte(`{"name": ".{0,32}"}`), 0o644)) //nolint:gosec

	resources, err := buildResourcesFromDownloadedFields(tmpDir, CodeCustomizer{}, false, nil)
	require.NoError(t, err, "malformed JSON must be skipped, not abort generation, even in strict mode")
	assert.Len(t, resources, 1)
	assert.Equal(t, "Good", resources[0].StructName)
}

// TestStrictModeDisabledByDefault confirms the env var is OFF unless explicitly
// set to a truthy value, preserving the daily auto-regen's best-effort behavior.
func TestStrictModeDisabledByDefault(t *testing.T) {
	t.Setenv(strictEnvVar, "")
	assert.False(t, strictMode(), "empty env var must not enable strict mode")
	t.Setenv(strictEnvVar, "0")
	assert.False(t, strictMode(), "0 must not enable strict mode")
	t.Setenv(strictEnvVar, "1")
	assert.True(t, strictMode(), "1 must enable strict mode")
	t.Setenv(strictEnvVar, "true")
	assert.True(t, strictMode(), "true must enable strict mode")
}

// TestFieldNameCollision pins the collision detection: two distinct JSON
// keys that normalize to the SAME Go field name (e.g. "id" and "ID" both ->
// "ID") collide. In non-strict mode the deterministic key sort decides the
// surviving winner; in strict mode it is a hard error.
func TestFieldNameCollision(t *testing.T) {
	// Two JSON keys collide on Go name "Foobar":
	//   "foobar" -> ToCamel -> "Foobar"
	//   "foo_bar" -> ToCamel -> "FooBar" (no collision) ... so craft a real one:
	//   "id" -> cleanName -> "ID"; "i_d" -> ToCamel -> "ID" -> collision.
	const colliding = `{"id": "", "i_d": ".{0,32}"}`

	t.Run("non-strict resolves deterministically", func(t *testing.T) {
		resource := NewResource("Test", "test")
		// Sorted order: "i_d" < "id". "i_d" is processed and assigned to the Go
		// field "ID" first, then "id" collides and (in non-strict) overwrites it.
		// The base-struct's injected ID lives under a whitespace-prefixed key and
		// does not participate, so t.Fields["ID"] holds exactly the colliding
		// winner — and which key wins is stable across runs.
		err := resource.processJSON([]byte(colliding))
		require.NoError(t, err)
		got := resource.BaseType().Fields["ID"]
		require.NotNil(t, got)
		assert.Equal(t, "id", got.JSONName, "later-sorted key wins deterministically")
	})

	t.Run("strict mode errors", func(t *testing.T) {
		t.Setenv(strictEnvVar, "1")
		resource := NewResource("Test", "test")
		err := resource.processJSON([]byte(colliding))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collision on Go field")
	})
}

func TestBuildResourcesFromDownloadedFields(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create test JSON files
	validJSON := `{
		"name": "test",
		"value": "^[0-9]*$",
		"enabled": "true|false"
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "Test.json"), []byte(validJSON), 0o644) //nolint:gosec
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "Invalid.json"), []byte("invalid json"), 0o644) //nolint:gosec
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "Setting.json"), []byte(validJSON), 0o644) //nolint:gosec
	require.NoError(t, err)

	// Test cases
	tests := []struct {
		name          string
		dir           string
		expectedLen   int
		errorContains string
	}{
		{
			"valid directory",
			tmpDir,
			1, // Only Test.json should be processed (Setting.json is skipped, Invalid.json fails)
			"",
		},
		{
			"non-existent directory",
			"non-existent",
			0,
			"unable to read fields directory",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			resources, err := buildResourcesFromDownloadedFields(tc.dir, CodeCustomizer{}, false, nil)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
				a.Nil(resources)
			} else {
				require.NoError(t, err)
				a.Len(resources, tc.expectedLen)
				if tc.expectedLen > 0 {
					a.Equal("Test", resources[0].StructName)
					a.Equal("test", resources[0].ResourcePath)
				}
			}
		})
	}
}

// fieldByJSONName scans a base type's fields for one whose JSONName matches.
// The map keys in baseType.Fields are not the field names (they carry sorting
// whitespace prefixes), so look-ups must go through JSONName.
func fieldByJSONName(fields map[string]*FieldInfo, jsonName string) *FieldInfo {
	for _, f := range fields {
		if f != nil && f.JSONName == jsonName {
			return f
		}
	}
	return nil
}

// TestCustomizeBaseType pins the per-resource fields that customizeBaseType
// injects into the base struct for backwards compatibility. These are
// non-generated fields the controller no longer emits (or never did) but that
// the library must keep producing so existing consumers do not break.
func TestCustomizeBaseType(t *testing.T) {
	t.Parallel()

	type expectedField struct {
		jsonName  string
		fieldName string
		fieldType string
	}

	cases := map[string]struct {
		structName string
		present    []expectedField
		// absent JSON names assert that fields injected for OTHER resources do
		// not leak into this one.
		absent []string
	}{
		"Device injects mac/adopted/model/state/type": {
			structName: "Device",
			present: []expectedField{
				{"mac", "MAC", "string"},
				{"adopted", "Adopted", "bool"},
				{"model", "Model", "string"},
				{"state", "State", "DeviceState"},
				{"type", "Type", "string"},
			},
			absent: []string{"key", "ip", "dev_id_override", "wlangroup_id"},
		},
		"User injects ip/dev_id_override": {
			structName: "User",
			present: []expectedField{
				{"ip", "IP", "string"},
				{"dev_id_override", "DevIdOverride", "int"},
			},
			absent: []string{"key", "mac", "wlangroup_id"},
		},
		"WLAN injects wlangroup_id": {
			structName: "WLAN",
			present: []expectedField{
				{"wlangroup_id", "WLANGroupID", "string"},
			},
			absent: []string{"key", "ip", "mac"},
		},
		"SettingUsg injects key/mdns_enabled": {
			structName: "SettingUsg",
			present: []expectedField{
				{"key", "Key", "string"},
				{"mdns_enabled", "MdnsEnabled", "bool"},
			},
			absent: []string{"ip", "mac", "wlangroup_id"},
		},
		"other Setting injects key but not mdns_enabled": {
			structName: "SettingMgmt",
			present: []expectedField{
				{"key", "Key", "string"},
			},
			absent: []string{"mdns_enabled", "ip", "mac", "wlangroup_id"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			resource := NewResource(tc.structName, "path")
			customizeBaseType(resource)

			fields := resource.BaseType().Fields
			for _, ef := range tc.present {
				f := fieldByJSONName(fields, ef.jsonName)
				if a.NotNilf(f, "expected field with json %q to be injected", ef.jsonName) {
					a.Equalf(ef.fieldName, f.FieldName, "FieldName for json %q", ef.jsonName)
					a.Equalf(ef.fieldType, f.FieldType, "FieldType for json %q", ef.jsonName)
				}
			}
			for _, jsonName := range tc.absent {
				a.Nilf(fieldByJSONName(fields, jsonName), "field with json %q must not be injected for %s", jsonName, tc.structName)
			}
		})
	}
}

// TestCustomizeBaseTypeValidations pins the validator tags attached to the
// injected MAC (Device) and IP (User) fields, since those drive runtime
// validation in the generated client.
func TestCustomizeBaseTypeValidations(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		structName    string
		jsonName      string
		wantValidate  string
		wantOmitEmpty bool
	}{
		"Device MAC gets mac validator": {
			structName:    "Device",
			jsonName:      "mac",
			wantValidate:  createValidations(false, validation{v: mac}),
			wantOmitEmpty: true,
		},
		"User IP gets ip validator": {
			structName:    "User",
			jsonName:      "ip",
			wantValidate:  createValidations(false, validation{v: ip}),
			wantOmitEmpty: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			resource := NewResource(tc.structName, "path")
			customizeBaseType(resource)

			f := fieldByJSONName(resource.BaseType().Fields, tc.jsonName)
			require.NotNil(t, f)
			a.Equal(tc.wantValidate, f.FieldValidation)
			a.Equal(tc.wantOmitEmpty, f.OmitEmpty)
		})
	}
}

// TestCustomizeResourceFieldProcessor pins the FieldProcessor side-effects that
// customizeResource installs for the special-cased settings resources. These
// rewrites are backwards-compat-critical and easy to break in a refactor.
func TestCustomizeResourceFieldProcessor(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		structName string
		// inputName is the cleaned CamelCase field name passed to the processor.
		inputName     string
		inField       *FieldInfo
		wantFieldName string
		wantFieldType string
		wantUnmarsh   string
	}{
		"SettingGlobalAp rewrites 6E-prefixed field to SixE": {
			structName:    "SettingGlobalAp",
			inputName:     "6EEnabled",
			inField:       NewFieldInfo("6EEnabled", "6e_enabled", "bool", "", "", false, false, ""),
			wantFieldName: "SixEEnabled",
			wantFieldType: "bool",
		},
		"SettingGlobalAp leaves non-6E field untouched": {
			structName:    "SettingGlobalAp",
			inputName:     "Enabled",
			inField:       NewFieldInfo("Enabled", "enabled", "bool", "", "", false, false, ""),
			wantFieldName: "Enabled",
			wantFieldType: "bool",
		},
		"SettingUsg rewrites *Timeout field to int/emptyStringInt": {
			structName:    "SettingUsg",
			inputName:     "SessionTimeout",
			inField:       NewFieldInfo("SessionTimeout", "session_timeout", "string", "", "", true, false, ""),
			wantFieldName: "SessionTimeout",
			wantFieldType: "int",
			wantUnmarsh:   "emptyStringInt",
		},
		"SettingUsg leaves ArpCacheTimeout untouched": {
			structName:    "SettingUsg",
			inputName:     "ArpCacheTimeout",
			inField:       NewFieldInfo("ArpCacheTimeout", "arp_cache_timeout", "string", "", "", true, false, ""),
			wantFieldName: "ArpCacheTimeout",
			wantFieldType: "string",
			wantUnmarsh:   "",
		},
		"SettingMgmt rewrites XSshKeys field type to nested struct": {
			structName:    "SettingMgmt",
			inputName:     "XSshKeys",
			inField:       NewFieldInfo("XSshKeys", "x_ssh_keys", "string", "", "", true, false, ""),
			wantFieldName: "XSshKeys",
			wantFieldType: "SettingMgmtXSshKeys",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			resource := NewResource(tc.structName, "path")
			customizeResource(resource, false)

			err := resource.FieldProcessor(tc.inputName, tc.inField)
			require.NoError(t, err)

			a.Equal(tc.wantFieldName, tc.inField.FieldName, "FieldName")
			a.Equal(tc.wantFieldType, tc.inField.FieldType, "FieldType")
			a.Equal(tc.wantUnmarsh, tc.inField.CustomUnmarshalType, "CustomUnmarshalType")
		})
	}
}

// TestCustomizeResourceSettingMgmtRegistersNestedType pins that customizeResource
// registers the x_ssh_keys nested struct in resource.Types so the generator
// emits a SettingMgmtXSshKeys type with the expected sub-fields.
func TestCustomizeResourceSettingMgmtRegistersNestedType(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	resource := NewResource("SettingMgmt", "path")
	customizeResource(resource, false)

	nested, ok := resource.Types["SettingMgmtXSshKeys"]
	require.True(t, ok, "SettingMgmtXSshKeys must be registered in resource.Types")
	require.NotNil(t, nested)

	a.Equal("SettingMgmtXSshKeys", nested.FieldName)
	a.Equal("x_ssh_keys", nested.JSONName)
	a.Equal("struct", nested.FieldType)

	// The nested struct must expose the SSH-key sub-fields by their JSON names.
	wantSubFields := map[string]string{ // json name -> Go field name
		"name":        "Name",
		"type":        "KeyType",
		"key":         "Key",
		"comment":     "Comment",
		"date":        "Date",
		"fingerprint": "Fingerprint",
	}
	for jsonName, fieldName := range wantSubFields {
		f := fieldByJSONName(nested.Fields, jsonName)
		if a.NotNilf(f, "nested field with json %q", jsonName) {
			a.Equalf(fieldName, f.FieldName, "Go field name for json %q", jsonName)
		}
	}
}

// TestCustomizeResourceV2Flag pins that customizeResource propagates the v2
// flag onto the resource so the V2 template is selected at render time.
func TestCustomizeResourceV2Flag(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		v2     bool
		wantV2 bool
	}{
		"v2 true sets V2":   {v2: true, wantV2: true},
		"v2 false keeps V2": {v2: false, wantV2: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resource := NewResource("Network", "path")
			customizeResource(resource, tc.v2)
			assert.Equal(t, tc.wantV2, resource.IsV2())
		})
	}
}
