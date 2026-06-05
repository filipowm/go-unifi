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

// TestNumberOrStringUnmarshal locks in the numberOrString decoding contract
// (ARCH-07): a JSON null must decode to an empty string rather than the literal
// 4-character string "null" (which would corrupt the value on round-trip),
// integers and floats keep their textual form, and quoted strings (e.g. "auto")
// pass through unquoted. Non-numeric/non-string scalars are rejected rather than
// stored as raw bytes.
func TestNumberOrStringUnmarshal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input   string
		want    string
		wantErr bool
	}{
		"null":          {input: `null`, want: ""},
		"empty string":  {input: `""`, want: ""},
		"integer":       {input: `42`, want: "42"},
		"negative int":  {input: `-7`, want: "-7"},
		"float":         {input: `3.14`, want: "3.14"},
		"quoted string": {input: `"auto"`, want: "auto"},
		"quoted number": {input: `"100"`, want: "100"},
		"bare true":     {input: `true`, wantErr: true},
		"object":        {input: `{"x":1}`, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			var n numberOrString
			err := json.Unmarshal([]byte(tc.input), &n)
			if tc.wantErr {
				require.Error(t, err, "input %q should be rejected", tc.input)
				return
			}
			require.NoError(t, err, "input %q should decode cleanly", tc.input)
			a.Equal(tc.want, string(n), "input %q decoded to wrong value", tc.input)
		})
	}
}
