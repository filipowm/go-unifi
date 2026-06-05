package main

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateGolden, when set, rewrites the *.golden files under testdata/ from the
// current generator output instead of asserting against them. Regenerate with:
//
//	go test ./codegen/ -run TestResourceGenerateCodeGolden -update-golden
//
// Because GenerateCode runs format.Source, golden output is gofmt-stable and any
// syntactically invalid Go produced by the templates fails the test for free.
var updateGolden = flag.Bool("update-golden", false, "rewrite golden files under testdata/ from generator output")

// collapseSpaces folds every run of spaces/tabs into a single space so that
// alignment-sensitive Contains assertions don't churn when gofmt re-aligns a
// struct-field or alias column. It intentionally keeps newlines.
func collapseSpaces(s string) string {
	return multiSpace.ReplaceAllString(s, " ")
}

var multiSpace = regexp.MustCompile(`[ \t]+`)

// goldenPath is the on-disk location of a named golden fixture.
func goldenPath(name string) string {
	return filepath.Join("testdata", name+".golden")
}

// assertGolden compares got against the named golden file, or rewrites it when
// -update-golden is set. It does NOT run in parallel, because writing goldens
// mutates shared on-disk state.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenPath(name)

	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0o644)) //nolint:gosec
		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err, "missing golden %s; regenerate with -update-golden", path)
	assert.Equal(t, string(want), got, "generated output drifted from golden %s; if intentional, regenerate with -update-golden", path)
}

// newWidgetV1 builds a representative V1 (non-Setting) resource exercising the
// full type-inference matrix: string, int (emptyStringInt custom unmarshal),
// float64, bool, a nested struct, and an array. It additionally pins BOTH custom
// unmarshal branches of the template so neither can silently break:
//
//   - the customUnmarshalType TYPE-CAST branch (dst.X = T(aux.X)), which is what
//     production booleanishString fields like Device.LtePoe actually emit
//     (see unifi/device.generated.go: LtePoe booleanishString + bool(aux.LtePoe));
//   - the customUnmarshalFunc FUNC branch (dst.X = fn(aux.X)), which is what
//     production *bool fields like Network.InternetAccessEnabled emit via the
//     real emptyBoolToTrue helper (see unifi/network.generated.go).
//
// Construction uses the real generator API: NewResource + processJSON (which
// drives fieldInfoFromValidation / fieldInfoFromMap / fieldInfoFromArray), with
// two manual FieldInfos for the custom-unmarshal paths that have no validation
// shape (mirroring how customizations.yml injects them).
func newWidgetV1(t *testing.T) *Resource {
	t.Helper()
	r := NewResource("Widget", "widget")

	const fields = `{
		"name": ".{0,32}",
		"count": "^[0-9]*$",
		"ratio": "[-+]?[0-9]*\\.?[0-9]+",
		"enabled": "true|false",
		"tags": [".{0,32}"],
		"nested": { "inner_value": ".{0,32}" }
	}`
	require.NoError(t, r.processJSON([]byte(fields)))

	// customUnmarshalType-only field: exercises the type-cast branch
	// (dst.X = T(aux.X)). This is the exact form production Device.LtePoe emits
	// (booleanishString alias + bool(aux.LtePoe)); pinning it here keeps
	// ARCH-02's booleanishString render under golden protection.
	lte := NewFieldInfo("LtePoe", "lte_poe", "bool", "", "", false, false, "booleanishString")
	r.BaseType().Fields["LtePoe"] = lte

	// customUnmarshalFunc field: exercises the typecast "func" branch
	// (dst.X = fn(aux.X)) which no validation-inferred field reaches. It uses the
	// real emptyBoolToTrue helper + *bool alias exactly like production
	// Network.InternetAccessEnabled, so the golden is not a non-compiling fiction.
	guarded := NewFieldInfo("Guarded", "guarded", "bool", "", "", false, false, "*bool")
	guarded.CustomUnmarshalFunc = "emptyBoolToTrue"
	r.BaseType().Fields["Guarded"] = guarded

	return r
}

// newGadgetV2 builds a representative V2 resource so apiv2.go.tmpl and the IsV2
// branch of GenerateCode are exercised.
func newGadgetV2(t *testing.T) *Resource {
	t.Helper()
	r := NewResource("Gadget", "gadget")
	r.V2 = true

	const fields = `{
		"label": ".{0,32}",
		"count": "^[0-9]*$"
	}`
	require.NoError(t, r.processJSON([]byte(fields)))
	return r
}

// TestResourceGenerateCodeGolden pins the full rendered output of the V1 and V2
// resource templates against checked-in golden files. NOT parallel: with
// -update-golden it writes shared files.
func TestResourceGenerateCodeGolden(t *testing.T) {
	cases := map[string]struct {
		build func(t *testing.T) *Resource
	}{
		"widget_v1": {build: newWidgetV1},
		"gadget_v2": {build: newGadgetV2},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource := tc.build(t)

			code, err := resource.GenerateCode()
			require.NoError(t, err)
			require.NotEmpty(t, code)

			assertGolden(t, name, code)
		})
	}
}

// TestResourceGenerateCodeV1Shape asserts the structural invariants of the V1
// template independent of golden churn: the field-type matrix, the custom
// unmarshal aliasing, and the REST CRUD endpoint shape.
func TestResourceGenerateCodeV1Shape(t *testing.T) {
	t.Parallel()

	code, err := newWidgetV1(t).GenerateCode()
	require.NoError(t, err)
	a := assert.New(t)

	// gofmt aligns struct-field and alias-block columns, so assert on
	// whitespace-collapsed forms: this pins field name + Go type + json tag
	// without churning when an unrelated field widens a column.
	norm := collapseSpaces(code)

	// Type matrix lands on the expected Go types.
	a.Contains(norm, "Count int `json:\"count,omitempty\"`")
	a.Contains(norm, "Ratio float64 `json:\"ratio,omitempty\"`")
	a.Contains(norm, "Enabled bool `json:\"enabled\"`")
	a.Contains(norm, "Tags []string `json:\"tags,omitempty\"")
	a.Contains(norm, "Nested WidgetNested `json:\"nested,omitempty\"`")

	// Nested struct is emitted as its own type.
	a.Contains(code, "type WidgetNested struct {")

	// emptyStringInt custom-unmarshal alias + typecast.
	a.Contains(norm, "Count emptyStringInt `json:\"count\"`")
	a.Contains(code, "dst.Count = int(aux.Count)")

	// customUnmarshalType-only alias + type-cast typecast (the production
	// booleanishString form, ARCH-02): dst.X = T(aux.X), no func.
	a.Contains(norm, "LtePoe booleanishString `json:\"lte_poe\"`")
	a.Contains(code, "dst.LtePoe = bool(aux.LtePoe)")

	// customUnmarshalFunc alias + func typecast (the production emptyBoolToTrue
	// form): dst.X = fn(aux.X).
	a.Contains(norm, "Guarded *bool `json:\"guarded\"`")
	a.Contains(code, "dst.Guarded = emptyBoolToTrue(aux.Guarded)")

	// V1 non-Setting CRUD uses the rest/ REST endpoints + Meta-wrapped response.
	a.Contains(norm, "Meta Meta `json:\"meta\"`") //nolint:dupword // "Meta Meta" is the rendered field-name + type, not prose
	a.Contains(code, `c.Get(ctx, fmt.Sprintf("s/%s/rest/widget", site)`)
	a.Contains(code, `c.Put(ctx, fmt.Sprintf("s/%s/rest/widget/%s", site, d.ID)`)
	a.Contains(code, `c.Delete(ctx, fmt.Sprintf("s/%s/rest/widget/%s", site, id)`)
}

// TestResourceGenerateCodeV2Shape asserts the IsV2 branch: ApiV2Path-based
// endpoints and the no-Meta (bare slice / bare struct) response shape.
func TestResourceGenerateCodeV2Shape(t *testing.T) {
	t.Parallel()

	code, err := newGadgetV2(t).GenerateCode()
	require.NoError(t, err)
	a := assert.New(t)

	// V2 endpoints are built from c.apiPaths.ApiV2Path and /site/.
	a.Contains(code, "c.apiPaths.ApiV2Path")
	a.Contains(code, `fmt.Sprintf("%s/site/%s/gadget", c.apiPaths.ApiV2Path, site)`)
	a.Contains(code, `fmt.Sprintf("%s/site/%s/gadget/%s", c.apiPaths.ApiV2Path, site, id)`)

	// V2 list returns a bare slice, get a bare struct — no Meta wrapper anywhere.
	a.Contains(code, "var respBody []Gadget")
	a.Contains(code, "var respBody Gadget")
	a.NotContains(code, "Meta Meta") //nolint:dupword // asserting the V2 template emits NO "Meta Meta" wrapper field
	a.NotContains(code, "/rest/")
	a.NotContains(code, "/stat/")
}

// TestResourceGenerateCodeSettingShape pins the Setting branch: the key const,
// the exported Get/Update setting wrappers, and the absence of REST CRUD.
func TestResourceGenerateCodeSettingShape(t *testing.T) {
	t.Parallel()

	r := NewResource("SettingFoo", "")
	customizeBaseType(r)
	require.NoError(t, r.processJSON([]byte(`{"value":".{0,32}"}`)))

	code, err := r.GenerateCode()
	require.NoError(t, err)
	a := assert.New(t)

	a.True(r.IsSetting())
	a.Contains(code, `const SettingFooKey = "foo"`)
	a.Contains(collapseSpaces(code), "Key string `json:\"key\"`")
	a.Contains(code, "func (c *client) GetSettingFoo(ctx context.Context, site string) (*SettingFoo, error)")
	a.Contains(code, "func (c *client) UpdateSettingFoo(ctx context.Context, site string, s *SettingFoo) (*SettingFoo, error)")
	a.Contains(code, "c.GetSetting(ctx, site, SettingFooKey)")
	a.Contains(code, "c.SetSetting(ctx, site, SettingFooKey, s)")
	// Settings go through GetSetting/SetSetting, not the rest/ CRUD endpoints.
	a.NotContains(code, "/rest/")
}

// TestResourceGenerateCodeEndpointPaths pins the conditional list-endpoint logic
// in api.go.tmpl: Device uses stat/, APGroup uses neither stat/ nor rest/, and a
// plain resource uses rest/. These are the lines a refactor most easily breaks.
func TestResourceGenerateCodeEndpointPaths(t *testing.T) {
	t.Parallel()

	listLine := func(t *testing.T, r *Resource) string {
		t.Helper()
		code, err := r.GenerateCode()
		require.NoError(t, err)
		for line := range strings.SplitSeq(code, "\n") {
			if strings.Contains(line, "c.Get(ctx, fmt.Sprintf(\"s/%s/") {
				return line
			}
		}
		t.Fatalf("no list endpoint line found in generated code for %s", r.Name())
		return ""
	}

	t.Run("device list uses stat/", func(t *testing.T) {
		t.Parallel()
		r := NewResource("Device", "device")
		customizeBaseType(r)
		require.NoError(t, r.processJSON([]byte(`{"name":".{0,32}"}`)))

		a := assert.New(t)
		line := listLine(t, r)
		a.Contains(line, "stat/")
		a.Contains(line, "s/%s/stat/device")
	})

	t.Run("apgroup list uses neither stat/ nor rest/", func(t *testing.T) {
		t.Parallel()
		r := NewResource("APGroup", "apgroups")
		require.NoError(t, r.processJSON([]byte(`{"name":".{0,32}"}`)))

		a := assert.New(t)
		line := listLine(t, r)
		// The APGroup *list* endpoint is the special branch: bare s/%s/apgroups.
		a.NotContains(line, "stat/")
		a.NotContains(line, "rest/")
		a.Contains(line, "s/%s/apgroups")
	})

	t.Run("plain resource list uses rest/", func(t *testing.T) {
		t.Parallel()
		r := NewResource("Widget", "widget")
		require.NoError(t, r.processJSON([]byte(`{"name":".{0,32}"}`)))

		a := assert.New(t)
		line := listLine(t, r)
		a.Contains(line, "rest/")
		a.Contains(line, "s/%s/rest/widget")
	})
}
