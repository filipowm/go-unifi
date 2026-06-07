package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// schemaRefPrefix is the JSON-pointer prefix every component-schema $ref carries.
const schemaRefPrefix = "#/components/schemas/"

// upperSnake matches a discriminator wire value (the controller's enum keys).
// Every discriminator.mapping key MUST match it so the generated
// ValueByDiscriminator() switch decodes real payloads.
var upperSnake = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Transform bridges the spec into oapi-codegen's oneOf code path, which emits
// per-variant union structs (the allOf+discriminator path silently drops them).
// It runs generically on the RAW JSON so it survives controller spec bumps, with
// naming/rename applied last; refs stay keyed by spec names until then. Returns
// the schema names oapi-codegen must NOT emit because a hand-written sibling in
// package official already defines them.
func Transform(doc map[string]any) ([]string, error) {
	schemas, err := schemasOf(doc)
	if err != nil {
		return nil, err
	}
	if err := downconvert(doc); err != nil {
		return nil, fmt.Errorf("downconverting OpenAPI document: %w", err)
	}
	if err := assertUpperSnakeMappings(schemas); err != nil {
		return nil, err
	}
	if err := dedupeEnums(schemas); err != nil {
		return nil, err
	}
	if err := fixDiamonds(schemas); err != nil {
		return nil, err
	}
	synthesizeOneOf(schemas)

	renames, err := buildRenameMap(schemas)
	if err != nil {
		return nil, err
	}
	applyRenames(doc, renames)
	// applyRenames swaps in a fresh, renamed schemas map; re-fetch so the
	// post-rename steps below mutate the live document, not the orphaned old map.
	schemas, _ = schemasOf(doc)
	if err := assertNoOneOfCycles(schemas); err != nil {
		return nil, err
	}

	// Hand-written collisions: package official already declares these models, so
	// alias / suppress generation of the spec twins (Stage 3 reconciles).
	if info, ok := schemas[appInfoTypeFinal].(map[string]any); ok {
		info[extGoType] = customInfo
	}
	return []string{siteOverviewFinal}, nil
}

// schemasOf returns components.schemas, failing loudly when the spec shape is
// not what every downstream step assumes.
func schemasOf(doc map[string]any) (map[string]any, error) {
	comps, ok := doc["components"].(map[string]any)
	if !ok {
		return nil, errors.New("spec has no components object")
	}
	schemas, ok := comps["schemas"].(map[string]any)
	if !ok {
		return nil, errors.New("spec has no components.schemas object")
	}
	return schemas, nil
}

// fields31Only are JSON-Schema keywords that exist only in OpenAPI 3.1; the
// downconvert is lossless precisely because the spec uses none of them.
var fields31Only = []string{
	"prefixItems", "const", "unevaluatedProperties", "unevaluatedItems",
	"$dynamicRef", "$dynamicAnchor", "if", "then", "else",
	"dependentSchemas", "dependentRequired", "patternProperties", "contentMediaType",
}

// downconvert rewrites the OpenAPI version 3.1.0 -> 3.0.3. It first asserts the
// spec carries no 3.1-exclusive construct so the version bump cannot lose data.
// The scan walks the WHOLE document (paths, parameters, requestBodies, responses,
// etc.), not just components.schemas, so no 3.1-only construct escapes the guard.
func downconvert(doc map[string]any) error {
	ver, _ := doc["openapi"].(string)
	if !strings.HasPrefix(ver, "3.1") {
		return fmt.Errorf("expected OpenAPI 3.1.x spec, got %q", ver)
	}
	var found []string
	scan31("root", doc, &found)
	if len(found) > 0 {
		sort.Strings(found)
		return fmt.Errorf("spec contains OpenAPI 3.1-only constructs, downconvert would be lossy: %s", strings.Join(found, ", "))
	}
	doc["openapi"] = "3.0.3"
	return nil
}

// scan31 recurses a schema subtree collecting any 3.1-only keyword usage.
func scan31(path string, node any, found *[]string) {
	switch n := node.(type) {
	case map[string]any:
		scan31Node(path, n, found)
		for k, v := range n {
			scan31(path+"/"+k, v, found)
		}
	case []any:
		for _, v := range n {
			scan31(path, v, found)
		}
	}
}

// scan31Node flags 3.1-only constructs carried directly on one schema node.
func scan31Node(path string, n map[string]any, found *[]string) {
	if t, ok := n["type"]; ok {
		if _, isArr := t.([]any); isArr {
			*found = append(*found, path+".type[]")
		}
		if t == "null" { // bare null type is 3.1-only (3.0 has no null type)
			*found = append(*found, path+".type=null")
		}
	}
	// exclusiveMinimum/Maximum flipped from boolean (3.0) to numeric (3.1);
	// a numeric value would be silently mis-downconverted.
	for _, kw := range []string{"exclusiveMinimum", "exclusiveMaximum"} {
		if v, ok := n[kw]; ok && isNumeric(v) {
			*found = append(*found, path+"."+kw)
		}
	}
	for _, kw := range fields31Only {
		if _, ok := n[kw]; ok {
			*found = append(*found, path+"."+kw)
		}
	}
}

// isNumeric reports whether v is a JSON number (float64 from encoding/json;
// ints accepted so hand-built test schemas match the real decode path too).
func isNumeric(v any) bool {
	switch v.(type) {
	case float64, float32, int, int64:
		return true
	default:
		return false
	}
}

// discriminatorOf returns a schema's discriminator object, or nil.
func discriminatorOf(schema any) map[string]any {
	sm, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	disc, _ := sm["discriminator"].(map[string]any)
	return disc
}

// assertUpperSnakeMappings guarantees every discriminator.mapping key is the
// UPPER_SNAKE wire value the controller actually sends.
func assertUpperSnakeMappings(schemas map[string]any) error {
	var bad []string
	for name, s := range schemas {
		disc := discriminatorOf(s)
		if disc == nil {
			continue
		}
		mapping, _ := disc["mapping"].(map[string]any)
		for k := range mapping {
			if !upperSnake.MatchString(k) {
				bad = append(bad, fmt.Sprintf("%s.mapping[%q]", name, k))
			}
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return fmt.Errorf("discriminator mapping keys must be UPPER_SNAKE_CASE: %s", strings.Join(bad, ", "))
	}
	return nil
}

// refName returns the trailing component name of a #/components/schemas ref.
func refName(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// discParentRefs returns the allOf members of schema that $ref a discriminator
// parent, in source order, paired with their index.
func discParentRefs(schema map[string]any, schemas map[string]any) []int {
	allOf, ok := schema["allOf"].([]any)
	if !ok {
		return nil
	}
	var idx []int
	for i, m := range allOf {
		mm, _ := m.(map[string]any)
		ref, _ := mm["$ref"].(string)
		if ref == "" {
			continue
		}
		if discriminatorOf(schemas[refName(ref)]) != nil {
			idx = append(idx, i)
		}
	}
	return idx
}

// fixDiamonds resolves schemas whose allOf extends 2+ discriminator parents.
// oapi-codegen refuses to merge two discriminators, so we keep the first parent
// ref and inline the others' properties (minus their discriminator) — lossless
// because those bases contribute fields, not a second union.
func fixDiamonds(schemas map[string]any) error {
	for _, s := range schemas {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		idx := discParentRefs(sm, schemas)
		if len(idx) < 2 {
			continue
		}
		allOf, _ := sm["allOf"].([]any)
		for _, i := range idx[1:] {
			member, _ := allOf[i].(map[string]any)
			ref, _ := member["$ref"].(string)
			parent, _ := schemas[refName(ref)].(map[string]any)
			if _, nested := parent["allOf"]; nested {
				return fmt.Errorf("diamond base %q has its own allOf; inlining would be lossy", refName(ref))
			}
			allOf[i] = inlineWithoutDiscriminator(parent)
		}
	}
	return nil
}

// inlineWithoutDiscriminator copies a parent's object fields except its
// discriminator, so the copy merges into a variant without a second union.
func inlineWithoutDiscriminator(parent map[string]any) map[string]any {
	out := map[string]any{"type": "object"}
	if props, ok := parent["properties"]; ok {
		out["properties"] = deepCopy(props)
	}
	if req, ok := parent["required"]; ok {
		out["required"] = deepCopy(req)
	}
	return out
}

// synthesizeOneOf rewrites every discriminator parent to a oneOf over ALL its
// variants. Variants are mined from BOTH allOf back-references and the
// discriminator.mapping (some variants only appear in the mapping), and the
// member list is sorted for deterministic output.
func synthesizeOneOf(schemas map[string]any) {
	backref := allOfBackrefs(schemas)
	for name, s := range schemas {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		disc := discriminatorOf(sm)
		if disc == nil {
			continue
		}
		variants := map[string]bool{}
		if mapping, ok := disc["mapping"].(map[string]any); ok {
			for _, v := range mapping {
				if ref, ok := v.(string); ok {
					variants[refName(ref)] = true
				}
			}
		}
		for v := range backref[name] {
			variants[v] = true
		}
		names := make([]string, 0, len(variants))
		for v := range variants {
			names = append(names, v)
		}
		sort.Strings(names)
		oneOf := make([]any, 0, len(names))
		for _, v := range names {
			oneOf = append(oneOf, map[string]any{"$ref": schemaRefPrefix + v})
		}
		sm["oneOf"] = oneOf
	}
}

// allOfBackrefs maps each schema to the set of schemas that extend it via allOf.
func allOfBackrefs(schemas map[string]any) map[string]map[string]bool {
	backref := map[string]map[string]bool{}
	for name, s := range schemas {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		allOf, _ := sm["allOf"].([]any)
		for _, m := range allOf {
			mm, _ := m.(map[string]any)
			ref, _ := mm["$ref"].(string)
			if ref == "" {
				continue
			}
			p := refName(ref)
			if backref[p] == nil {
				backref[p] = map[string]bool{}
			}
			backref[p][name] = true
		}
	}
	return backref
}

// assertNoOneOfCycles guards against a synthesized oneOf chain that loops (a
// member that is itself a union reaching back to its parent), which would make
// oapi-codegen recurse forever.
func assertNoOneOfCycles(schemas map[string]any) error {
	oneOfMembers := func(name string) []string {
		sm, _ := schemas[name].(map[string]any)
		members, _ := sm["oneOf"].([]any)
		var out []string
		for _, m := range members {
			mm, _ := m.(map[string]any)
			if ref, ok := mm["$ref"].(string); ok {
				out = append(out, refName(ref))
			}
		}
		return out
	}
	const ( //nolint:revive
		visiting = 1
		done     = 2
	)
	state := map[string]int{}
	var visit func(string) []string
	visit = func(name string) []string {
		switch state[name] {
		case done:
			return nil
		case visiting:
			return []string{name}
		}
		state[name] = visiting
		for _, child := range oneOfMembers(name) {
			if cyc := visit(child); cyc != nil {
				return append([]string{name}, cyc...)
			}
		}
		state[name] = done
		return nil
	}
	names := make([]string, 0, len(schemas))
	for n := range schemas {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if cyc := visit(n); cyc != nil {
			return fmt.Errorf("circular oneOf chain: %s", strings.Join(cyc, " -> "))
		}
	}
	return nil
}

// deepCopy returns a structural copy of a decoded-JSON value so inlined fragments
// never alias the source schema.
func deepCopy(v any) any {
	switch n := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(n))
		for k, val := range n {
			out[k] = deepCopy(val)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, val := range n {
			out[i] = deepCopy(val)
		}
		return out
	default:
		return v
	}
}
