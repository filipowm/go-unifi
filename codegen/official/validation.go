package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// extExtraTags is the oapi-codegen extension that appends raw struct-tag
	// key/values to a generated field.
	extExtraTags = "x-oapi-codegen-extra-tags"
	// maxItemsSentinel is the spec's "unbounded" placeholder (math.MaxInt32);
	// emitting it as a max= rule would be meaningless noise, so it is dropped.
	maxItemsSentinel = 2147483647
)

// injectValidationTags walks every schema and stamps a go-playground `validate`
// tag derived from its OpenAPI constraints onto each constrained property. It
// runs BEFORE dedupeEnums so inline enum values are still readable; enumRef then
// carries the x-* tag through dedup. Fails loud on a tag-corrupting enum value.
func injectValidationTags(schemas map[string]any) error {
	for name, s := range schemas {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		for _, bag := range propertyBags(sm) {
			for prop, node := range bag {
				pm, ok := node.(map[string]any)
				if !ok {
					continue
				}
				rules, err := validationRules(pm)
				if err != nil {
					return fmt.Errorf("%s.%s: %w", name, prop, err)
				}
				if len(rules) == 0 {
					continue
				}
				setValidateTag(pm, rules)
			}
		}
	}
	return nil
}

// setValidateTag stores the rules as a validate struct-tag, always leading with
// omitempty so a nil pointer / absent field skips validation rather than
// false-failing (JSON omitempty != validator omitempty).
func setValidateTag(prop map[string]any, rules []string) {
	tag := "omitempty," + strings.Join(rules, ",")
	prop[extExtraTags] = map[string]any{"validate": tag}
}

// validationRules maps a property's OpenAPI constraints to ordered go-playground
// rules: enum->oneof (scalar or array via dive), numeric min/max->gte/lte,
// string length->min/max, array length->min/max. required and format are NOT
// emitted (see codegen/CLAUDE.md for why).
func validationRules(prop map[string]any) ([]string, error) {
	switch prop["type"] {
	case "array":
		return arrayRules(prop)
	case "string":
		return scalarRules(prop, lengthRules(prop, "minLength", "maxLength"))
	case "integer", "number":
		return scalarRules(prop, numericRules(prop))
	default:
		// Typeless enums (rare) still deserve a oneof; nothing else to emit.
		return scalarRules(prop, nil)
	}
}

// scalarRules prefers a oneof from an enum and otherwise falls back to the
// type's range/length rules (an enum already constrains the value set).
func scalarRules(prop map[string]any, fallback []string) ([]string, error) {
	one, err := oneofRule(prop)
	if err != nil {
		return nil, err
	}
	if one != "" {
		return []string{one}, nil
	}
	return fallback, nil
}

// arrayRules emits slice-length rules then, for an array-of-enum, dive into the
// items so each element is checked against its oneof.
func arrayRules(prop map[string]any) ([]string, error) {
	rules := lengthRules(prop, "minItems", "maxItems")
	items, ok := prop["items"].(map[string]any)
	if !ok {
		return rules, nil
	}
	one, err := oneofRule(items)
	if err != nil {
		return nil, err
	}
	if one != "" {
		rules = append(rules, "dive", one)
	}
	return rules, nil
}

// numericRules turns minimum/maximum into inclusive gte/lte bounds.
func numericRules(prop map[string]any) []string {
	var rules []string
	if v, ok := numberValue(prop["minimum"]); ok {
		rules = append(rules, "gte="+v)
	}
	if v, ok := numberValue(prop["maximum"]); ok {
		rules = append(rules, "lte="+v)
	}
	return rules
}

// lengthRules turns a min/max count pair (string length or array items) into
// go-playground min/max rules, dropping the unbounded maxItems sentinel.
func lengthRules(prop map[string]any, minKey, maxKey string) []string {
	var rules []string
	if v, ok := intValue(prop[minKey]); ok {
		rules = append(rules, "min="+v)
	}
	if n, ok := prop[maxKey].(float64); ok && n != maxItemsSentinel {
		rules = append(rules, "max="+strconv.FormatInt(int64(n), 10))
	}
	return rules
}

// oneofRule renders an enum as a space-delimited oneof. The integer-typed
// property with string enum values quirk (e.g. WifiBasicDataRate) is handled
// transparently since values are formatted from their JSON form. Fails loud if a
// value contains a space, which would silently split into bogus oneof members.
func oneofRule(node map[string]any) (string, error) {
	raw, ok := node["enum"].([]any)
	if !ok {
		return "", nil
	}
	vals := make([]string, 0, len(raw))
	for _, v := range raw {
		s := enumValue(v)
		if strings.Contains(s, " ") {
			return "", fmt.Errorf("enum value %q contains a space; oneof tag would be corrupted", s)
		}
		vals = append(vals, s)
	}
	return "oneof=" + strings.Join(vals, " "), nil
}

// enumValue renders one enum literal for a oneof tag. JSON numbers format
// without a trailing zero; strings (including the int-typed-string quirk) pass
// through verbatim.
func enumValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// intValue formats a JSON number as an integer literal; ok is false when absent.
func intValue(v any) (string, bool) {
	f, ok := v.(float64)
	if !ok {
		return "", false
	}
	return strconv.FormatInt(int64(f), 10), true
}

// numberValue formats a JSON number in plain decimal (no exponent, no trailing
// zero) so go-playground can parse the bound; ok is false when absent.
func numberValue(v any) (string, bool) {
	f, ok := v.(float64)
	if !ok {
		return "", false
	}
	return strconv.FormatFloat(f, 'f', -1, 64), true
}
