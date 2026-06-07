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
	customInfo        = "Info"
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
	// ACLRule and ACLRuleObject are independent spec schemas that currently share
	// all variants (IpAclRule/MacAclRule) and this one action enum, yet are kept as
	// distinct generated types on purpose: aliasing them would turn a future spec
	// divergence into a silent breaking change. Only the enum is deduped here.
	{
		name:   "ACLRuleAction",
		values: []string{"ALLOW", "BLOCK"},
		targets: []enumTarget{
			{"ACL rule", "action"},
			{"ACL rule update", "action"},
			{"ACL ruleObject", "action"},
		},
	},
	// Detail vs create-or-update shapes of one resource carry byte-identical
	// value-sets; collapse each onto the detail-shape name (unambiguous in-family).
	{
		name:   "FirewallPolicyConnectionStateFilter",
		values: []string{"ESTABLISHED", "INVALID", "NEW", "RELATED"},
		targets: []enumTarget{
			{"Firewall policy", "connectionStateFilter"},
			{"Create or update firewall policy", "connectionStateFilter"},
		},
	},
	{
		name:   "FirewallPolicyIpsecFilter",
		values: []string{"MATCH_ENCRYPTED", "MATCH_NOT_ENCRYPTED"},
		targets: []enumTarget{
			{"Firewall policy", "ipsecFilter"},
			{"Create or update firewall policy", "ipsecFilter"},
		},
	},
	{
		name:   "IpAclRuleProtocolFilter",
		values: []string{"TCP", "UDP"},
		targets: []enumTarget{
			{"IntegrationIpAclRuleDto", "protocolFilter"},
			{"IntegrationIpAclRuleCreateUpdateDto", "protocolFilter"},
		},
	},
}

// dedupeEnums hoists each shared enum into its own component and repoints the
// target properties at it, validating the value-sets match (fail loud otherwise).
func dedupeEnums(schemas map[string]any) error {
	return dedupeEnumsWith(schemas, sharedEnums)
}

// dedupeEnumsWith is the table-driven core; the table is a parameter so tests can
// exercise one entry without mutating the package-global sharedEnums.
func dedupeEnumsWith(schemas map[string]any, table []sharedEnum) error {
	for _, se := range table {
		if _, exists := schemas[se.name]; exists {
			return fmt.Errorf("shared enum %q already exists as a schema", se.name)
		}
		for _, t := range se.targets {
			_, prop, err := enumProperty(schemas, t)
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
			// Validated above by enumProperty, so the lookups are safe.
			props, prop, _ := enumProperty(schemas, t)
			if node, isArray := enumNode(prop); isArray {
				prop["items"] = enumRef(node, se.name) // repoint items, keep array constraints
			} else {
				props[t.property] = enumRef(node, se.name)
			}
		}
	}
	return nil
}

// enumProperty resolves a target's property, searching the schema's own
// properties and any contributed via inline allOf members (e.g. IpAclRule).
// Returns the containing properties bag and the property, failing loudly if absent.
func enumProperty(schemas map[string]any, t enumTarget) (map[string]any, map[string]any, error) {
	s, ok := schemas[t.schema].(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("enum dedup: schema %q not found", t.schema)
	}
	bags := propertyBags(s)
	if len(bags) == 0 {
		return nil, nil, fmt.Errorf("enum dedup: schema %q has no properties", t.schema)
	}
	for _, props := range bags {
		if prop, ok := props[t.property].(map[string]any); ok {
			return props, prop, nil
		}
	}
	return nil, nil, fmt.Errorf("enum dedup: %q has no property %q", t.schema, t.property)
}

// propertyBags returns every properties map reachable from a schema: its own
// plus those contributed by inline allOf members.
func propertyBags(s map[string]any) []map[string]any {
	var bags []map[string]any
	if p, ok := s["properties"].(map[string]any); ok {
		bags = append(bags, p)
	}
	if allOf, ok := s["allOf"].([]any); ok {
		for _, m := range allOf {
			mm, ok := m.(map[string]any)
			if !ok {
				continue
			}
			if p, ok := mm["properties"].(map[string]any); ok {
				bags = append(bags, p)
			}
		}
	}
	return bags
}

// enumRef points an enum node at the shared type while preserving any sibling
// metadata (description, example, x-*) it carried, so dedup never drops it. A
// bare $ref in OpenAPI 3.0 ignores siblings, so surviving keys are wrapped in a
// single-member allOf; a node with only type/enum yields a plain $ref.
func enumRef(node map[string]any, name string) map[string]any {
	ref := map[string]any{"$ref": schemaRefPrefix + name}
	rest := map[string]any{}
	for k, v := range node {
		if k == "type" || k == "enum" {
			continue
		}
		rest[k] = v
	}
	if len(rest) == 0 {
		return ref
	}
	rest["allOf"] = []any{ref}
	return rest
}

// enumNode returns the map carrying the enum keyword and whether it is an array
// element enum: a scalar enum lives on the property, an array's on its items.
func enumNode(prop map[string]any) (map[string]any, bool) {
	if prop["type"] == "array" {
		if items, ok := prop["items"].(map[string]any); ok {
			return items, true
		}
	}
	return prop, false
}

// enumValues returns a property's enum values sorted for comparison, reading
// from items for an array-of-enum property.
func enumValues(prop map[string]any) []string {
	node, _ := enumNode(prop)
	raw, _ := node["enum"].([]any)
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
