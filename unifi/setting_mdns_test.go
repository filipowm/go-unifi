package unifi_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filipowm/go-unifi/v2/unifi"
)

// TestSettingMdnsNetworkSelectionRoundTrip checks the network selection survives
// a decode/encode, and that a nil selection is omitted while an explicitly empty
// one is sent as [] so a prior selection can be cleared.
func TestSettingMdnsNetworkSelectionRoundTrip(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		in         string
		wantFor    string
		wantIDs    []string
		wantOutput string
	}{
		"live controller shape (enabled_for some)": {
			in:         `{"key":"mdns","mode":"all","enabled_for":"some","enabled_for_network_ids":["net-a","net-b"],"predefined_services":[],"custom_services":[]}`,
			wantFor:    "some",
			wantIDs:    []string{"net-a", "net-b"},
			wantOutput: `{"key":"mdns","mode":"all","enabled_for":"some","enabled_for_network_ids":["net-a","net-b"]}`,
		},
		"nil network ids are omitted": {
			in:         `{"key":"mdns","mode":"auto","enabled_for":"all"}`,
			wantFor:    "all",
			wantOutput: `{"key":"mdns","mode":"auto","enabled_for":"all"}`,
		},
		"explicitly empty network ids are sent as []": {
			in:         `{"key":"mdns","mode":"auto","enabled_for":"all","enabled_for_network_ids":[]}`,
			wantFor:    "all",
			wantIDs:    []string{},
			wantOutput: `{"key":"mdns","mode":"auto","enabled_for":"all","enabled_for_network_ids":[]}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var s unifi.SettingMdns
			require.NoError(t, json.Unmarshal([]byte(tc.in), &s))
			assert.Equal(t, tc.wantFor, s.EnabledFor)
			assert.Equal(t, tc.wantIDs, s.EnabledForNetworkIds)

			out, err := json.Marshal(&s)
			require.NoError(t, err)
			assert.JSONEq(t, tc.wantOutput, string(out))
		})
	}
}
