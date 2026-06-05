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

// TestEmptyStringIntUnmarshal pins emptyStringInt.UnmarshalJSON (TEST-11): the
// empty-string token decodes to 0, a quoted number is unquoted then parsed, a bare
// number is parsed directly, and the malformed cases (non-numeric quoted string,
// unterminated quote, bare non-numeric token) surface the Unquote/Atoi errors
// rather than silently zeroing.
func TestEmptyStringIntUnmarshal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input   string
		want    int
		wantErr bool
	}{
		"empty string":           {input: `""`, want: 0},
		"bare zero":              {input: `0`, want: 0},
		"bare positive":          {input: `42`, want: 42},
		"bare negative":          {input: `-7`, want: -7},
		"quoted number":          {input: `"100"`, want: 100},
		"quoted negative":        {input: `"-3"`, want: -3},
		"quoted non-numeric":     {input: `"abc"`, wantErr: true},
		"bare non-numeric":       {input: `auto`, wantErr: true},
		"unterminated quote":     {input: `"123`, wantErr: true},
		"float is not an int":    {input: `3.14`, wantErr: true},
		"quoted float not int":   {input: `"3.14"`, wantErr: true},
		"quoted empty in quotes": {input: `"  "`, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			var n emptyStringInt
			err := json.Unmarshal([]byte(tc.input), &n)
			if tc.wantErr {
				require.Error(t, err, "input %q should be rejected", tc.input)
				return
			}
			require.NoError(t, err, "input %q should decode cleanly", tc.input)
			a.Equal(tc.want, int(n), "input %q decoded to wrong value", tc.input)
		})
	}
}

// TestEmptyStringIntMarshal pins emptyStringInt.MarshalJSON (TEST-11): nil and the
// zero value both marshal to the empty-string token "" (matching the controller's
// wire form), while a non-zero value marshals to its bare integer.
func TestEmptyStringIntMarshal(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// nil receiver marshals to "".
	var nilPtr *emptyStringInt
	b, err := nilPtr.MarshalJSON()
	require.NoError(t, err)
	a.Equal(`""`, string(b), "nil emptyStringInt must marshal to empty-string token")

	// zero value marshals to "".
	zero := emptyStringInt(0)
	b, err = zero.MarshalJSON()
	require.NoError(t, err)
	a.Equal(`""`, string(b), "zero emptyStringInt must marshal to empty-string token")

	// non-zero marshals to its bare integer.
	nonZero := emptyStringInt(57)
	b, err = nonZero.MarshalJSON()
	require.NoError(t, err)
	a.Equal(`57`, string(b))

	// negative non-zero marshals to its bare integer.
	neg := emptyStringInt(-12)
	b, err = neg.MarshalJSON()
	require.NoError(t, err)
	a.Equal(`-12`, string(b))

	// end-to-end through json.Marshal of an addressable struct (a *holder, as the
	// client marshals request bodies by pointer): the addressable field lets the
	// json package invoke the pointer-receiver MarshalJSON.
	type holder struct {
		N emptyStringInt `json:"n"`
	}
	out, err := json.Marshal(&holder{N: 0})
	require.NoError(t, err)
	a.JSONEq(`{"n":""}`, string(out))

	out, err = json.Marshal(&holder{N: 9})
	require.NoError(t, err)
	a.JSONEq(`{"n":9}`, string(out))
}
