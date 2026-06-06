package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// asMap / asSlice / asStr are checked-assertion helpers keeping tests both
// lint-clean (forcetypeassert) and readable.
func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	require.True(t, ok)
	return m
}

func asSlice(t *testing.T, v any) []any {
	t.Helper()
	s, ok := v.([]any)
	require.True(t, ok)
	return s
}

func asStr(t *testing.T, v any) string {
	t.Helper()
	s, ok := v.(string)
	require.True(t, ok)
	return s
}

// docWith wraps schemas in a minimal 3.1 document.
func docWith(schemas map[string]any) map[string]any {
	return map[string]any{
		"openapi":    "3.1.0",
		"components": map[string]any{"schemas": schemas},
	}
}

func TestDownconvertRejects31Constructs(t *testing.T) {
	t.Parallel()
	cases := map[string]map[string]any{
		"type array": {"type": []any{"string", "null"}},
		"const":      {"const": "x"},
		"prefixItems": {"prefixItems": []any{
			map[string]any{"type": "string"},
		}},
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc := docWith(map[string]any{"Bad": bad})
			schemas, _ := schemasOf(doc)
			err := downconvert(doc, schemas)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "3.1-only")
		})
	}
}

func TestDownconvertSetsVersion(t *testing.T) {
	t.Parallel()
	doc := docWith(map[string]any{"Ok": map[string]any{"type": "object"}})
	schemas, _ := schemasOf(doc)
	require.NoError(t, downconvert(doc, schemas))
	assert.Equal(t, "3.0.3", doc["openapi"])
}

func TestAssertUpperSnakeMappings(t *testing.T) {
	t.Parallel()
	good := map[string]any{"P": map[string]any{
		"discriminator": map[string]any{"propertyName": "type", "mapping": map[string]any{"WIRED": "#/components/schemas/X", "WPA2_PERSONAL": "#/components/schemas/Y"}},
	}}
	require.NoError(t, assertUpperSnakeMappings(good))

	bad := map[string]any{"P": map[string]any{
		"discriminator": map[string]any{"propertyName": "type", "mapping": map[string]any{"Wired": "#/components/schemas/X"}},
	}}
	err := assertUpperSnakeMappings(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UPPER_SNAKE_CASE")
}

func TestSynthesizeOneOfMinesBothSources(t *testing.T) {
	t.Parallel()
	schemas := map[string]any{
		"Parent": map[string]any{
			"discriminator": map[string]any{
				"propertyName": "type",
				"mapping":      map[string]any{"A": "#/components/schemas/MappingOnly"},
			},
		},
		// Extends Parent via allOf but is absent from the mapping.
		"BackrefVariant": map[string]any{
			"allOf": []any{map[string]any{"$ref": "#/components/schemas/Parent"}},
		},
		"MappingOnly": map[string]any{"type": "object"},
	}
	synthesizeOneOf(schemas)
	oneOf := asSlice(t, asMap(t, schemas["Parent"])["oneOf"])
	got := []string{
		refName(asStr(t, asMap(t, oneOf[0])["$ref"])),
		refName(asStr(t, asMap(t, oneOf[1])["$ref"])),
	}
	// Sorted union of both sources.
	assert.Equal(t, []string{"BackrefVariant", "MappingOnly"}, got)
}

func TestFixDiamondsInlinesSecondParent(t *testing.T) {
	t.Parallel()
	schemas := map[string]any{
		"P1": map[string]any{
			"discriminator": map[string]any{"propertyName": "t"},
			"properties":    map[string]any{"t": map[string]any{"type": "string"}},
		},
		"P2": map[string]any{
			"discriminator": map[string]any{"propertyName": "t"},
			"properties":    map[string]any{"t": map[string]any{"type": "string"}, "extra": map[string]any{"type": "string"}},
		},
		"V": map[string]any{"allOf": []any{
			map[string]any{"$ref": "#/components/schemas/P1"},
			map[string]any{"$ref": "#/components/schemas/P2"},
		}},
	}
	require.NoError(t, fixDiamonds(schemas))
	allOf := asSlice(t, asMap(t, schemas["V"])["allOf"])

	// First parent kept as a $ref (sole surviving discriminator).
	assert.Equal(t, "#/components/schemas/P1", asMap(t, allOf[0])["$ref"])
	// Second parent inlined: no $ref, no discriminator, properties preserved.
	inl := asMap(t, allOf[1])
	_, hasRef := inl["$ref"]
	assert.False(t, hasRef)
	_, hasDisc := inl["discriminator"]
	assert.False(t, hasDisc)
	assert.Contains(t, asMap(t, inl["properties"]), "extra")
}

func TestFixDiamondsRejectsAllOfBase(t *testing.T) {
	t.Parallel()
	schemas := map[string]any{
		"P1": map[string]any{"discriminator": map[string]any{"propertyName": "t"}},
		"P2": map[string]any{"discriminator": map[string]any{"propertyName": "t"}, "allOf": []any{}},
		"V": map[string]any{"allOf": []any{
			map[string]any{"$ref": "#/components/schemas/P1"},
			map[string]any{"$ref": "#/components/schemas/P2"},
		}},
	}
	err := fixDiamonds(schemas)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "own allOf")
}

func TestAssertNoOneOfCyclesDetectsLoop(t *testing.T) {
	t.Parallel()
	schemas := map[string]any{
		"A": map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/B"}}},
		"B": map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/A"}}},
	}
	err := assertNoOneOfCycles(schemas)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular oneOf")
}

func TestAssertNoOneOfCyclesAcceptsTree(t *testing.T) {
	t.Parallel()
	schemas := map[string]any{
		"Parent": map[string]any{"oneOf": []any{
			map[string]any{"$ref": "#/components/schemas/V1"},
			map[string]any{"$ref": "#/components/schemas/V2"},
		}},
		"V1": map[string]any{"type": "object"},
		"V2": map[string]any{"type": "object"},
	}
	require.NoError(t, assertNoOneOfCycles(schemas))
}

func TestScan31IgnoresValidSchema(t *testing.T) {
	t.Parallel()
	var found []string
	scan31("root", map[string]any{
		"type":       "object",
		"properties": map[string]any{"x": map[string]any{"type": "string", "enum": []any{"A"}}},
	}, &found)
	assert.Empty(t, found)
}

func TestDeepCopyIsIndependent(t *testing.T) {
	t.Parallel()
	src := map[string]any{"a": []any{map[string]any{"b": 1}}}
	cp := asMap(t, deepCopy(src))
	asMap(t, asSlice(t, cp["a"])[0])["b"] = 2
	assert.Equal(t, 1, asMap(t, asSlice(t, src["a"])[0])["b"])
}
