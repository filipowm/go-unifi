package unifi

import (
	"encoding/json"
	"testing"
)

// Regression: an absent `enabled` key must unmarshal to true (not the bool zero
// value false). The controller omits `enabled` for an enabled network, so without
// this `terraform import` records enabled=false and the next full-replace PUT
// disables the network.
func TestNetworkEnabledAbsentDefaultsTrue(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"absent -> true", `{"name":"WAN","purpose":"wan"}`, true},
		{"explicit false -> false", `{"enabled":false}`, false},
		{"explicit true -> true", `{"enabled":true}`, true},
	}
	for _, c := range cases {
		var n Network
		if err := json.Unmarshal([]byte(c.in), &n); err != nil {
			t.Fatalf("%s: unmarshal: %v", c.name, err)
		}
		if n.Enabled != c.want {
			t.Errorf("%s: Enabled = %v, want %v", c.name, n.Enabled, c.want)
		}
	}
}
