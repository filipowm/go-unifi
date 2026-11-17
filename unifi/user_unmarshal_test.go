package unifi_test

import (
	"encoding/json"
	"testing"

	"github.com/filipowm/go-unifi/v2/unifi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserUnmarshalDevIdOverride(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body string
		want int
	}{
		"number":       {`{"dev_id_override": 7}`, 7},
		"numeric text": {`{"dev_id_override": "7"}`, 7},
		"empty string": {`{"dev_id_override": ""}`, 0},
		"null":         {`{"dev_id_override": null}`, 0},
		"absent":       {`{}`, 0},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var u unifi.User
			require.NoError(t, json.Unmarshal([]byte(tc.body), &u))
			assert.Equal(t, tc.want, u.DevIdOverride)
		})
	}
}
