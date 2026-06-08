package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateTagOf injects tags into a one-property schema and returns the emitted
// validate string (empty when none), so tests assert the exact user-facing tag.
func validateTagOf(t *testing.T, prop map[string]any) string {
	t.Helper()
	schemas := map[string]any{"S": map[string]any{
		"type":       "object",
		"properties": map[string]any{"p": prop},
	}}
	require.NoError(t, injectValidationTags(schemas))
	extra, ok := prop[extExtraTags].(map[string]any)
	if !ok {
		return ""
	}
	return asStr(t, extra["validate"])
}

func TestValidationTagInlineScalar(t *testing.T) {
	t.Parallel()
	// minLength/maxLength -> min/max on a string, omitempty-led.
	got := validateTagOf(t, map[string]any{"type": "string", "minLength": float64(1), "maxLength": float64(127)})
	assert.Equal(t, "omitempty,min=1,max=127", got)
}

func TestValidationTagNumericRange(t *testing.T) {
	t.Parallel()
	// minimum/maximum -> gte/lte, large bound stays plain decimal (no exponent).
	got := validateTagOf(t, map[string]any{"type": "integer", "minimum": float64(0), "maximum": float64(31536000)})
	assert.Equal(t, "omitempty,gte=0,lte=31536000", got)
}

func TestValidationTagIntEnumWithStringValues(t *testing.T) {
	t.Parallel()
	// The integer-typed-property-with-string-enum-values quirk: values pass through.
	got := validateTagOf(t, map[string]any{
		"type": "integer",
		"enum": []any{"1000", "2000", "5500"},
	})
	assert.Equal(t, "omitempty,oneof=1000 2000 5500", got)
}

func TestValidationTagArrayWithMinItems(t *testing.T) {
	t.Parallel()
	got := validateTagOf(t, map[string]any{
		"type":     "array",
		"minItems": float64(1),
		"items":    map[string]any{"type": "string"},
	})
	assert.Equal(t, "omitempty,min=1", got)
}

func TestValidationTagArrayOfEnumDives(t *testing.T) {
	t.Parallel()
	// minItems then dive into the items' oneof; number enum keeps plain decimals.
	got := validateTagOf(t, map[string]any{
		"type":     "array",
		"minItems": float64(1),
		"items":    map[string]any{"type": "number", "enum": []any{2.4, float64(5), float64(6)}},
	})
	assert.Equal(t, "omitempty,min=1,dive,oneof=2.4 5 6", got)
}

func TestValidationTagMaxItemsSentinelFiltered(t *testing.T) {
	t.Parallel()
	// The math.MaxInt32 "unbounded" sentinel must not become a max= rule.
	got := validateTagOf(t, map[string]any{
		"type":     "array",
		"minItems": float64(1),
		"maxItems": float64(maxItemsSentinel),
		"items":    map[string]any{"type": "string"},
	})
	assert.Equal(t, "omitempty,min=1", got)
}

func TestValidationTagAllOfNestedField(t *testing.T) {
	t.Parallel()
	// ~36% of constraints live under allOf[1]; propertyBags must reach them.
	schemas := map[string]any{"S": map[string]any{
		"allOf": []any{
			map[string]any{"$ref": "#/components/schemas/Base"},
			map[string]any{"properties": map[string]any{
				"ttlSeconds": map[string]any{"type": "integer", "minimum": float64(0), "maximum": float64(86400)},
			}},
		},
	}}
	require.NoError(t, injectValidationTags(schemas))
	member := asMap(t, asSlice(t, asMap(t, schemas["S"])["allOf"])[1])
	prop := asMap(t, asMap(t, member["properties"])["ttlSeconds"])
	extra := asMap(t, prop[extExtraTags])
	assert.Equal(t, "omitempty,gte=0,lte=86400", asStr(t, extra["validate"]))
}

func TestValidationTagFailsOnSpaceEnum(t *testing.T) {
	t.Parallel()
	schemas := map[string]any{"S": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"p": map[string]any{"type": "string", "enum": []any{"OK", "BAD VALUE"}},
		},
	}}
	err := injectValidationTags(schemas)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains a space")
}

func TestValidationRequiredAndFormatNotEmitted(t *testing.T) {
	t.Parallel()
	// required is structural, not value-presence; format maps to a Go type. Neither
	// should leak into the validate tag.
	got := validateTagOf(t, map[string]any{"type": "string", "format": "uuid"})
	assert.Empty(t, got)
}
