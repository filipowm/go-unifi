package unifi //nolint: testpackage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserUnmarshalDevIdOverride covers the forms dev_id_override arrives in.
// UniFi Network 11 sends a numeric string ("3263") where earlier controllers
// sent a number; a decode failure here makes the whole User unreadable.
func TestUserUnmarshalDevIdOverride(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input string
		want  int
	}{
		"number":         {input: `{"mac":"aa:bb:cc:dd:ee:ff","dev_id_override":3263}`, want: 3263},
		"numeric string": {input: `{"mac":"aa:bb:cc:dd:ee:ff","dev_id_override":"3263"}`, want: 3263},
		"empty string":   {input: `{"mac":"aa:bb:cc:dd:ee:ff","dev_id_override":""}`, want: 0},
		"null":           {input: `{"mac":"aa:bb:cc:dd:ee:ff","dev_id_override":null}`, want: 0},
		"absent":         {input: `{"mac":"aa:bb:cc:dd:ee:ff"}`, want: 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var u User
			require.NoError(t, json.Unmarshal([]byte(tc.input), &u))
			assert.Equal(t, tc.want, u.DevIdOverride)
			assert.Equal(t, "aa:bb:cc:dd:ee:ff", u.MAC)
		})
	}
}
