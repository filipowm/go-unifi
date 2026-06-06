package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsurePath pins ensurePath's three outcomes: an existing directory is a
// no-op returning (false,nil); a missing path is created returning (true,nil);
// a path that is a FILE returns the "isn't a directory" error.
func TestEnsurePath(t *testing.T) {
	t.Parallel()

	t.Run("existing dir is a no-op", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		created, err := ensurePath(dir)
		require.NoError(t, err)
		assert.False(t, created, "an existing directory must not report creation")
	})

	t.Run("missing path is created", func(t *testing.T) {
		t.Parallel()
		target := filepath.Join(t.TempDir(), "nested", "deep")
		created, err := ensurePath(target)
		require.NoError(t, err)
		assert.True(t, created, "a missing path must report creation")
		info, statErr := os.Stat(target)
		require.NoError(t, statErr)
		assert.True(t, info.IsDir(), "the created path must be a directory")
	})

	t.Run("file path is not a directory", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(t.TempDir(), "afile")
		require.NoError(t, os.WriteFile(file, []byte("x"), 0o644)) //nolint:gosec
		created, err := ensurePath(file)
		require.ErrorContains(t, err, "isn't a directory")
		assert.False(t, created)
	})
}

// TestFindProjectRoot chdirs into a synthetic tree whose only go.mod sits at the
// temp root and asserts findProjectRoot walks up to it. NOT parallel: it mutates
// process-global cwd via t.Chdir (restored automatically by the test framework).
func TestFindProjectRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/synthetic\n"), 0o644)) //nolint:gosec

	sub := filepath.Join(root, "a", "b")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	// Start two levels below the go.mod so the upward walk has to climb.
	t.Chdir(sub)

	got, err := findProjectRoot()
	require.NoError(t, err)

	// macOS /var is a symlink to /private/var, so compare resolved paths.
	wantResolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	gotResolved, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	assert.Equal(t, wantResolved, gotResolved, "findProjectRoot must return the dir holding go.mod")
}

// TestFindCodegenDir asserts findCodegenDir returns the project root (as found
// by findProjectRoot) with "codegen" joined onto it. NOT parallel: t.Chdir.
func TestFindCodegenDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/synthetic\n"), 0o644)) //nolint:gosec
	t.Chdir(root)

	got, err := findCodegenDir()
	require.NoError(t, err)

	// findCodegenDir == filepath.Join(findProjectRoot(), "codegen"). Derive the
	// expectation from findProjectRoot itself so symlink resolution (macOS /var ->
	// /private/var) matches without re-stat-ing a not-yet-created codegen dir.
	gotRoot, err := findProjectRoot()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(gotRoot, "codegen"), got, "findCodegenDir must join \"codegen\" onto the project root")
	assert.Equal(t, "codegen", filepath.Base(got))
}

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
