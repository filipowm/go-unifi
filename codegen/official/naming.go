package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
)

const (
	// extGoType makes oapi-codegen reuse an existing Go type instead of emitting one.
	extGoType = "x-go-type"
	// Hand-written models in package official that the spec twins must defer to.
	appInfoTypeFinal  = "ApplicationInfo"
	siteOverviewFinal = "SiteOverview"
	handWrittenInfo   = "Info"
)

// nameOverrides force a specific Go type name where the default normalization
// would collide. "IP Address selector" (a variant) would normalize identically
// to its parent "IP address selector"; the spec's own "specific" idiom resolves it.
var nameOverrides = map[string]string{
	"IP Address selector": "SpecificIPAddressSelector",
}

// buildRenameMap computes the final Go type name for every schema key and fails
// loudly if two schemas still collide after overrides — the collision set is
// recomputed on the POST-transform schema set (synthesized members included).
func buildRenameMap(schemas map[string]any) (map[string]string, error) {
	renames := make(map[string]string, len(schemas))
	collisions := map[string][]string{}
	for key := range schemas {
		final := finalName(key)
		renames[key] = final
		collisions[final] = append(collisions[final], key)
	}
	var clashes []string
	for final, keys := range collisions {
		if len(keys) > 1 {
			sort.Strings(keys)
			clashes = append(clashes, fmt.Sprintf("%s <= %v", final, keys))
		}
	}
	if len(clashes) > 0 {
		sort.Strings(clashes)
		return nil, fmt.Errorf("unresolved type-name collisions: %s", strings.Join(clashes, "; "))
	}
	return renames, nil
}

// finalName maps a spec schema name to its Go type name: explicit override, else
// strip the Integration/Dto affixes, flip the "Create or update X" verb phrase,
// then apply oapi-codegen's own camel-casing so our name matches what it emits.
func finalName(key string) string {
	if o, ok := nameOverrides[key]; ok {
		return o
	}
	n := key
	if rest, ok := strings.CutPrefix(n, "Create or update "); ok {
		n = rest + " create or update"
	}
	n = strings.TrimPrefix(n, "Integration")
	n = strings.TrimSuffix(n, "Dto")
	return codegen.ToCamelCase(n)
}

// applyRenames rewrites schema keys plus every $ref and discriminator.mapping
// value across the document to the final names.
func applyRenames(doc map[string]any, renames map[string]string) {
	comps, _ := doc["components"].(map[string]any)
	old, _ := comps["schemas"].(map[string]any)
	renamed := make(map[string]any, len(old))
	for k, v := range old {
		renamed[renames[k]] = v
	}
	comps["schemas"] = renamed
	rewriteRefs(doc, renames)
}

// rewriteRefs walks the document rewriting component-schema $ref pointers and
// discriminator.mapping targets in place.
func rewriteRefs(node any, renames map[string]string) {
	switch n := node.(type) {
	case map[string]any:
		if ref, ok := n["$ref"].(string); ok {
			n["$ref"] = rewriteRef(ref, renames)
		}
		if disc, ok := n["discriminator"].(map[string]any); ok {
			if mapping, ok := disc["mapping"].(map[string]any); ok {
				for k, v := range mapping {
					if s, ok := v.(string); ok {
						mapping[k] = rewriteRef(s, renames)
					}
				}
			}
		}
		for _, v := range n {
			rewriteRefs(v, renames)
		}
	case []any:
		for _, v := range n {
			rewriteRefs(v, renames)
		}
	}
}

// rewriteRef maps a single #/components/schemas ref to its renamed target.
func rewriteRef(ref string, renames map[string]string) string {
	name, ok := strings.CutPrefix(ref, schemaRefPrefix)
	if !ok {
		return ref
	}
	if to, ok := renames[name]; ok {
		return schemaRefPrefix + to
	}
	return ref
}

// enumTarget is a property whose inline enum is replaced by a shared type.
type enumTarget struct{ schema, property string }

// sharedEnum hoists one enum value-set into a single named type reused across a
// tri-shape family, collapsing oapi-codegen's otherwise-duplicated per-schema
// enums (e.g. the ACL action triplet) into one public type.
type sharedEnum struct {
	name    string
	values  []string
	targets []enumTarget
}

// sharedEnums is the enum-dedup table. Only same-family value-sets are merged:
// identical value-sets that recur across UNRELATED families (e.g. ALLOW/BLOCK on
// Wi-Fi client filtering, or the protocol-name set across IPv4/IPv6) are left
// distinct on purpose — one shared name there would be semantically misleading.
var sharedEnums = []sharedEnum{
	{
		name:   "ACLRuleAction",
		values: []string{"ALLOW", "BLOCK"},
		targets: []enumTarget{
			{"ACL rule", "action"},
			{"ACL rule update", "action"},
			{"ACL ruleObject", "action"},
		},
	},
}

// dedupeEnums hoists each shared enum into its own component and repoints the
// target properties at it, validating the value-sets match (fail loud otherwise).
func dedupeEnums(schemas map[string]any) error {
	for _, se := range sharedEnums {
		if _, exists := schemas[se.name]; exists {
			return fmt.Errorf("shared enum %q already exists as a schema", se.name)
		}
		for _, t := range se.targets {
			prop, err := enumProperty(schemas, t)
			if err != nil {
				return err
			}
			if got := enumValues(prop); !equalStrings(got, se.values) {
				return fmt.Errorf("enum dedup %s: %s.%s has values %v, want %v", se.name, t.schema, t.property, got, se.values)
			}
		}
		enumVals := make([]any, len(se.values))
		for i, v := range se.values {
			enumVals[i] = v
		}
		schemas[se.name] = map[string]any{"type": "string", "enum": enumVals}
		for _, t := range se.targets {
			// Validated above by enumProperty, so the asserts are safe.
			s, _ := schemas[t.schema].(map[string]any)
			props, _ := s["properties"].(map[string]any)
			props[t.property] = map[string]any{"$ref": schemaRefPrefix + se.name}
		}
	}
	return nil
}

// enumProperty resolves a target's property schema, failing loudly if absent.
func enumProperty(schemas map[string]any, t enumTarget) (map[string]any, error) {
	s, ok := schemas[t.schema].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("enum dedup: schema %q not found", t.schema)
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("enum dedup: schema %q has no properties", t.schema)
	}
	prop, ok := props[t.property].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("enum dedup: %q has no property %q", t.schema, t.property)
	}
	return prop, nil
}

// enumValues returns a property's enum values sorted for comparison.
func enumValues(prop map[string]any) []string {
	raw, _ := prop["enum"].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// equalStrings reports whether two string slices are equal (both pre-sorted).
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
