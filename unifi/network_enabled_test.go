package unifi_test

import (
	"encoding/json"
	"testing"

	"github.com/filipowm/go-unifi/v2/unifi"
)

// Regression: an absent `enabled` key must unmarshal to true (the controller omits
// it for an enabled network). Without this, terraform import records enabled=false
// and the next full-replace PUT disables the network.
func TestNetworkEnabledAbsentDefaultsTrue(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want bool
	}{
		{"absent -> true", `{"name":"WAN","purpose":"wan"}`, true},
		{"explicit false -> false", `{"enabled":false}`, false},
		{"explicit true -> true", `{"enabled":true}`, true},
	} {
		var n unifi.Network
		if err := json.Unmarshal([]byte(c.in), &n); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if n.Enabled != c.want {
			t.Errorf("%s: Enabled = %v, want %v", c.name, n.Enabled, c.want)
		}
	}
}
