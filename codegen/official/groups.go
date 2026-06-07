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
//
// Naming convention (mirrors go-github / k8s / Stripe SDK idioms):
//   - PLURAL for true resource collections (DNSPolicies, ACLs, TrafficMatchingLists)
//   - SINGULAR for feature-area groups (Firewall, Hotspot, Supporting, Info)
var groupOverrides = map[string]string{
	"UniFi Devices":              "Devices",
	"DNS Policies":               "DNSPolicies",
	"Access Control (ACL Rules)": "ACLs",
	"Traffic Matching Lists":     "TrafficMatchingLists",
	"WiFi Broadcasts":            "WifiBroadcasts",
	"Supporting Resources":       "Supporting",
	"Application Info":           "Info",
}

// stemOverrides maps a group name to the explicit token list used by methodName
// when tokenize(group) would not produce the correct strip set — e.g. because
// the group name is pluralised ("DNSPolicies", "ACLs") but the operationId
// contains the singular resource form ("DnsPolicy", "Acl", "TrafficMatchingList").
var stemOverrides = map[string][]string{
	"ACLs":                 {"ACL"},
	"DNSPolicies":          {"DNS", "Policy"},
	"TrafficMatchingLists": {"Traffic", "Matching", "List"},
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
// CreatePolicy), then normalizes the read verb (normalizeReadVerb). The leading
// verb is never stripped; a word matches a stem case-insensitively. For pluralised
// group names where the operationId contains the singular resource form,
// stemOverrides provides the correct token set.
func methodName(op operation) string {
	stem, ok := stemOverrides[op.Group]
	if !ok {
		stem = tokenize(op.Group)
	}
	tokens := tokenize(op.Name)
	out := make([]string, 0, len(tokens))
	for i, tok := range tokens {
		if i > 0 && matchesStem(tok, stem) {
			continue
		}
		out = append(out, upperFirst(tok))
	}
	return strings.Join(normalizeReadVerb(out, op), "")
}

// listEnvelope are the page/collection words a normalized collection read drops:
// the verb already says List, so Page/Overview/List(s) are redundant noise.
var listEnvelope = map[string]bool{"Page": true, "Overview": true, "List": true, "Lists": true}

// normalizeReadVerb makes reads uniform across the surface: a collection read
// becomes List<Qualifier> (Get->List, trailing envelope words dropped) and a
// single-item GET drops a trailing Details qualifier (Get<Qualifier>). Non-read
// verbs (Create/Update/Delete/Patch/Execute/...) pass through untouched.
func normalizeReadVerb(tokens []string, op operation) []string {
	switch {
	case op.IsList():
		if len(tokens) > 0 && tokens[0] == "Get" {
			tokens[0] = "List"
		}
		for len(tokens) > 1 && listEnvelope[tokens[len(tokens)-1]] {
			tokens = tokens[:len(tokens)-1]
		}
	case op.HTTPMethod == "GET" && len(tokens) > 1 && tokens[len(tokens)-1] == "Details":
		tokens = tokens[:len(tokens)-1]
	}
	return tokens
}

// matchesStem reports whether a token matches any stem word, comparing
// case-insensitively. The stem word is also checked in its singular form (so a
// plural stem "Broadcasts" matches token "Broadcast"). The token itself is never
// singularised: that would cause "Lists" to match a singular stem "List", making
// the pluralised list-endpoint operationId indistinguishable from the single-item
// one and producing a within-group method-name collision.
func matchesStem(token string, stem []string) bool {
	for _, s := range stem {
		if strings.EqualFold(token, s) || strings.EqualFold(token, singular(s)) {
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

// tokenize splits a camelCase/PascalCase/ACRONYM identifier into words. For
// plain camelCase the split is at each lower-to-upper boundary; for acronyms
// (consecutive uppercase), the split also fires at the acronym-to-word boundary
// so that "DNSPolicy" -> ["DNS","Policy"] and "ACL" stays ["ACL"].
func tokenize(s string) []string {
	runes := []rune(s)
	n := len(runes)
	var tokens []string
	start := 0
	for i := 1; i < n; i++ {
		if unicode.IsUpper(runes[i]) {
			switch {
			case !unicode.IsUpper(runes[i-1]):
				// lower-to-upper: "WifiBroadcasts" -> "Wifi" / "Broadcasts"
				tokens = append(tokens, string(runes[start:i]))
				start = i
			case i+1 < n && !unicode.IsUpper(runes[i+1]):
				// upper-run before a lowercase word: "DNSPolicy" -> "DNS" / "Policy"
				tokens = append(tokens, string(runes[start:i]))
				start = i
			}
		}
	}
	if start < n {
		tokens = append(tokens, string(runes[start:]))
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
