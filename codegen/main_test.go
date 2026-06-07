package main

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupLogging asserts the level mapping on the logger RETURNED by
// setupLogging. setupLogging no longer mutates the package-global logger (it
// returns a fresh instance the CLI injects), so this test is now fully
// parallel-safe and shares no state with any other test.
func TestSetupLogging(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		debug, trace bool
		want         logrus.Level
	}{
		"default is info":       {debug: false, trace: false, want: logrus.InfoLevel},
		"debug enables debug":   {debug: true, trace: false, want: logrus.DebugLevel},
		"trace enables trace":   {debug: false, trace: true, want: logrus.TraceLevel},
		"trace wins over debug": {debug: true, trace: true, want: logrus.TraceLevel},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			l := setupLogging(tc.debug, tc.trace)
			assert.Equal(t, tc.want, l.Level)
		})
	}
}

func TestResolveDir(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		base     string
		dir      string
		expected string
	}{
		"absolute path returned as-is": {
			base:     "/home/user",
			dir:      "/absolute/dir",
			expected: "/absolute/dir",
		},
		"relative path joined with base": {
			base:     "/home/user",
			dir:      "relative/dir",
			expected: "/home/user/relative/dir",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, resolveDir(tc.base, tc.dir))
		})
	}
}

// integration tests for the CLI
// these tests require Internet access and/or shell out to `go run .`; gate them
// behind testing.Short() so `go test -short ./codegen/...` runs fully offline.

// skipIfShort skips a live-network / subprocess integration test under -short.
func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping live-network / subprocess integration test in -short mode")
	}
}

func execCli(ctx context.Context, args ...string) (string, error) {
	in := make([]string, 0, 2+len(args))
	in = append(in, "run", ".")
	in = append(in, args...)
	cmd := exec.CommandContext(ctx, "go", in...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestHelpFlag(t *testing.T) {
	t.Parallel()
	skipIfShort(t)

	out, err := execCli(t.Context(), "-h")

	require.Error(t, err)
	assert.Contains(t, out, "Usage: codegen [OPTIONS] version")
}

func TestInvalidFlag(t *testing.T) {
	t.Parallel()
	skipIfShort(t)

	out, err := execCli(t.Context(), "-invalid")

	require.Error(t, err)
	assert.Contains(t, out, "flag provided but not defined: -invalid")
}

func TestDefaultVersion(t *testing.T) {
	t.Parallel()
	skipIfShort(t)

	out, err := execCli(t.Context(), "-version-base-dir", t.TempDir(), "-output-dir", t.TempDir())

	require.NoError(t, err)
	assert.Contains(t, out, "UniFi Controller version")
}

func testGenerate(t *testing.T, opts *options) error {
	t.Helper()

	if opts.logger == nil {
		opts.logger = setupLogging(false, false)
	}
	if opts.versionBaseDir == "" {
		opts.versionBaseDir = t.TempDir()
	}
	if opts.outputDir == "" {
		opts.outputDir = t.TempDir()
	}
	if opts.firmwareUpdateApi == "" {
		opts.firmwareUpdateApi = defaultFirmwareUpdateApi
	}
	return generate(*opts)
}

func TestNonExistentVersion(t *testing.T) {
	t.Parallel()
	skipIfShort(t)

	err := testGenerate(t, &options{version: "1.2.3"})

	require.Error(t, err)
}

func TestInvalidVersion(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	err := testGenerate(t, &options{version: "invalid-version"})

	r.Error(err)
	r.Regexp("(?i)malformed", err.Error())
	r.ErrorContains(err, "invalid-version")
}

func TestGenerateLatest(t *testing.T) {
	t.Parallel()
	skipIfShort(t)
	r := require.New(t)

	root := t.TempDir()
	opts := &options{version: LatestVersionMarker, outputDir: filepath.Join(root, "unifi")}

	err := testGenerate(t, opts)
	r.NoError(err)

	files, err := os.ReadDir(opts.versionBaseDir)
	r.NoError(err)
	assert.NotEmptyf(t, files, "version base dir '%s' should not be empty", opts.versionBaseDir)

	files, err = os.ReadDir(opts.outputDir)
	r.NoError(err)
	assert.NotEmptyf(t, files, "output dir '%s' should not be empty", opts.outputDir)

	// Marker is written beside outDir (at root), not in cwd, and matches version.generated.go.
	marker, err := os.ReadFile(filepath.Join(root, ".unifi-version"))
	r.NoError(err)
	version := strings.TrimSpace(string(marker))
	r.NotEmpty(version)

	versionGo, err := os.ReadFile(filepath.Join(opts.outputDir, "version.generated.go"))
	r.NoError(err)
	assert.Contains(t, string(versionGo), `"`+version+`"`, "marker must match version.generated.go")

	// Assert that generate() commits the Official OpenAPI spec snapshot.
	specGlob := filepath.Join(opts.versionBaseDir, "openapi", "integration-*.json")
	specFiles, err := filepath.Glob(specGlob)
	r.NoError(err)
	r.Len(specFiles, 1, "exactly one Official OpenAPI spec snapshot must be committed under %s/openapi/", opts.versionBaseDir)

	specData, err := os.ReadFile(specFiles[0])
	r.NoError(err)
	r.True(json.Valid(specData), "snapshot must be valid JSON")

	var spec struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	r.NoError(json.Unmarshal(specData, &spec))
	r.Truef(strings.HasPrefix(spec.OpenAPI, "3.1"), "expected OpenAPI 3.1.x, got %q", spec.OpenAPI)
	r.NotEmpty(spec.Info.Version, "spec info.version must be non-empty")

	// The version in the filename must match the spec's info.version.
	base := filepath.Base(specFiles[0])
	filenameVer := strings.TrimSuffix(strings.TrimPrefix(base, "integration-"), ".json")
	r.Equal(filenameVer, spec.Info.Version, "spec info.version must match the snapshot filename")
}

// TestDownloadGenerationInputs_SkipsLegacyDownloadWhenSnapshotComplete is the
// primary regression guard for #124: when the committed frozen snapshot is
// complete (sentinel present), downloadGenerationInputs must return the snapshot
// dir immediately without attempting any network download.
func TestDownloadGenerationInputs_SkipsLegacyDownloadWhenSnapshotComplete(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	baseDir := t.TempDir()
	internalVer, err := version.NewVersion("9.5.21")
	r.NoError(err)

	// Stage a complete frozen legacy field snapshot (JSON + sentinel).
	frozenDir := legacyFieldsDir(baseDir, internalVer)
	r.NoError(os.MkdirAll(frozenDir, 0o755))
	r.NoError(os.WriteFile(filepath.Join(frozenDir, "Device.json"), []byte(`{"k":"v"}`), 0o600))
	r.NoError(os.WriteFile(filepath.Join(frozenDir, extractCompleteSentinel), nil, 0o600))

	// Stage the Official spec snapshot so the official download is also skipped.
	specPath := officialSpecSnapshotPath(baseDir, internalVer)
	r.NoError(os.MkdirAll(filepath.Dir(specPath), 0o755))
	r.NoError(os.WriteFile(specPath, []byte(`{"openapi":"3.1.0"}`), 0o600))

	// A URL that the host guard rejects — proves no legacy download was attempted.
	badURL, _ := url.Parse("https://evil.example.com/bad.deb")
	internalVersion := NewUnifiVersion(internalVer, badURL)
	officialVersion := NewUnifiVersion(internalVer, badURL)

	got, err := downloadGenerationInputs(internalVersion, officialVersion, baseDir, setupLogging(false, false))
	r.NoError(err, "must skip download when frozen snapshot is complete")
	r.Equal(frozenDir, got, "must return the frozen snapshot dir")
}

// TestDownloadGenerationInputs_FallsThroughToDownloadWhenNoSentinel covers the
// inverse: a frozen dir missing the extraction sentinel is not accepted as
// complete, and the download branch is entered. The host-guard error proves it.
func TestDownloadGenerationInputs_FallsThroughToDownloadWhenNoSentinel(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	baseDir := t.TempDir()
	internalVer, err := version.NewVersion("9.5.21")
	r.NoError(err)

	// Stage the dir WITHOUT the sentinel — simulate a partial/crashed prior run.
	frozenDir := legacyFieldsDir(baseDir, internalVer)
	r.NoError(os.MkdirAll(frozenDir, 0o755))
	r.NoError(os.WriteFile(filepath.Join(frozenDir, "Device.json"), []byte(`{"k":"v"}`), 0o600))

	// A URL that the host guard rejects — trips immediately, proving the branch was entered.
	badURL, _ := url.Parse("https://evil.example.com/bad.deb")
	internalVersion := NewUnifiVersion(internalVer, badURL)
	officialVersion := NewUnifiVersion(internalVer, badURL)

	_, err = downloadGenerationInputs(internalVersion, officialVersion, baseDir, setupLogging(false, false))
	r.Error(err, "must attempt download when sentinel is absent")
	r.ErrorContains(err, "not an allowed Ubiquiti host")
}

func TestGenerateDownloadOnly(t *testing.T) {
	t.Parallel()
	skipIfShort(t)
	r := require.New(t)

	opts := &options{version: LatestVersionMarker, downloadOnly: true}

	err := testGenerate(t, opts)
	r.NoError(err)

	files, err := os.ReadDir(opts.versionBaseDir)
	r.NoError(err)
	assert.NotEmptyf(t, files, "version base dir '%s' should not be empty", opts.versionBaseDir)

	files, err = os.ReadDir(opts.outputDir)
	r.NoError(err) // test generated dir
	assert.Emptyf(t, files, "output dir '%s' should be empty", opts.outputDir)
}
