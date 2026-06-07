package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Firewall":                   "Firewall",             // as-is
		"Networks":                   "Networks",             // as-is
		"Clients":                    "Clients",              // as-is
		"Sites":                      "Sites",                // as-is
		"Hotspot":                    "Hotspot",              // as-is
		"UniFi Devices":              "Devices",              // override
		"DNS Policies":               "DNSPolicies",          // override: plural collection
		"Access Control (ACL Rules)": "ACLs",                 // override: plural collection
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
	// item != "" marks a list op (List<Qualifier>); a GET with no item is a
	// single read (a trailing Details qualifier is dropped); other verbs pass
	// through after stem stripping.
	cases := []struct{ group, op, method, item, want string }{
		{"Firewall", "CreateFirewallPolicy", "POST", "", "CreatePolicy"},
		{"Firewall", "GetFirewallPolicyOrdering", "GET", "", "GetPolicyOrdering"},
		{"Firewall", "GetFirewallZones", "GET", "FirewallZone", "ListZones"},
		{"Firewall", "GetFirewallPolicies", "GET", "FirewallPolicy", "ListPolicies"},
		{"Firewall", "GetFirewallPolicy", "GET", "", "GetPolicy"},
		{"Firewall", "GetFirewallZone", "GET", "", "GetZone"},
		{"Devices", "AdoptDevice", "POST", "", "Adopt"},
		{"Devices", "GetAdoptedDeviceOverviewPage", "GET", "AdoptedDeviceOverview", "ListAdopted"},
		{"Devices", "GetAdoptedDeviceDetails", "GET", "", "GetAdopted"},
		{"Devices", "GetPendingDevicePage", "GET", "DevicePendingAdoption", "ListPending"},
		{"Devices", "ExecutePortAction", "POST", "", "ExecutePortAction"}, // no Device token
		{"Networks", "GetNetworksOverviewPage", "GET", "NetworkOverview", "List"},
		{"Networks", "GetNetworkDetails", "GET", "", "Get"},
		{"Networks", "GetNetworkReferences", "GET", "", "GetReferences"},
		{"DNSPolicies", "CreateDnsPolicy", "POST", "", "Create"},
		{"DNSPolicies", "GetDnsPolicy", "GET", "", "Get"},
		{"DNSPolicies", "GetDnsPolicyPage", "GET", "DNSPolicy", "List"},
		{"ACLs", "CreateAclRule", "POST", "", "CreateRule"},
		{"ACLs", "GetAclRulePage", "GET", "ACLRuleObject", "ListRule"},
		{"ACLs", "GetAclRuleOrdering", "GET", "", "GetRuleOrdering"},
		{"ACLs", "GetAclRule", "GET", "", "GetRule"},
		{"TrafficMatchingLists", "CreateTrafficMatchingList", "POST", "", "Create"},
		{"TrafficMatchingLists", "GetTrafficMatchingList", "GET", "", "Get"},
		{"TrafficMatchingLists", "GetTrafficMatchingLists", "GET", "TrafficMatchingList", "List"},
		{"WifiBroadcasts", "GetWifiBroadcastPage", "GET", "WifiBroadcastOverview", "List"},
		{"WifiBroadcasts", "GetWifiBroadcastDetails", "GET", "", "Get"},
		{"Hotspot", "GetVouchers", "GET", "HotspotVoucherDetails", "ListVouchers"}, // no stem token
		{"Hotspot", "GetVoucher", "GET", "", "GetVoucher"},
		{"Supporting", "GetDeviceTagPage", "GET", "DeviceTag", "ListDeviceTag"}, // "Device" kept: not Supporting's stem
	}
	for _, c := range cases {
		op := operation{Group: c.group, Name: c.op, HTTPMethod: c.method, ItemType: c.item}
		assert.Equalf(t, c.want, methodName(op), "methodName(%q, %q)", c.group, c.op)
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
