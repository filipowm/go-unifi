package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopyWithLimit pins the G110 decompression-bomb defense: a source larger
// than maxSize must error, the boundary at exactly maxSize must succeed with the
// right byte count, and an under-cap source must copy through.
func TestCopyWithLimit(t *testing.T) {
	t.Parallel()

	const maxSize = 16

	cases := map[string]struct {
		input   string
		wantN   int64
		wantErr string
	}{
		"under cap copies through": {
			input: strings.Repeat("a", maxSize-1),
			wantN: maxSize - 1,
		},
		"exactly at cap succeeds": {
			input: strings.Repeat("a", maxSize),
			wantN: maxSize,
		},
		"over cap trips the bomb guard": {
			input:   strings.Repeat("a", maxSize+1),
			wantN:   maxSize + 1,
			wantErr: "decompression bomb",
		},
		"empty source copies zero bytes": {
			input: "",
			wantN: 0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			var dst bytes.Buffer
			n, err := copyWithLimit(&dst, strings.NewReader(tc.input), maxSize)

			a.Equal(tc.wantN, n, "byte count")
			if tc.wantErr != "" {
				r.ErrorContains(err, tc.wantErr)
				return
			}
			r.NoError(err)
			a.Equal(tc.input, dst.String(), "destination content")
		})
	}
}

// TestCopyWithLimit_BoundaryByteCount asserts the boundary semantics explicitly:
// a source of exactly maxSize bytes returns n == maxSize with no error, while
// maxSize+1 bytes returns the bomb error after reading past the cap.
func TestCopyWithLimit_BoundaryByteCount(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	const maxSize int64 = 8

	atCap := bytes.Repeat([]byte{'x'}, int(maxSize))
	var dst bytes.Buffer
	n, err := copyWithLimit(&dst, bytes.NewReader(atCap), maxSize)
	r.NoError(err)
	a.Equal(maxSize, n)
	a.Equal(int(maxSize), dst.Len())

	overCap := bytes.Repeat([]byte{'x'}, int(maxSize)+1)
	var dst2 bytes.Buffer
	n2, err := copyWithLimit(&dst2, bytes.NewReader(overCap), maxSize)
	r.ErrorContains(err, "decompression bomb")
	a.Greater(n2, maxSize, "must have read past the cap to detect overflow")
}
