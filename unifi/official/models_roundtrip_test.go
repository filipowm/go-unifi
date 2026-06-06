package official //nolint:testpackage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFirewallPolicyActionRoundTrip exercises a FLAT discriminated family
// (ALLOW/BLOCK/REJECT): the generated union must decode the controller's wire
// discriminator and expose the right variant, including an empty variant alias.
func TestFirewallPolicyActionRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("ALLOW carries its own field", func(t *testing.T) {
		t.Parallel()
		var a FirewallPolicyAction
		require.NoError(t, json.Unmarshal([]byte(`{"type":"ALLOW","allowReturnTraffic":true}`), &a))

		d, err := a.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "ALLOW", d)

		allow, err := a.AsFirewallPolicyActionAllow()
		require.NoError(t, err)
		require.NotNil(t, allow.AllowReturnTraffic)
		assert.True(t, *allow.AllowReturnTraffic)

		// Re-marshal/unmarshal must keep the discriminator stable.
		b, err := json.Marshal(a)
		require.NoError(t, err)
		var a2 FirewallPolicyAction
		require.NoError(t, json.Unmarshal(b, &a2))
		d2, _ := a2.Discriminator()
		assert.Equal(t, "ALLOW", d2)
	})

	for _, tc := range []string{"BLOCK", "REJECT"} {
		t.Run(tc+" is an empty-variant alias", func(t *testing.T) {
			t.Parallel()
			var a FirewallPolicyAction
			require.NoError(t, json.Unmarshal([]byte(`{"type":"`+tc+`"}`), &a))
			d, err := a.Discriminator()
			require.NoError(t, err)
			assert.Equal(t, tc, d)

			v, err := a.ValueByDiscriminator()
			require.NoError(t, err)
			assert.NotNil(t, v)
		})
	}
}

// TestWifiSecurityConfigurationNestedRoundTrip exercises a NESTED discriminated
// family: WifiSecurityConfigurationDetailObject -> WPA2_ENTERPRISE variant ->
// radiusConfiguration.nasId, itself a discriminated union. The whole tree must
// decode and round-trip with no field loss and no circular-chain blow-up.
func TestWifiSecurityConfigurationNestedRoundTrip(t *testing.T) {
	t.Parallel()

	const payload = `{
		"type":"WPA2_ENTERPRISE",
		"coaEnabled":true,
		"radiusConfiguration":{
			"profileId":"11111111-1111-1111-1111-111111111111",
			"nasId":{"type":"USER_DEFINED","value":"my-nas"}
		}
	}`

	var sec WifiSecurityConfigurationDetailObject
	require.NoError(t, json.Unmarshal([]byte(payload), &sec))

	d, err := sec.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, "WPA2_ENTERPRISE", d)

	wpa2, err := sec.AsWifiWpa2EnterpriseSecurityConfigurationDetail()
	require.NoError(t, err)
	require.NotNil(t, wpa2.CoaEnabled)
	assert.True(t, *wpa2.CoaEnabled)
	require.NotNil(t, wpa2.RadiusConfiguration, "nested radiusConfiguration must survive")

	// Drill into the nested sub-discriminator (nasId).
	nasID, err := wpa2.RadiusConfiguration.NasId.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, "USER_DEFINED", nasID)
	nas, err := wpa2.RadiusConfiguration.NasId.AsWifiUserDefinedNasId()
	require.NoError(t, err)
	require.NotNil(t, nas.Value)
	assert.Equal(t, "my-nas", *nas.Value)

	// Round-trip the whole tree; the top discriminator must remain stable.
	b, err := json.Marshal(sec)
	require.NoError(t, err)
	var sec2 WifiSecurityConfigurationDetailObject
	require.NoError(t, json.Unmarshal(b, &sec2))
	d2, _ := sec2.Discriminator()
	assert.Equal(t, "WPA2_ENTERPRISE", d2)
}
