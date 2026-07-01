package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	snapshotDir   = "../openapi"
	committedFile = "../../unifi/official/models.generated.go"
)

// loadSnapshot reads the committed Official OpenAPI snapshot bytes for the pinned
// version — the same resolution the standalone frontend uses, so the bytes match
// the committed generated surface even when multiple snapshots are committed.
func loadSnapshot(t *testing.T) []byte {
	t.Helper()
	path, err := ResolveSnapshot(snapshotDir, resolveSnapshotVersion(snapshotDir, ""))
	require.NoError(t, err)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return raw
}

// NOTE: the three tests below call GenerateModels, which invokes oapi-codegen's
// codegen.Generate — NOT concurrency-safe (it mutates a package-global map).
// They MUST run serially; do not add t.Parallel() (the paralleltest linter is
// off and won't catch a re-add).

// TestGenerateModelsDeterministic proves generation is byte-identical on re-run
// (no map-iteration ordering leaks into the output).
func TestGenerateModelsDeterministic(t *testing.T) {
	raw := loadSnapshot(t)
	a, err := GenerateModels(raw, defaultPackageName)
	require.NoError(t, err)
	b, err := GenerateModels(raw, defaultPackageName)
	require.NoError(t, err)
	assert.Equal(t, a, b, "two generations of the same snapshot must be identical")
}

// TestModelsMatchCommitted asserts the committed models.generated.go is exactly
// what the snapshot produces today — the in-repo mirror of the determinism gate.
func TestModelsMatchCommitted(t *testing.T) {
	raw := loadSnapshot(t)
	got, err := GenerateModels(raw, defaultPackageName)
	require.NoError(t, err)
	want, err := os.ReadFile(committedFile)
	require.NoError(t, err)
	if string(want) != got {
		t.Fatalf("committed %s is stale; re-run `go run .` in codegen/official", committedFile)
	}
}

// TestGeneratedSurface spot-checks the transform's headline guarantees on the
// real spec: DO NOT EDIT header, no leaked Integration/Dto names, deduped ACL
// action enum, and the empty-variant alias.
func TestGeneratedSurface(t *testing.T) {
	code, err := GenerateModels(loadSnapshot(t), defaultPackageName)
	require.NoError(t, err)

	// Placeholder godoc rewrite: oapi-codegen zero-info phrases must be gone.
	assert.NotContains(t, code, "defines model for")
	assert.NotContains(t, code, "defines parameters for")
	assert.NotContains(t, code, "defines body for")
	// Replacement phrases must be present.
	assert.Contains(t, code, "is a generated model for the UniFi Official API.")
	assert.Contains(t, code, "holds query parameters for the UniFi Official API.")
	assert.Contains(t, code, "is a generated request body for the UniFi Official API.")
	// Spec-supplied uppercase "Defines values for" docs must survive the rewrite — the regex
	// must never be broadened to consume them.
	assert.GreaterOrEqual(t, 50, strings.Count(code, "// Defines values for "), "spec-supplied enum docs must be preserved exactly")

	assert.Contains(t, code, "DO NOT EDIT.")
	assert.Contains(t, code, "package official")
	assert.NotContains(t, code, "type Integration")
	assert.NotContains(t, code, "Dto struct")
	assert.Contains(t, code, "type ACLRuleAction string")
	assert.NotContains(t, code, "ACLRuleObjectAction")
	// Detail vs create-or-update enums collapsed onto the detail-shape names.
	for _, gone := range []string{
		"FirewallPolicyCreateOrUpdateConnectionStateFilter",
		"FirewallPolicyCreateOrUpdateIpsecFilter",
		"IpAclRuleCreateUpdateProtocolFilter",
	} {
		assert.NotContains(t, code, gone)
	}
	assert.Contains(t, code, "type FirewallPolicyConnectionStateFilter string")
	assert.Contains(t, code, "type FirewallPolicyIpsecFilter string")
	assert.Contains(t, code, "type IpAclRuleProtocolFilter string")
	assert.Contains(t, code, "type FirewallPolicyActionBlock = FirewallPolicyAction")
	// Hand-written collisions are deferred to the hand-written siblings.
	assert.NotContains(t, code, "type SiteOverview struct")
	assert.Contains(t, code, "type ApplicationInfo = Info")
}

func TestResolveSnapshot(t *testing.T) {
	t.Parallel()

	// Committed dir: empty version resolves to the pinned/newest snapshot.
	path, err := ResolveSnapshot(snapshotDir, "")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, ".json"))

	// Empty dir: no snapshots is an error.
	_, err = ResolveSnapshot(t.TempDir(), "")
	require.Error(t, err)
}

// TestResolveSnapshotMultiple proves multiple committed snapshots coexist: an
// empty version selects the newest by numeric order (not lexicographic), and an
// explicit version selects exactly that one.
func TestResolveSnapshotMultiple(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, v := range []string{"10.1.78", "10.1.85", "9.5.21", "10.1.9"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "integration-"+v+".json"), []byte("{}"), 0o644))
	}

	// Empty version -> newest by numeric version (10.1.85, not 10.1.9 or 9.x).
	latest, err := ResolveSnapshot(dir, "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "integration-10.1.85.json"), latest)

	// Explicit version -> that exact snapshot, even when it is not the newest.
	pinned, err := ResolveSnapshot(dir, "10.1.78")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "integration-10.1.78.json"), pinned)

	// Explicit version with no committed snapshot fails loudly, listing what exists.
	_, err = ResolveSnapshot(dir, "10.2.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "10.1.85")
}

// TestResolveSnapshotVersionPin proves the standalone default follows the
// .unifi-version-official pin when one is present, so regeneration reproduces the
// committed surface rather than silently jumping to a newer committed snapshot.
func TestResolveSnapshotVersionPin(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	specDir := filepath.Join(root, "openapi")
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	for _, v := range []string{"10.1.78", "10.1.85"} {
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "integration-"+v+".json"), []byte("{}"), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, ".unifi-version-official"), []byte("10.1.78\n"), 0o644))

	// Pin present: default resolution picks the pinned version, not the newest.
	assert.Equal(t, "10.1.78", resolveSnapshotVersion(specDir, ""))
	// Explicit flag always wins over the pin.
	assert.Equal(t, "10.1.85", resolveSnapshotVersion(specDir, "10.1.85"))
}
