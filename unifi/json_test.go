package unifi //nolint: testpackage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBooleanishStringUnmarshal locks in the permissive decoding contract for
// booleanishString (ARCH-02): the wired Device fields (LtePoe/LteExtAnt) now send
// bare JSON booleans, while older controllers sent "enabled"/"disabled". The
// decoder must accept all of these forms — and NEVER hard-error on an unexpected
// scalar (a single bad field must not poison the whole Device decode).
func TestBooleanishStringUnmarshal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input string
		want  bool
	}{
		"bare true":         {input: `true`, want: true},
		"bare false":        {input: `false`, want: false},
		"quoted true":       {input: `"true"`, want: true},
		"quoted false":      {input: `"false"`, want: false},
		"enabled":           {input: `"enabled"`, want: true},
		"disabled":          {input: `"disabled"`, want: false},
		"string one":        {input: `"1"`, want: true},
		"string zero":       {input: `"0"`, want: false},
		"empty string":      {input: `""`, want: false},
		"null":              {input: `null`, want: false},
		"garbage string":    {input: `"wat"`, want: false},
		"unexpected number": {input: `42`, want: false},
		"object":            {input: `{"x":1}`, want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			var b booleanishString
			err := json.Unmarshal([]byte(tc.input), &b)
			require.NoError(t, err, "booleanishString must never hard-error on input %q", tc.input)
			a.Equal(tc.want, bool(b), "input %q decoded to wrong bool", tc.input)
		})
	}
}
