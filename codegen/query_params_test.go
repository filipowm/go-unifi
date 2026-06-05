package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildQueryString pins the deterministic, URL-encoded rendering of the
// queryParams customization (ARCH-19): keys sorted, values encoded, no leading
// "?".
func TestBuildQueryString(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		params map[string]string
		want   string
	}{
		"single":          {map[string]string{"includeSystemFeatures": "true"}, "includeSystemFeatures=true"},
		"sorted multiple": {map[string]string{"b": "2", "a": "1"}, "a=1&b=2"},
		"encoded value":   {map[string]string{"q": "a b&c"}, "q=a+b%26c"},
		"empty":           {map[string]string{}, ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, buildQueryString(tc.params))
		})
	}
}

// TestQuerySuffix pins that QuerySuffix prepends "?" only when query params are
// declared, so templates can splice it directly into a URL.
func TestQuerySuffix(t *testing.T) {
	t.Parallel()

	none := NewResource("Foo", "foo")
	assert.Empty(t, none.QuerySuffix(), "no query params -> empty suffix")

	with := NewResource("Bar", "bar")
	with.QueryString = "includeSystemFeatures=true"
	assert.Equal(t, "?includeSystemFeatures=true", with.QuerySuffix())

	// A query value that url-encodes to contain '%' (e.g. '&' -> '%26') must have
	// the '%' DOUBLED, because templates splice QuerySuffix into a fmt.Sprintf
	// FORMAT-string literal; a lone '%' would be read as a (malformed) verb and
	// trip go vet / corrupt the generated URL. See ARCH-19 / FR-codegen-templates-1.
	enc := NewResource("Baz", "baz")
	enc.QueryString = buildQueryString(map[string]string{"q": "a&b"}) // -> "q=a%26b"
	assert.Equal(t, "?q=a%%26b", enc.QuerySuffix(), "percent signs must be escaped for the fmt.Sprintf format literal")
}

// TestQueryParamsAppendedAfterIDSegmentV2 is the core ARCH-19 regression: the V2
// template must append the query string AFTER the "/%s" id segment on
// get/update/delete URLs (and after the bare path on list/create), so the id
// never lands behind the query string the way the old resourcePath "?" hack did.
func TestQueryParamsAppendedAfterIDSegmentV2(t *testing.T) {
	t.Parallel()

	r := NewResource("Gadget", "described-features")
	r.V2 = true
	r.QueryString = "includeSystemFeatures=true"
	require.NoError(t, r.processJSON([]byte(`{"label":".{0,32}"}`)))

	code, err := r.GenerateCode()
	require.NoError(t, err)
	a := assert.New(t)

	// list / create: bare path then query.
	a.Contains(code, `fmt.Sprintf("%s/site/%s/described-features?includeSystemFeatures=true", c.apiPaths.ApiV2Path, site)`)
	// get / delete: id segment BEFORE the query.
	a.Contains(code, `fmt.Sprintf("%s/site/%s/described-features/%s?includeSystemFeatures=true", c.apiPaths.ApiV2Path, site, id)`)
	// update: id from d.ID before the query.
	a.Contains(code, `fmt.Sprintf("%s/site/%s/described-features/%s?includeSystemFeatures=true", c.apiPaths.ApiV2Path, site, d.ID)`)

	// The malformed "?...=.../%s" shape (query before id) must NEVER be emitted.
	a.NotContains(code, "?includeSystemFeatures=true/%s")
}

// TestQueryParamsAppendedAfterIDSegmentV1 mirrors the V2 assertion for the v1 REST
// template so a query param never lands before the id there either.
func TestQueryParamsAppendedAfterIDSegmentV1(t *testing.T) {
	t.Parallel()

	r := NewResource("Widget", "widget")
	r.QueryString = "expand=true"
	require.NoError(t, r.processJSON([]byte(`{"name":".{0,32}"}`)))

	code, err := r.GenerateCode()
	require.NoError(t, err)
	a := assert.New(t)

	a.Contains(code, `fmt.Sprintf("s/%s/rest/widget?expand=true", site)`)
	a.Contains(code, `fmt.Sprintf("s/%s/rest/widget/%s?expand=true", site, id)`)
	a.Contains(code, `fmt.Sprintf("s/%s/rest/widget/%s?expand=true", site, d.ID)`)
	a.NotContains(code, "?expand=true/%s")
}

// TestNoQueryParamsLeavesURLsUnchanged pins that a resource with no queryParams
// renders exactly the legacy URLs (no stray "?"), so the QuerySuffix plumbing is
// inert by default.
func TestNoQueryParamsLeavesURLsUnchanged(t *testing.T) {
	t.Parallel()

	r := NewResource("Widget", "widget")
	require.NoError(t, r.processJSON([]byte(`{"name":".{0,32}"}`)))

	code, err := r.GenerateCode()
	require.NoError(t, err)

	assert.NotContains(t, code, "?", "no query params must leave URLs free of any query string")
	assert.Contains(t, code, `fmt.Sprintf("s/%s/rest/widget/%s", site, id)`)
}

// TestValidateResourcePathStrictRejectsRawQuery is the ARCH-19 interim guard: a
// raw "?" smuggled into resourcePath is a HARD error under strict mode, so nobody
// can re-introduce the malformed-URL footgun. NOT parallel: sets a process env
// var read by strictMode().
func TestValidateResourcePathStrictRejectsRawQuery(t *testing.T) {
	t.Setenv(strictEnvVar, "1")

	r := NewResource("Foo", "foo?includeSystemFeatures=true")
	err := r.validateResourcePath()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "raw query string")
	assert.Contains(t, err.Error(), "queryParams")

	// A clean path is fine.
	clean := NewResource("Bar", "bar")
	require.NoError(t, clean.validateResourcePath())
}

// TestValidateResourcePathNonStrictWarns confirms the guard only warns (no error)
// when strict mode is off, preserving the daily auto-regen's best-effort behavior.
// NOT parallel: env var.
func TestValidateResourcePathNonStrictWarns(t *testing.T) {
	t.Setenv(strictEnvVar, "")

	r := NewResource("Foo", "foo?q=1")
	assert.NoError(t, r.validateResourcePath(), "non-strict mode must warn, not error")
}

// TestDescribedFeatureMigratedToQueryParams pins that the production customizations
// migrate DescribedFeature off the raw "?" resourcePath hack and onto queryParams,
// and that the resulting generated URLs are well-formed (id before the query). It
// runs the real customizer over the resource so the YAML is exercised end-to-end.
func TestDescribedFeatureMigratedToQueryParams(t *testing.T) {
	t.Parallel()

	customizer, err := NewCodeCustomizer(defaultCustomizationsPath)
	require.NoError(t, err)

	r := NewResource("DescribedFeature", "describedfeature")
	r.V2 = true
	customizer.ApplyToResource(r)

	a := assert.New(t)
	a.Equal("described-features", r.ResourcePath, "raw '?' must be gone from resourcePath")
	a.NotContains(r.ResourcePath, "?")
	a.Equal("includeSystemFeatures=true", r.QueryString)

	require.NoError(t, r.processJSON([]byte(`{"name":".{0,32}"}`)))
	code, err := r.GenerateCode()
	require.NoError(t, err)

	// Well-formed: id segment precedes the query on the id-suffixed endpoints.
	a.Contains(code, "described-features/%s?includeSystemFeatures=true")
	a.NotContains(code, "?includeSystemFeatures=true/%s")

	// And the resource passes the raw-query guard cleanly.
	require.NoError(t, r.validateResourcePath())
}

// TestDescribedFeatureFromV2DefsURLsWellFormed builds DescribedFeature through the
// full offline pipeline (committed codegen/v2 defs + production customizations,
// rendered as a V2 resource exactly like generateCode does) and asserts every
// emitted URL is well-formed — the end-to-end ARCH-19 guarantee.
func TestDescribedFeatureFromV2DefsURLsWellFormed(t *testing.T) {
	t.Parallel()

	customizer, err := NewCodeCustomizer(defaultCustomizationsPath)
	require.NoError(t, err)

	resources, err := buildResourcesFromDownloadedFields("v2", *customizer, true, nil)
	require.NoError(t, err)

	var df *Resource
	for _, r := range resources {
		if r.StructName == "DescribedFeature" {
			df = r
			break
		}
	}
	require.NotNil(t, df, "DescribedFeature must be present in the v2 defs")

	assert.Equal(t, "described-features", df.ResourcePath, "raw '?' must be gone from resourcePath")
	assert.Equal(t, "includeSystemFeatures=true", df.QueryString)

	code, err := df.GenerateCode()
	require.NoError(t, err)

	for line := range strings.SplitSeq(code, "\n") {
		if strings.Contains(line, "?includeSystemFeatures=true/%s") {
			t.Fatalf("malformed URL emitted (query before id): %s", strings.TrimSpace(line))
		}
	}
	// The id-suffixed endpoints must keep the id before the query.
	assert.Contains(t, code, "described-features/%s?includeSystemFeatures=true")
}
