// Package shared holds the minimal symbols used by both the root codegen
// (package main) and the internal generation engine (codegen/internal). Keeping
// them here avoids the package-main import barrier.
package shared

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Logger is the minimal logging surface the generation pipeline depends on. It
// is backed by log/slog (see NewSlogLogger), so production code can inject a real
// logger while tests inject their own instance with a capturing handler,
// asserting output in parallel without mutating shared state.
type Logger interface {
	Tracef(format string, args ...any)
	Debugf(format string, args ...any)
	Debugln(args ...any)
	Infof(format string, args ...any)
	Infoln(args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Error(args ...any)
}

// OrDefaultLogger returns logger if non-nil, otherwise calls fallback(). The
// fallback is a function rather than a value so callers can avoid constructing
// the default unless it is actually needed.
func OrDefaultLogger(logger Logger, fallback func() Logger) Logger {
	if logger == nil {
		return fallback()
	}
	return logger
}

// EnsurePath checks if a path exists and is a directory; if not it creates it.
// Returns true if the directory was freshly created.
func EnsurePath(path string) (bool, error) {
	targetInfo, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if err = os.MkdirAll(path, 0o755); err != nil {
			return false, err
		}
		return true, nil
	}
	if !targetInfo.IsDir() {
		return false, fmt.Errorf("%s isn't a directory", path)
	}
	return false, nil
}

// FindProjectRoot walks up from the working directory until it finds a go.mod file.
func FindProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd { // reached the filesystem/volume root
			break
		}
		wd = parent
	}
	return "", errors.New("unable to find project root")
}

// FindCodegenDir returns the codegen/ subdirectory of the project root.
func FindCodegenDir() (string, error) {
	root, err := FindProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "codegen"), nil
}

// CopyWithLimit copies src to dst, capping total bytes to guard against
// decompression bombs (gosec G110). Returns the number of bytes copied and an
// error if the cap is exceeded.
func CopyWithLimit(dst io.Writer, src io.Reader, maxSize int64) (int64, error) {
	n, err := io.CopyN(dst, src, maxSize+1) // read one past the cap to detect overflow
	if errors.Is(err, io.EOF) {
		err = nil // source smaller than the cap — fine
	}
	if err != nil {
		return n, err
	}
	if n > maxSize {
		return n, fmt.Errorf("decompressed size exceeds %d bytes (possible decompression bomb)", maxSize)
	}
	return n, nil
}
