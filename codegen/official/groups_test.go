package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Firewall":                   "Firewall",  // as-is
		"Networks":                   "Networks",  // as-is
		"Clients":                    "Clients",   // as-is
		"Sites":                      "Sites",     // as-is
		"Hotspot":                    "Hotspot",   // as-is
		"UniFi Devices":              "Devices",   // override
		"DNS Policies":               "DNSPolicies",      // override: plural collection
		"Access Control (ACL Rules)": "ACLs",            // override: plural collection
		"Traffic Matching Lists":     "TrafficMatchingLists", // override: plural collection
		"WiFi Broadcasts":            "WifiBroadcasts",
		"Supporting Resources":       "Supporting",
		"Application Info":           "Info",
		// A brand-new tag auto-yields a default PascalCase group (caught in the golden diff).
		"Brand New Thing": "BrandNewThing",
	}
	for in, want := range cases {
		assert.Equalf(t, want, groupName(in), "groupName(%q)", in)
	}
}

func TestMethodName(t *testing.T) {
	t.Parallel()
	cases := []struct{ group, op, want string }{
		{"Firewall", "CreateFirewallPolicy", "CreatePolicy"},
		{"Firewall", "GetFirewallPolicyOrdering", "GetPolicyOrdering"},
		{"Firewall", "GetFirewallZones", "GetZones"},
		{"Devices", "AdoptDevice", "Adopt"},
		{"Devices", "GetAdoptedDeviceOverviewPage", "GetAdoptedOverviewPage"},
		{"Devices", "ExecutePortAction", "ExecutePortAction"}, // no Device token
		{"Networks", "GetNetworksOverviewPage", "GetOverviewPage"},
		{"DNSPolicies", "CreateDnsPolicy", "Create"},
		{"DNSPolicies", "GetDnsPolicy", "Get"},
		{"DNSPolicies", "GetDnsPolicyPage", "GetPage"},
		{"ACLs", "CreateAclRule", "CreateRule"},
		{"ACLs", "GetAclRulePage", "GetRulePage"},
		{"ACLs", "GetAclRuleOrdering", "GetRuleOrdering"},
		{"TrafficMatchingLists", "CreateTrafficMatchingList", "Create"},
		{"TrafficMatchingLists", "GetTrafficMatchingList", "Get"},
		{"TrafficMatchingLists", "GetTrafficMatchingLists", "GetLists"},
		{"WifiBroadcasts", "GetWifiBroadcastPage", "GetPage"},
		{"Hotspot", "GetVouchers", "GetVouchers"},              // no stem token
		{"Supporting", "GetDeviceTagPage", "GetDeviceTagPage"}, // "Device" kept: not Supporting's stem
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, methodName(c.group, c.op), "methodName(%q, %q)", c.group, c.op)
	}
}

// TestBuildGroupsFromSpec asserts grouping derives from tags: every tag with
// operations yields a group, hand-written groups carry their custom methods, and
// each group's stripped method names are unique.
func TestBuildGroupsFromSpec(t *testing.T) {
	t.Parallel()
	groups, err := buildGroups(loadOps(t))
	require.NoError(t, err)

	byName := map[string]group{}
	for _, g := range groups {
		byName[g.Name] = g
	}
	// Docs-only tags (zero ops) never appear; hand-written-only groups do.
	for _, want := range []string{"Firewall", "Devices", "ACLs", "DNSPolicies", "Info", "Sites"} {
		_, ok := byName[want]
		assert.Truef(t, ok, "expected group %q", want)
	}
	for _, gone := range []string{"GettingStarted", "Filtering", "ErrorHandling"} {
		_, ok := byName[gone]
		assert.Falsef(t, ok, "docs-only tag %q must not yield a group", gone)
	}

	// Info/Sites carry only their re-homed hand-written methods (op == nil).
	requireMethod(t, byName["Info"], "Get", true)
	requireMethod(t, byName["Sites"], "List", true)
	requireMethod(t, byName["Sites"], "ResolveID", true)
	// Firewall carries generated wrappers.
	requireMethod(t, byName["Firewall"], "CreatePolicy", false)
}

// TestBuildGroupsCollisionFailsLoud feeds two operations that strip to the same
// name into one group and asserts the guard fires.
func TestBuildGroupsCollisionFailsLoud(t *testing.T) {
	t.Parallel()
	ops := []operation{
		{Name: "GetFirewallPolicy", Group: "Firewall", HTTPMethod: "GET", SubPath: "/x", ReturnType: "FirewallPolicy"},
		{Name: "GetPolicy", Group: "Firewall", HTTPMethod: "GET", SubPath: "/y", ReturnType: "FirewallPolicy"},
	}
	_, err := buildGroups(ops)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate method name")
}

// requireMethod asserts a group contains a method, optionally hand-written (op nil).
func requireMethod(t *testing.T, g group, name string, handWritten bool) {
	t.Helper()
	for _, m := range g.Methods {
		if m.Name == name {
			assert.Equalf(t, handWritten, m.op == nil, "method %s.%s hand-written?", g.Name, name)
			return
		}
	}
	t.Fatalf("group %s missing method %q", g.Name, name)
}
