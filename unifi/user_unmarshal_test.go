package unifi

import (
	"encoding/json"
	"testing"
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

			var u User
			if err := json.Unmarshal([]byte(tc.body), &u); err != nil {
				t.Fatalf("unmarshal %s: %v", tc.body, err)
			}
			if u.DevIdOverride != tc.want {
				t.Fatalf("DevIdOverride = %d, want %d", u.DevIdOverride, tc.want)
			}
		})
	}
}
