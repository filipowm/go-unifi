package main

import (
	"errors"
	"strings"
	"unicode"
)

// groupOverrides maps an OpenAPI tag to a Go group name where the default
// PascalCase normalization is ambiguous or unwieldy. Entries mirror nameOverrides:
// a new tag auto-yields a group; this table only tidies the awkward ones. Tags
// not listed normalize verbatim ("Firewall", "Networks", "Clients", "Sites",
// "Hotspot").
var groupOverrides = map[string]string{
	"UniFi Devices":              "Devices",
	"DNS Policies":               "DnsPolicy",
	"Access Control (ACL Rules)": "ACL",
	"Traffic Matching Lists":     "TrafficMatching",
	"WiFi Broadcasts":            "WifiBroadcasts",
	"Supporting Resources":       "Supporting",
	"Application Info":           "Info",
}

// operationGroup resolves an operation's group from its primary (first) tag.
// Docs-only tags carry zero operations and so never reach here; an operation with
// no tag is a spec error and fails loud.
func operationGroup(raw map[string]any) (string, error) {
	tags, _ := raw["tags"].([]any)
	if len(tags) == 0 {
		return "", errors.New("has no tag (cannot assign a group)")
	}
	tag, _ := tags[0].(string)
	if tag == "" {
		return "", errors.New("has an empty primary tag")
	}
	return groupName(tag), nil
}

// groupName maps an OpenAPI tag to its Go group name: explicit override else the
// default PascalCase normalization.
func groupName(tag string) string {
	if o, ok := groupOverrides[tag]; ok {
		return o
	}
	return defaultGroupName(tag)
}

// defaultGroupName PascalCases a tag: split on non-alphanumerics, upper-case the
// first rune of each word, join. Existing casing inside a word is preserved.
func defaultGroupName(tag string) string {
	var b strings.Builder
	for _, word := range strings.FieldsFunc(tag, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		b.WriteString(upperFirst(word))
	}
	return b.String()
}

// methodName strips the group's resource word(s) from a PascalCase operation name
// so it reads cleanly under the accessor (createFirewallPolicy under Firewall ->
// CreatePolicy). The leading verb is never stripped; a word matches a stem
// case-insensitively, singular or plural.
func methodName(group, opName string) string {
	stem := tokenize(group)
	tokens := tokenize(opName)
	out := make([]string, 0, len(tokens))
	for i, tok := range tokens {
		if i > 0 && matchesStem(tok, stem) {
			continue
		}
		out = append(out, upperFirst(tok))
	}
	return strings.Join(out, "")
}

// matchesStem reports whether a token matches any stem word, comparing
// case-insensitively and ignoring a single trailing plural "s".
func matchesStem(token string, stem []string) bool {
	for _, s := range stem {
		if strings.EqualFold(token, s) || strings.EqualFold(singular(token), singular(s)) {
			return true
		}
	}
	return false
}

// singular drops a single trailing "s" so plural and singular resource words match.
func singular(s string) string {
	if len(s) > 1 && (s[len(s)-1] == 's' || s[len(s)-1] == 'S') {
		return s[:len(s)-1]
	}
	return s
}

// tokenize splits a camelCase/PascalCase identifier into words at each
// lowercase/digit-to-uppercase boundary. The operationIds carry no consecutive
// capitals, so this is exact.
func tokenize(s string) []string {
	var tokens []string
	start := 0
	for i, r := range s {
		if i > start && unicode.IsUpper(r) && !unicode.IsUpper(rune(s[i-1])) {
			tokens = append(tokens, s[start:i])
			start = i
		}
	}
	if start < len(s) {
		tokens = append(tokens, s[start:])
	}
	return tokens
}

// upperFirst upper-cases the first rune of s, preserving the rest.
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// lowerFirst lower-cases the first rune of s, preserving the rest. It names the
// unexported per-group impl type (Firewall -> firewallClient).
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
