package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	snapshotDir   = "../openapi"
	committedFile = "../../unifi/official/models.generated.go"
)

// loadSnapshot reads the committed Official OpenAPI snapshot bytes.
func loadSnapshot(t *testing.T) []byte {
	t.Helper()
	path, err := ResolveSnapshot(snapshotDir)
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
	path, err := ResolveSnapshot(snapshotDir)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, ".json"))

	_, err = ResolveSnapshot(t.TempDir())
	require.Error(t, err)
}
