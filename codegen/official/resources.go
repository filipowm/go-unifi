package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// customOps are operationIds whose wrappers live in hand-written siblings
// (info.go, sites.go) and must NOT be generated. Their Go types collide with
// hand-written models (Info, SiteOverview), so the models pass excludes/aliases
// them too — see transform.go.
var customOps = map[string]bool{
	"getInfo":             true,
	"getSiteOverviewPage": true,
}

// httpMethods is the fixed iteration order for a path item, so generation is
// deterministic regardless of JSON map ordering.
var httpMethods = []string{"get", "post", "put", "delete", "patch"}

// paginationParams are the pagination query params the runtime owns via the
// variadic ListOption (WithOffset/WithLimit on every list wrapper); they never
// surface as positional method arguments. The third standardized list param,
// the optional "filter", is likewise owned by the option (WithFilter) — it is
// dropped from list method args by the required-only check below, while a
// REQUIRED filter on a non-list op (the bulk deleteVouchers) stays an explicit
// arg. Pagination/filtering is thus one uniform mechanism, not a per-resource arg.
var paginationParams = map[string]bool{"offset": true, "limit": true}

// param is one method argument sourced from a path or required query parameter.
type param struct {
	Name  string // Go identifier (the spec parameter name is already lowerCamel)
	Query bool   // true when sourced from a (required) query parameter
}

// operation is the generator's view of one OpenAPI operation: enough to emit the
// wrapper body, the interface signature, and the mock — derived from the
// operationId + HTTP method + parameters, never from path regexes.
type operation struct {
	Name       string  // Go method name (PascalCase operationId)
	Group      string  // PascalCase group name from the operation's primary tag
	HTTPMethod string  // GET/POST/PUT/DELETE/PATCH
	SubPath    string  // path with the leading /v1 stripped and {param}->%s
	PathArgs   []param // ordered path arguments (URL order)
	QueryArgs  []param // ordered required query arguments
	BodyType   string  // Go request-body type, "" when none
	ItemType   string  // Go element type for a paginated list, "" when not a list
	ReturnType string  // Go single-return type, "" when the call returns no body
}

// IsList reports whether the wrapper auto-paginates and returns a slice.
func (o operation) IsList() bool { return o.ItemType != "" }

// RequiredFilter returns the name of a required "filter" query arg, if any, so
// the wrapper can guard against an empty value (bulk deleteVouchers).
func (o operation) RequiredFilter() string {
	for _, q := range o.QueryArgs {
		if q.Name == "filter" {
			return q.Name
		}
	}
	return ""
}

// buildOperations extracts every generatable operation from the untouched spec
// document, resolving request/response Go types through the same finalName map
// the models pass uses, so wrapper types match the emitted model types exactly.
func buildOperations(doc map[string]any) ([]operation, error) {
	schemas, err := schemasOf(doc)
	if err != nil {
		return nil, err
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		return nil, errors.New("spec has no paths object")
	}
	pathKeys := make([]string, 0, len(paths))
	for p := range paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	var ops []operation
	for _, p := range pathKeys {
		item, ok := paths[p].(map[string]any)
		if !ok {
			continue
		}
		for _, m := range httpMethods {
			raw, ok := item[m].(map[string]any)
			if !ok {
				continue
			}
			op, skip, err := buildOperation(p, m, raw, schemas)
			if err != nil {
				return nil, err
			}
			if !skip {
				ops = append(ops, op)
			}
		}
	}
	return ops, nil
}

// buildOperation classifies a single operation; skip is true for hand-written
// operations that must not be generated.
func buildOperation(path, method string, raw, schemas map[string]any) (operation, bool, error) {
	opID, _ := raw["operationId"].(string)
	if opID == "" {
		return operation{}, false, fmt.Errorf("operation %s %s has no operationId", strings.ToUpper(method), path)
	}
	if customOps[opID] {
		return operation{}, true, nil
	}
	group, err := operationGroup(raw)
	if err != nil {
		return operation{}, false, fmt.Errorf("operation %s %s: %w", strings.ToUpper(method), path, err)
	}
	op := operation{
		Name:       pascal(opID),
		Group:      group,
		HTTPMethod: strings.ToUpper(method),
		SubPath:    subPath(path),
		PathArgs:   pathArgs(path),
		QueryArgs:  requiredQueryArgs(raw),
		BodyType:   bodyType(raw),
	}

	respName := successResponseSchema(raw)
	switch {
	case respName == "": // no response body (DELETE / action POST)
	case method == "get" && pageItemSchema(schemas, respName) != "":
		op.ItemType = finalName(pageItemSchema(schemas, respName))
	default:
		op.ReturnType = finalName(respName)
	}
	return op, false, nil
}

// pascal upper-cases the first rune of a lowerCamel operationId.
func pascal(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// subPath strips the leading /v1 version segment and rewrites {param} to %s, the
// printf placeholder the wrapper fills from its path arguments.
func subPath(path string) string {
	sub := strings.TrimPrefix(path, "/v1")
	var b strings.Builder
	for seg := range strings.SplitSeq(sub, "/") {
		if seg == "" {
			continue
		}
		b.WriteByte('/')
		if strings.HasPrefix(seg, "{") {
			b.WriteString("%s")
		} else {
			b.WriteString(seg)
		}
	}
	return b.String()
}

// pathArgs returns the {param} names in URL order, the canonical argument order.
func pathArgs(path string) []param {
	var out []param
	for seg := range strings.SplitSeq(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			out = append(out, param{Name: seg[1 : len(seg)-1]})
		}
	}
	return out
}

// requiredQueryArgs returns the required non-pagination query parameters in
// source order; these become trailing method arguments.
func requiredQueryArgs(raw map[string]any) []param {
	params, _ := raw["parameters"].([]any)
	var out []param
	for _, p := range params {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if pm["in"] != "query" {
			continue
		}
		required, _ := pm["required"].(bool)
		name, _ := pm["name"].(string)
		if !required || paginationParams[name] {
			continue
		}
		out = append(out, param{Name: name, Query: true})
	}
	return out
}

// bodyType returns the Go type of the JSON request body, or "" when absent.
func bodyType(raw map[string]any) string {
	if ref := jsonSchemaRef(raw["requestBody"]); ref != "" {
		return finalName(ref)
	}
	return ""
}

// successResponseSchema returns the schema name of the 200/201 JSON response, or
// "" when the operation returns no body.
func successResponseSchema(raw map[string]any) string {
	responses, ok := raw["responses"].(map[string]any)
	if !ok {
		return ""
	}
	for _, code := range []string{"200", "201"} {
		if ref := jsonSchemaRef(responses[code]); ref != "" {
			return ref
		}
	}
	return ""
}

// jsonSchemaRef digs application/json.schema.$ref out of a requestBody/response
// object, returning the bare schema name.
func jsonSchemaRef(node any) string {
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	content, ok := m["content"].(map[string]any)
	if !ok {
		return ""
	}
	media, ok := content["application/json"].(map[string]any)
	if !ok {
		return ""
	}
	schema, ok := media["schema"].(map[string]any)
	if !ok {
		return ""
	}
	if ref, ok := schema["$ref"].(string); ok {
		return refName(ref)
	}
	return ""
}

// pageItemSchema returns the element schema of a paginated envelope (its data
// array items $ref), or "" when respName is not a page.
func pageItemSchema(schemas map[string]any, respName string) string {
	s, ok := schemas[respName].(map[string]any)
	if !ok {
		return ""
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		return ""
	}
	data, ok := props["data"].(map[string]any)
	if !ok || data["type"] != "array" {
		return ""
	}
	items, ok := data["items"].(map[string]any)
	if !ok {
		return ""
	}
	if ref, ok := items["$ref"].(string); ok {
		return refName(ref)
	}
	return ""
}
