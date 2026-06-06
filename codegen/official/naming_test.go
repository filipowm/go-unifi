package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinalName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Firewall policy action":                   "FirewallPolicyAction",
		"IntegrationDnsARecordDto":                 "DnsARecord",
		"IntegrationFirewallPolicyActionAllowDto":  "FirewallPolicyActionAllow",
		"Create or update Network":                 "NetworkCreateOrUpdate",
		"Create or update DNS policy":              "DNSPolicyCreateOrUpdate",
		"ACL ruleObject":                           "ACLRuleObject",
		"ACL rule update":                          "ACLRuleUpdate",
		"Site overview":                            "SiteOverview",
		"IP address selector":                      "IPAddressSelector",
		"IP Address selector":                      "SpecificIPAddressSelector", // explicit override
		"Wifi security configuration detailObject": "WifiSecurityConfigurationDetailObject",
	}
	for in, want := range cases {
		assert.Equalf(t, want, finalName(in), "finalName(%q)", in)
	}
}

func TestBuildRenameMapDetectsCollision(t *testing.T) {
	t.Parallel()
	// Two distinct keys normalizing to the same Go name with no override.
	schemas := map[string]any{
		"Foo bar": map[string]any{},
		"Foo Bar": map[string]any{},
	}
	_, err := buildRenameMap(schemas)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collision")
}

func TestBuildRenameMapResolvesKnownCollision(t *testing.T) {
	t.Parallel()
	// The IP selector case-collision is resolved by the override.
	schemas := map[string]any{
		"IP address selector": map[string]any{},
		"IP Address selector": map[string]any{},
	}
	renames, err := buildRenameMap(schemas)
	require.NoError(t, err)
	assert.Equal(t, "IPAddressSelector", renames["IP address selector"])
	assert.Equal(t, "SpecificIPAddressSelector", renames["IP Address selector"])
}

func TestApplyRenamesRewritesRefsAndMappings(t *testing.T) {
	t.Parallel()
	doc := map[string]any{"components": map[string]any{"schemas": map[string]any{
		"Create or update Network": map[string]any{
			"discriminator": map[string]any{
				"propertyName": "management",
				"mapping":      map[string]any{"GATEWAY": "#/components/schemas/IntegrationGatewayManagedNetworkCreateUpdateDto"},
			},
			"oneOf": []any{map[string]any{"$ref": "#/components/schemas/IntegrationGatewayManagedNetworkCreateUpdateDto"}},
		},
		"IntegrationGatewayManagedNetworkCreateUpdateDto": map[string]any{"type": "object"},
	}}}
	schemas, _ := schemasOf(doc)
	renames, err := buildRenameMap(schemas)
	require.NoError(t, err)
	applyRenames(doc, renames)

	out, _ := schemasOf(doc)
	assert.Contains(t, out, "NetworkCreateOrUpdate")
	assert.Contains(t, out, "GatewayManagedNetworkCreateUpdate")
	parent, ok := out["NetworkCreateOrUpdate"].(map[string]any)
	require.True(t, ok)
	oneOf, ok := parent["oneOf"].([]any)
	require.True(t, ok)
	member, ok := oneOf[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "#/components/schemas/GatewayManagedNetworkCreateUpdate", member["$ref"])
	disc, ok := parent["discriminator"].(map[string]any)
	require.True(t, ok)
	mapping, ok := disc["mapping"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "#/components/schemas/GatewayManagedNetworkCreateUpdate", mapping["GATEWAY"])
}

func TestDedupeEnumsCollapsesACLAction(t *testing.T) {
	t.Parallel()
	action := func() map[string]any {
		return map[string]any{"properties": map[string]any{
			"action": map[string]any{"type": "string", "enum": []any{"ALLOW", "BLOCK"}},
		}}
	}
	schemas := map[string]any{
		"ACL rule":        action(),
		"ACL rule update": action(),
		"ACL ruleObject":  action(),
	}
	require.NoError(t, dedupeEnums(schemas))
	require.Contains(t, schemas, "ACLRuleAction")
	for _, n := range []string{"ACL rule", "ACL rule update", "ACL ruleObject"} {
		props := asMap(t, asMap(t, schemas[n])["properties"])
		prop := asMap(t, props["action"])
		assert.Equal(t, "#/components/schemas/ACLRuleAction", prop["$ref"])
	}
}

func TestDedupeEnumsRejectsValueMismatch(t *testing.T) {
	t.Parallel()
	schemas := map[string]any{
		"ACL rule":        map[string]any{"properties": map[string]any{"action": map[string]any{"type": "string", "enum": []any{"ALLOW", "BLOCK"}}}},
		"ACL rule update": map[string]any{"properties": map[string]any{"action": map[string]any{"type": "string", "enum": []any{"ALLOW", "DENY"}}}},
		"ACL ruleObject":  map[string]any{"properties": map[string]any{"action": map[string]any{"type": "string", "enum": []any{"ALLOW", "BLOCK"}}}},
	}
	err := dedupeEnums(schemas)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "values")
}
