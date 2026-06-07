package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// committedDir holds the generated Official surface files.
const committedDir = "../../unifi/official"

// loadOps parses the committed snapshot into the generator's operation view.
func loadOps(t *testing.T) []operation {
	t.Helper()
	var doc map[string]any
	require.NoError(t, json.Unmarshal(loadSnapshot(t), &doc))
	ops, err := buildOperations(doc)
	require.NoError(t, err)
	return ops
}

// findOp returns the operation with the given Go method name.
func findOp(t *testing.T, ops []operation, name string) operation {
	t.Helper()
	for _, o := range ops {
		if o.Name == name {
			return o
		}
	}
	t.Fatalf("operation %q not found", name)
	return operation{}
}

// TestSurfaceMatchesCommitted byte-guards every generated surface file against
// the committed copy — the in-repo mirror of the determinism gate.
func TestSurfaceMatchesCommitted(t *testing.T) {
	files, err := GenerateSurfaceFiles(loadSnapshot(t), defaultPackageName)
	require.NoError(t, err)
	for _, f := range files {
		want, err := os.ReadFile(filepath.Join(committedDir, f.name))
		require.NoError(t, err)
		if string(want) != f.code {
			t.Fatalf("committed %s is stale; re-run `go run .` in codegen/official", f.name)
		}
	}
}

// TestSurfaceDeterministic proves generation is byte-identical on re-run.
func TestSurfaceDeterministic(t *testing.T) {
	raw := loadSnapshot(t)
	a, err := GenerateSurfaceFiles(raw, defaultPackageName)
	require.NoError(t, err)
	b, err := GenerateSurfaceFiles(raw, defaultPackageName)
	require.NoError(t, err)
	require.Equal(t, a, b, "two generations of the same snapshot must be identical")
}

// TestHandWrittenOperationsSkipped asserts info/sites operations carry no
// generated wrapper (their hand-written siblings own them).
func TestHandWrittenOperationsSkipped(t *testing.T) {
	for _, o := range loadOps(t) {
		assert.NotEqual(t, "GetInfo", o.Name)
		assert.NotEqual(t, "GetSiteOverviewPage", o.Name)
	}
}

// TestNonCRUDClassification spot-checks the headline non-CRUD shapes: list
// pagination, PATCH, required-filter bulk delete, ordering query params,
// references and statistics.
func TestNonCRUDClassification(t *testing.T) {
	ops := loadOps(t)

	list := findOp(t, ops, "GetNetworksOverviewPage")
	assert.True(t, list.IsList())
	assert.Equal(t, "NetworkOverview", list.ItemType)

	patch := findOp(t, ops, "PatchFirewallPolicy")
	assert.Equal(t, "PATCH", patch.HTTPMethod)
	assert.Equal(t, "PatchFirewallPolicy", patch.BodyType)
	assert.Equal(t, "FirewallPolicy", patch.ReturnType)

	del := findOp(t, ops, "DeleteVouchers")
	assert.Equal(t, "filter", del.RequiredFilter())
	assert.Equal(t, "VoucherDeletionResults", del.ReturnType)

	ordering := findOp(t, ops, "GetFirewallPolicyOrdering")
	require.Len(t, ordering.QueryArgs, 2)
	assert.Equal(t, "sourceFirewallZoneId", ordering.QueryArgs[0].Name)

	refs := findOp(t, ops, "GetNetworkReferences")
	assert.False(t, refs.IsList())
	assert.Equal(t, "NetworkReferences", refs.ReturnType)

	action := findOp(t, ops, "ExecutePortAction")
	assert.Empty(t, action.ReturnType)
	assert.Equal(t, "PortActionRequest", action.BodyType)
	require.Len(t, action.PathArgs, 3) // siteId, deviceId, portIdx in URL order
	assert.Equal(t, "portIdx", action.PathArgs[2].Name)
}
