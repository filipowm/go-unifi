package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// officialPassTimeout bounds the Official frontend subprocess (a `go run` that
// compiles the isolated module and regenerates the surface offline).
const officialPassTimeout = 5 * time.Minute

// runOfficialPass locates the codegen dir, then drives the Official frontend to
// emit the surface into <outDir>/official from the pinned snapshot under
// versionBaseDir. The version is passed explicitly so the generated Go surface is
// deterministic even when multiple snapshots are committed side by side.
func runOfficialPass(versionBaseDir, outDir, officialSpecVersion string, logger Logger) error {
	logger.Infoln("Generating Official API surface...")
	codegenDir, err := findCodegenDir()
	if err != nil {
		return fmt.Errorf("unable to find codegen directory: %w", err)
	}
	specDir := filepath.Join(versionBaseDir, "openapi")
	if err := generateOfficialSurface(codegenDir, specDir, outDir, officialSpecVersion, logger); err != nil {
		return fmt.Errorf("unable to generate Official API surface: %w", err)
	}
	return nil
}

// generateOfficialSurface runs the isolated codegen/official frontend as a
// subprocess. That module carries its own go.mod, so the oapi-codegen / parser
// toolchain stays out of the published root module graph; we shell out rather
// than import it. It emits the Official models, tri-shape wrappers, Client
// interface and mock into <outDir>/official from the committed spec snapshot in
// specDir. officialSpecVersion selects which committed snapshot to generate from
// (empty falls back to the frontend's default: the pin, else newest).
func generateOfficialSurface(codegenDir, specDir, outDir, officialSpecVersion string, logger Logger) error {
	officialDir := filepath.Join(codegenDir, "official")
	if _, err := os.Stat(officialDir); err != nil {
		return fmt.Errorf("locating official frontend at %s: %w", officialDir, err)
	}
	target := filepath.Join(outDir, "official")
	if _, err := ensurePath(target); err != nil {
		return fmt.Errorf("preparing official output dir %s: %w", target, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), officialPassTimeout)
	defer cancel()
	// Args are internal generation paths, not external input (gosec G204).
	cmd := exec.CommandContext(ctx, "go", "run", ".", "-openapi-dir="+specDir, "-out-dir="+target, "-openapi-version="+officialSpecVersion) //nolint:gosec
	cmd.Dir = officialDir
	cmd.Env = append(os.Environ(), "GOFLAGS=-buildvcs=false")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running official frontend: %w\n%s", err, out)
	}
	logger.Infof("Generated Official API surface in %s", target)
	return nil
}
