package main

import (
	"io"
	"path/filepath"

	"github.com/filipowm/go-unifi/v2/codegen/shared"
)

// ensurePath delegates to shared.EnsurePath. The local wrapper keeps
// utils_test.go (which tests the lowercase name) working without changes.
func ensurePath(p string) (bool, error) { return shared.EnsurePath(p) }

// findProjectRoot delegates to shared.FindProjectRoot.
func findProjectRoot() (string, error) { return shared.FindProjectRoot() }

// findCodegenDir delegates to shared.FindCodegenDir.
func findCodegenDir() (string, error) { return shared.FindCodegenDir() }

// copyWithLimit delegates to shared.CopyWithLimit.
func copyWithLimit(dst io.Writer, src io.Reader, maxSize int64) (int64, error) {
	return shared.CopyWithLimit(dst, src, maxSize)
}

// resolveDir returns dir as-is if absolute, otherwise joined with base.
func resolveDir(base, dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(base, dir)
}
