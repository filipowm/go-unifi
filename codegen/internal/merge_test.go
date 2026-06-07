package internal //nolint:testpackage // tests access unexported symbols

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFields stages a field-JSON file under dir; helper for the merge tests.
func writeFields(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)) //nolint:gosec
}

func resourceNames(resources []*Resource) []string {
	names := make([]string, 0, len(resources))
	for _, r := range resources {
		names = append(names, r.StructName)
	}
	sort.Strings(names)
	return names
}

func findResource(resources []*Resource, name string) *Resource {
	for _, r := range resources {
		if r.StructName == name {
			return r
		}
	}
	return nil
}

// TestBuildMergedResources_FloorBoundedUnion pins the two-snapshot merge: the
// result is the union of floor and newest keyed by struct name, newest wins for
// shared resources (newest field shapes), floor-only resources are kept, and a
// resource absent from both snapshots never appears (the retired-before-floor
// drop).
func TestBuildMergedResources_FloorBoundedUnion(t *testing.T) {
	t.Parallel()

	floor := t.TempDir()
	newest := t.TempDir()

	// Shared resource: newest adds a field the floor lacks -> newest wins.
	writeFields(t, floor, "Shared.json", `{"old_field": ".{0,32}"}`)
	writeFields(t, newest, "Shared.json", `{"old_field": ".{0,32}", "new_field": ".{0,32}"}`)
	// Floor-only resource (retired between floor and newest) -> kept.
	writeFields(t, floor, "FloorOnly.json", `{"name": ".{0,32}"}`)
	// Newest-only resource (added after the floor) -> kept.
	writeFields(t, newest, "NewestOnly.json", `{"name": ".{0,32}"}`)

	merged, err := buildMergedResources(floor, newest, CodeCustomizer{}, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"FloorOnly", "NewestOnly", "Shared"}, resourceNames(merged))

	// Shared takes the NEWEST shape: NewField is present (it exists only in newest).
	shared := findResource(merged, "Shared")
	require.NotNil(t, shared)
	assert.Contains(t, shared.BaseType().Fields, "NewField", "newest must win for shared resources")
}

// TestBuildMergedResources_FloorOnlyFieldDropped pins the production newest-wins
// semantics at field level: when a shared resource has a field ONLY in the floor
// (absent from newest), that field is dropped — newest's struct wins in full.
// This is the non-obvious behavior that drops the 17 floor-only fields in the
// real 9.0.114/9.5.21 merge; a regression to field-union would pass the basic
// FloorBoundedUnion test above but fail this one.
func TestBuildMergedResources_FloorOnlyFieldDropped(t *testing.T) {
	t.Parallel()

	floor := t.TempDir()
	newest := t.TempDir()

	// Shared resource: floor has an extra field that newest dropped -> newest wins,
	// floor_only_field must NOT appear in the merged resource.
	writeFields(t, floor, "Shared.json", `{"old_field": ".{0,32}", "floor_only_field": ".{0,32}"}`)
	writeFields(t, newest, "Shared.json", `{"old_field": ".{0,32}", "new_field": ".{0,32}"}`)

	merged, err := buildMergedResources(floor, newest, CodeCustomizer{}, nil)
	require.NoError(t, err)

	shared := findResource(merged, "Shared")
	require.NotNil(t, shared)
	assert.Contains(t, shared.BaseType().Fields, "NewField", "newest field must be present")
	assert.NotContains(t, shared.BaseType().Fields, "FloorOnlyField",
		"floor-only field must be dropped: newest struct wins in full (no field-level union)")
}

// TestBuildMergedResources_EmptyFloorIsSingleSnapshot pins that an empty floor
// disables the merge: generation proceeds from the newest snapshot alone, with
// no floor-bounding applied.
func TestBuildMergedResources_EmptyFloorIsSingleSnapshot(t *testing.T) {
	t.Parallel()

	newest := t.TempDir()
	writeFields(t, newest, "Only.json", `{"name": ".{0,32}"}`)

	merged, err := buildMergedResources("", newest, CodeCustomizer{}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"Only"}, resourceNames(merged))
}
