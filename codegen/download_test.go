package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
	"github.com/xor-gate/ar"
)

// Helper function to create a temporary zip file with given entries. 'entries' maps file names to their content.
func createTempZipFile(t *testing.T, entries map[string]string) string {
	t.Helper()
	tempDir := t.TempDir()
	tempFileName := filepath.Join(tempDir, "test.zip")
	tempFile, err := os.Create(tempFileName)
	require.NoError(t, err, "Failed to create temp zip file")
	// We need to truncate and write zip contents
	w := zip.NewWriter(tempFile)
	for name, content := range entries {
		f, err := w.Create(name)
		require.NoError(t, err, "Failed to add entry %s", name)
		_, err = f.Write([]byte(content))
		require.NoError(t, err, "Failed to write content for %s", name)
	}
	err = w.Close()
	require.NoError(t, err, "Failed to close zip writer")
	err = tempFile.Close()
	require.NoError(t, err, "Failed to close temp file")
	return tempFile.Name()
}

// buildAceJar builds an in-memory zip (mimicking ace.jar) holding the given
// api/fields/*.json entries.
func buildAceJar(t *testing.T, fields map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range fields {
		f, err := zw.Create(name)
		require.NoError(t, err, "creating zip entry %s", name)
		_, err = f.Write([]byte(content))
		require.NoError(t, err, "writing zip entry %s", name)
	}
	require.NoError(t, zw.Close(), "closing zip writer")
	return buf.Bytes()
}

// buildDataTarXz wraps the given files in a tar archive and xz-compresses it,
// mimicking the data.tar.xz member of a .deb. The map keys are tar entry names.
func buildDataTarXz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Typeflag: tar.TypeReg,
			Mode:     0o644,
			Size:     int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr), "writing tar header %s", name)
		_, err := tw.Write(content)
		require.NoError(t, err, "writing tar body %s", name)
	}
	require.NoError(t, tw.Close(), "closing tar writer")

	var xzBuf bytes.Buffer
	xw, err := xz.NewWriter(&xzBuf)
	require.NoError(t, err, "creating xz writer")
	_, err = xw.Write(tarBuf.Bytes())
	require.NoError(t, err, "writing xz body")
	require.NoError(t, xw.Close(), "closing xz writer")
	return xzBuf.Bytes()
}

// buildDeb builds an ar archive (the .deb container) holding the supplied
// members. Use it to construct a tiny ar(data.tar.xz(ace.jar)) fixture or a
// malformed one missing the data.tar.xz member.
func buildDeb(t *testing.T, members map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	aw := ar.NewWriter(&buf)
	require.NoError(t, aw.WriteGlobalHeader(), "writing ar global header")
	for name, content := range members {
		hdr := &ar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0o644,
		}
		require.NoError(t, aw.WriteHeader(hdr), "writing ar header %s", name)
		_, err := aw.Write(content)
		require.NoError(t, err, "writing ar body %s", name)
	}
	return buf.Bytes()
}

// buildControllerDeb assembles a full ar(data.tar.xz(tar(ace.jar))) fixture
// whose ace.jar contains the given api/fields/*.json entries.
func buildControllerDeb(t *testing.T, fields map[string]string) []byte {
	t.Helper()
	aceJar := buildAceJar(t, fields)
	dataTarXz := buildDataTarXz(t, map[string][]byte{
		"./usr/lib/unifi/lib/ace.jar": aceJar,
	})
	return buildDeb(t, map[string][]byte{"data.tar.xz": dataTarXz})
}

// buildOfficialSpecTar wraps the given spec bytes in a plain tar at the
// integration.json path inside a UniFi OS Server package's data tar.
func buildOfficialSpecTar(t *testing.T, spec []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: officialSpecTarPath, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(spec))}))
	_, err := tw.Write(spec)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	return buf.Bytes()
}

// buildUosDeb assembles a full ar(data.tar.xz(tar(integration.json))) fixture
// mimicking the UniFi OS Server package.
func buildUosDeb(t *testing.T, spec []byte) []byte {
	t.Helper()
	dataTarXz := buildDataTarXz(t, map[string][]byte{officialSpecTarPath: spec})
	return buildDeb(t, map[string][]byte{"data.tar.xz": dataTarXz})
}

// TestExtractOfficialSpec_HappyPath returns the integration.json bytes verbatim.
func TestExtractOfficialSpec_HappyPath(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	spec := []byte(`{"openapi":"3.1.0","info":{"version":"10.1.78"}}`)
	got, err := extractOfficialSpec(bytes.NewReader(buildOfficialSpecTar(t, spec)))
	r.NoError(err)
	a.Equal(spec, got, "spec must be returned byte-for-byte")
}

// TestExtractOfficialSpec_NotFound returns the sentinel when the tar lacks it.
func TestExtractOfficialSpec_NotFound(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	r.NoError(tw.WriteHeader(&tar.Header{Name: "./usr/lib/unifi/other.json", Typeflag: tar.TypeReg, Mode: 0o644, Size: 2}))
	_, err := tw.Write([]byte("{}"))
	r.NoError(err)
	r.NoError(tw.Close())

	_, err = extractOfficialSpec(bytes.NewReader(buf.Bytes()))
	r.ErrorIs(err, errOfficialSpecNotFound)
}

// TestExtractOfficialSpec_Oversize trips the decompression-bomb cap.
func TestExtractOfficialSpec_Oversize(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	oversize := bytes.Repeat([]byte("a"), maxOpenAPISpecSize+1)
	_, err := extractOfficialSpec(bytes.NewReader(buildOfficialSpecTar(t, oversize)))
	r.Error(err)
	r.ErrorContains(err, "decompression bomb")
}

// TestWriteOfficialSpecSnapshot_AtomicWrite writes the snapshot into a not-yet
// existing nested dir and leaves no temp file behind.
func TestWriteOfficialSpecSnapshot_AtomicWrite(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	spec := []byte(`{"openapi":"3.1.0"}`)
	outPath := filepath.Join(t.TempDir(), "openapi", "integration-10.1.78.json")
	r.NoError(writeOfficialSpecSnapshot(spec, outPath))

	got, err := os.ReadFile(outPath)
	r.NoError(err)
	a.Equal(spec, got)

	entries, err := os.ReadDir(filepath.Dir(outPath))
	r.NoError(err)
	for _, e := range entries {
		a.NotContains(e.Name(), ".tmp-", "temp snapshot file must not be left behind")
	}
}

// TestWriteOfficialSpecSnapshot_InvalidJSON rejects non-JSON content.
func TestWriteOfficialSpecSnapshot_InvalidJSON(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	outPath := filepath.Join(t.TempDir(), "integration.json")
	err := writeOfficialSpecSnapshot([]byte("not json"), outPath)
	r.Error(err)
	r.ErrorContains(err, "not valid JSON")
	_, statErr := os.Stat(outPath)
	r.ErrorIs(statErr, os.ErrNotExist, "invalid spec must not be published")
}

// TestDownloadAndExtractOfficialSpec_FullChainOffline drives the full
// download -> ar -> xz -> tar -> integration.json -> snapshot chain offline.
func TestDownloadAndExtractOfficialSpec_FullChainOffline(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	spec := []byte(`{"openapi":"3.1.0","info":{"version":"10.1.78"}}`)
	deb := buildUosDeb(t, spec)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	outPath := filepath.Join(t.TempDir(), "openapi", "integration-10.1.78.json")
	err = DownloadAndExtractOfficialSpec(context.Background(), server.Client(), *u, outPath)
	r.NoError(err)

	got, err := os.ReadFile(outPath)
	r.NoError(err)
	a.Equal(spec, got, "snapshot must be byte-for-byte deterministic")
}

// TestDownloadAndExtractOfficialSpec_NotFoundOffline surfaces the sentinel for a
// package that carries no integration.json and writes no snapshot.
func TestDownloadAndExtractOfficialSpec_NotFoundOffline(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	dataTarXz := buildDataTarXz(t, map[string][]byte{"./usr/lib/unifi/other.json": []byte("{}")})
	deb := buildDeb(t, map[string][]byte{"data.tar.xz": dataTarXz})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	outPath := filepath.Join(t.TempDir(), "openapi", "integration-10.1.78.json")
	err = DownloadAndExtractOfficialSpec(context.Background(), server.Client(), *u, outPath)
	r.ErrorIs(err, errOfficialSpecNotFound)
	_, statErr := os.Stat(outPath)
	r.ErrorIs(statErr, os.ErrNotExist)
}

// TestDownloadOfficialSpecSnapshot_SkipsIfSnapshotExists verifies that
// downloadOfficialSpecSnapshot skips the network download entirely when the
// committed snapshot file is already present, making `go generate` fully offline
// when both the legacy fields and the Official spec snapshots are committed.
func TestDownloadOfficialSpecSnapshot_SkipsIfSnapshotExists(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// Write a pre-existing snapshot file so downloadOfficialSpecSnapshot finds it.
	specPath := filepath.Join(t.TempDir(), "integration-10.1.78.json")
	r.NoError(os.WriteFile(specPath, []byte(`{"openapi":"3.1.0"}`), 0o600))

	// Use a URL that would be rejected by the host guard — if the function
	// mistakenly tried to download, it would return an error here.
	badURL, _ := url.Parse("https://evil.example.com/bad.deb")
	logger := setupLogging(false, false)

	err := downloadOfficialSpecSnapshot(context.Background(), http.DefaultClient, *badURL, specPath, logger)
	r.NoError(err, "must skip download when snapshot already exists")

	// The pre-existing file must be intact (not overwritten).
	data, readErr := os.ReadFile(specPath)
	r.NoError(readErr)
	r.JSONEq(`{"openapi":"3.1.0"}`, string(data), "existing snapshot must not be overwritten")
}

// TestDownloadOfficialSpecSnapshot_OldVersionSkips pins the <10.1.78 regression-safety
// path in downloadOfficialSpecSnapshot: a UOS package without integration.json must
// yield nil (generation continues) and write no snapshot. This tests the httptest
// seam introduced so the non-fatal swallow-and-continue branch is fully covered.
func TestDownloadOfficialSpecSnapshot_OldVersionSkips(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// Build a UOS deb without integration.json, mimicking pre-10.1.78 packages.
	dataTarXz := buildDataTarXz(t, map[string][]byte{"./usr/lib/unifi/other.json": []byte("{}")})
	deb := buildDeb(t, map[string][]byte{"data.tar.xz": dataTarXz})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	specURL, err := url.Parse(server.URL)
	r.NoError(err)

	specPath := filepath.Join(t.TempDir(), "integration-9.5.21.json")
	logger := setupLogging(false, false)

	// downloadOfficialSpecSnapshot must return nil (non-fatal skip), not propagate errOfficialSpecNotFound.
	err = downloadOfficialSpecSnapshot(context.Background(), server.Client(), *specURL, specPath, logger)
	r.NoError(err, "pre-Official-API package must skip without error; generation must continue")

	// No snapshot file must be written for old packages.
	_, statErr := os.Stat(specPath)
	r.ErrorIs(statErr, os.ErrNotExist, "no snapshot must be written for packages lacking integration.json")
}

// TestDownloadAndExtractOfficialSpec_NotFoundHTTP asserts the non-200 branch.
func TestDownloadAndExtractOfficialSpec_NotFoundHTTP(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	err = DownloadAndExtractOfficialSpec(context.Background(), server.Client(), *u, filepath.Join(t.TempDir(), "integration.json"))
	r.Error(err)
	r.ErrorContains(err, "HTTP404")
}

// TestDownloadAndExtractOfficialSpec_RejectsBadURL trips the host/scheme guard
// before any request and writes nothing.
func TestDownloadAndExtractOfficialSpec_RejectsBadURL(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	u, err := url.Parse("http://dl.ui.com/unifi/x.deb") // http, not https
	r.NoError(err)

	outPath := filepath.Join(t.TempDir(), "integration.json")
	err = DownloadAndExtractOfficialSpec(context.Background(), http.DefaultClient, *u, outPath)
	r.Error(err)
	r.ErrorContains(err, "must use https")
	_, statErr := os.Stat(outPath)
	r.ErrorIs(statErr, os.ErrNotExist)
}

// TestDownloadAndExtractOfficialSpec_ContextCancelled aborts on a pre-cancelled
// context and leaves no snapshot.
func TestDownloadAndExtractOfficialSpec_ContextCancelled(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	deb := buildUosDeb(t, []byte(`{"openapi":"3.1.0"}`))
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	outPath := filepath.Join(t.TempDir(), "integration.json")
	err = DownloadAndExtractOfficialSpec(ctx, server.Client(), *u, outPath)
	r.ErrorIs(err, context.Canceled)
	_, statErr := os.Stat(outPath)
	r.ErrorIs(statErr, os.ErrNotExist)
}

// TestDownloadAndExtractOfficialSpec_Live performs the REAL fetch+extract from
// dl.ui.com to prove the spec source end-to-end. Gated behind -short so CI/unit
// runs stay offline; the full quality gate (no -short) exercises it live.
func TestDownloadAndExtractOfficialSpec_Live(t *testing.T) {
	t.Parallel()
	skipIfShort(t)
	r := require.New(t)
	a := assert.New(t)

	uv, err := NewUnifiVersionProvider(defaultFirmwareUpdateApi).Latest()
	r.NoError(err)
	specURL, err := uv.OfficialSpecURL()
	r.NoError(err)

	outPath := filepath.Join(t.TempDir(), "openapi", "integration-"+uv.Version.String()+".json")
	err = DownloadAndExtractOfficialSpec(context.Background(), http.DefaultClient, *specURL, outPath)
	r.NoError(err, "live fetch of %s must succeed", specURL.String())

	data, err := os.ReadFile(outPath)
	r.NoError(err)
	var spec struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	r.NoError(json.Unmarshal(data, &spec))
	a.Truef(strings.HasPrefix(spec.OpenAPI, "3.1"), "expected OpenAPI 3.1, got %q", spec.OpenAPI)
	a.NotEmpty(spec.Info.Version, "spec info.version must be present")
}

// Test when the output directory already exists AND carries the completion
// sentinel: DownloadAndExtract treats it as already-extracted and performs no
// download. The URL is a non-loopback dummy that would be rejected by the host
// guard if the code mistakenly tried to fetch it, so reaching NoError proves the
// download was skipped.
func TestDownloadAndExtract_WithCompletedDirectory(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	tempDir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(tempDir, extractCompleteSentinel), nil, 0o600))
	testURL, _ := url.Parse("http://example.com/test.deb")

	err := DownloadAndExtract(context.Background(), http.DefaultClient, *testURL, tempDir)

	r.NoError(err, "Expected no error / no download when sentinel present")
}

// Test that an existing-but-sentinel-less directory (a partial/crashed prior
// run) is NOT treated as complete: the code proceeds to validate+fetch,
// and here the disallowed host trips the guard, proving the dir was not
// silently accepted.
func TestDownloadAndExtract_PartialDirectoryReExtracts(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	tempDir := t.TempDir()
	// Simulate leftover junk from a crashed extract, but no sentinel.
	r.NoError(os.WriteFile(filepath.Join(tempDir, "Partial.json"), []byte("{}"), 0o600))
	testURL, _ := url.Parse("https://example.com/test.deb")

	err := DownloadAndExtract(context.Background(), http.DefaultClient, *testURL, tempDir)

	r.Error(err, "partial dir must not be accepted as complete")
	r.ErrorContains(err, "not an allowed Ubiquiti host")
}

// // Test when output path is not a directory.
func TestDownloadAndExtract_PathNotDirectory(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, "dummy")
	_, err := os.Create(tempFilePath)
	r.NoError(err, "Failed to create temp file")
	testURL, _ := url.Parse("http://example.com/test.deb")

	err = DownloadAndExtract(context.Background(), http.DefaultClient, *testURL, tempFilePath)

	r.Error(err, "Expected error because tempFilePath is not a directory")
	r.ErrorContains(err, tempFilePath+" isn't a directory")
}

// // Test extractJSON when the jar file cannot be opened.
func TestExtractJSON_OpenJarError(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	err := extractJSON("nonexisting.jar", t.TempDir())

	r.Error(err)
	r.ErrorContains(err, "unable to open jar")
}

// Test extractJSON with a valid zip file that contains a JSON file under api/fields/ and no Setting.json (so splitting is skipped).
func TestExtractJSON_NoSettings(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	tempDir := t.TempDir()
	jarFile := createTempZipFile(t, map[string]string{"api/fields/dummy.json": "{\"key\": \"value\"}"})

	err := extractJSON(jarFile, tempDir)
	r.NoError(err)

	// Check that dummy.json has been extracted
	expectedPath := filepath.Join(tempDir, "dummy.json")
	data, err := os.ReadFile(expectedPath)
	r.NoError(err, "Expected file %s to exist", expectedPath)
	r.JSONEq("{\"key\": \"value\"}", string(data), "Extracted file content mismatch")
}

// Test extractJSON with Setting.json present, so that it splits settings into individual files.
func TestExtractJSON_WithSettings(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	tempDir := t.TempDir()
	entries := map[string]string{"api/fields/Setting.json": "{\"foo\": {\"bar\": 1}}"}
	jarFile := createTempZipFile(t, entries)

	err := extractJSON(jarFile, tempDir)
	r.NoError(err)

	// Check that the split settings file exists
	settingFile := filepath.Join(tempDir, "SettingFoo.json")
	data, err := os.ReadFile(settingFile)
	r.NoError(err)
	r.Contains(string(data), "bar")
}

// Test sanitizeExtractedPath with valid input.
func TestSanitizeExtractedPath_Valid(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	tempDir := t.TempDir()
	filePath := "api/fields/dummy.json"

	result, err := sanitizeExtractedPath(filePath, tempDir)
	r.NoError(err, "Expected nil error from sanitizeExtractedPath")

	expExpected := filepath.Join(tempDir, "dummy.json")
	absExpected, err := filepath.Abs(expExpected)
	r.NoError(err, "Failed to get abs path")
	a.Equal(absExpected, result, "Sanitized path mismatch")
}

// Test splitSettingsFile returns nil (no-op) when Setting.json is absent.
func TestSplitSettingsFile_NoSettingFile(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	err := splitSettingsFile(t.TempDir())
	r.NoError(err, "Expected no error when Setting.json does not exist")
}

// Test splitSettingsFile writes one Setting<Camel>.json file per top-level key.
func TestSplitSettingsFile_SplitsPerSetting(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)
	tempDir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(tempDir, "Setting.json"),
		[]byte(`{"foo_bar": {"x": 1}, "baz": {"y": "z"}}`),
		0o600,
	)
	r.NoError(err, "Failed to write Setting.json")

	err = splitSettingsFile(tempDir)
	r.NoError(err)

	fooData, err := os.ReadFile(filepath.Join(tempDir, "SettingFooBar.json"))
	r.NoError(err, "Expected SettingFooBar.json to exist")
	a.JSONEq(`{"x": 1}`, string(fooData), "SettingFooBar.json content mismatch")

	bazData, err := os.ReadFile(filepath.Join(tempDir, "SettingBaz.json"))
	r.NoError(err, "Expected SettingBaz.json to exist")
	a.JSONEq(`{"y": "z"}`, string(bazData), "SettingBaz.json content mismatch")
}

// Test splitSettingsFile surfaces an unmarshal error on malformed Setting.json.
func TestSplitSettingsFile_InvalidJSON(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "Setting.json"), []byte("not json"), 0o600)
	r.NoError(err, "Failed to write Setting.json")

	err = splitSettingsFile(tempDir)
	r.Error(err)
	r.ErrorContains(err, "unable to unmarshal settings")
}

// Test extractJSON with invalid Setting.json content, expecting an unmarshal error.
func TestExtractJSON_InvalidSettings(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	tempDir := t.TempDir()
	jarFile := createTempZipFile(t, map[string]string{"api/fields/Setting.json": "invalid json"})

	err := extractJSON(jarFile, tempDir)

	r.Error(err)
	r.ErrorContains(err, "unable to unmarshal settings")
}

// TestSanitizeExtractedPath_Traversal pins the zip-slip / path-traversal guard.
// filepath.Base() strips the directory components so traversal-style names are
// mitigated by being re-anchored inside destinationDir; the test documents that
// intended behavior and exercises the !HasPrefix 'invalid file path' branch.
func TestSanitizeExtractedPath_Traversal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		// the entry name as it appears inside the archive
		filePath string
		// when set, the produced path must equal filepath.Join(destDir, wantBase)
		wantBase string
		// when set, the call must error and contain this substring
		wantErr string
	}{
		"parent traversal is re-anchored in dir": {
			filePath: "../../etc/passwd",
			wantBase: "passwd",
		},
		"absolute path is re-anchored in dir": {
			filePath: "/etc/shadow",
			wantBase: "shadow",
		},
		"dot-dot only collapses to dir itself -> rejected": {
			// filepath.Base("..") == "..", so Join(dir, "..") escapes the dir
			// and trips the HasPrefix guard with the 'invalid file path' error.
			filePath: "..",
			wantErr:  "invalid file path",
		},
		"plain name stays in dir": {
			filePath: "Device.json",
			wantBase: "Device.json",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			destDir := t.TempDir()
			got, err := sanitizeExtractedPath(tc.filePath, destDir)

			if tc.wantErr != "" {
				r.ErrorContains(err, tc.wantErr)
				return
			}

			r.NoError(err)
			absExpected, absErr := filepath.Abs(filepath.Join(destDir, tc.wantBase))
			r.NoError(absErr)
			a.Equal(absExpected, got)
			// The sanitized result must always stay inside the destination dir.
			absDest, absErr := filepath.Abs(destDir)
			r.NoError(absErr)
			a.True(strings.HasPrefix(got, absDest), "sanitized path %q escaped dest %q", got, absDest)
		})
	}
}

// TestSanitizeExtractedPath_SiblingPrefix ensures a sibling directory that
// shares the destination dir's name prefix (e.g. /tmp/dest vs /tmp/dest-evil)
// cannot be reached: because filepath.Base re-anchors the file inside destDir,
// the result is always within destDir.
func TestSanitizeExtractedPath_SiblingPrefix(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	base := t.TempDir()
	destDir := filepath.Join(base, "dest")
	r.NoError(os.Mkdir(destDir, 0o755))
	sibling := filepath.Join(base, "dest-evil")
	r.NoError(os.Mkdir(sibling, 0o755))

	// An attacker-style name pointing at the sibling dir.
	got, err := sanitizeExtractedPath("../dest-evil/payload.json", destDir)
	r.NoError(err)

	absDest, err := filepath.Abs(destDir)
	r.NoError(err)
	absSibling, err := filepath.Abs(sibling)
	r.NoError(err)

	a.Equal(filepath.Join(absDest, "payload.json"), got, "must be re-anchored inside dest")
	a.False(strings.HasPrefix(got, absSibling+string(filepath.Separator)), "must not land in sibling dir")
}

// TestExtractZipEntry_Oversize crafts a zip entry larger than maxJSONSize and
// asserts the decompression-bomb error propagates out of extractJSON as
// 'unable to write JSON file'.
func TestExtractZipEntry_Oversize(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	tempDir := t.TempDir()

	oversize := strings.Repeat("a", maxJSONSize+1)
	jarFile := createTempZipFile(t, map[string]string{"api/fields/Big.json": oversize})

	err := extractJSON(jarFile, tempDir)
	r.Error(err)
	r.ErrorContains(err, "unable to write JSON file")
	r.ErrorContains(err, "decompression bomb")
}

// TestOpenDebDataTar_MissingMember feeds an ar archive without a data.tar.xz
// member and asserts the 'unable to find .deb data file' error.
func TestOpenDebDataTar_MissingMember(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	deb := buildDeb(t, map[string][]byte{"control.tar.gz": []byte("nope")})

	_, err := openDebDataTar(bytes.NewReader(deb))
	r.Error(err)
	r.ErrorContains(err, "unable to find .deb data file")
}

// TestOpenDebDataTar_HappyPath decodes a well-formed ar(data.tar.xz) stream and
// returns a reader over the decompressed tar contents.
func TestOpenDebDataTar_HappyPath(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	dataTarXz := buildDataTarXz(t, map[string][]byte{"./hello.txt": []byte("hi")})
	deb := buildDeb(t, map[string][]byte{"data.tar.xz": dataTarXz})

	reader, err := openDebDataTar(bytes.NewReader(deb))
	r.NoError(err)

	tr := tar.NewReader(reader)
	hdr, err := tr.Next()
	r.NoError(err)
	r.Equal("./hello.txt", hdr.Name)
}

// TestExtractAceJar_MissingJar walks a tar stream that does not contain ace.jar
// and asserts the 'unable to find ace.jar' error branch.
func TestExtractAceJar_MissingJar(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	content := []byte("not a jar")
	r.NoError(tw.WriteHeader(&tar.Header{Name: "./usr/lib/unifi/lib/other.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))}))
	_, err := tw.Write(content)
	r.NoError(err)
	r.NoError(tw.Close())

	_, err = extractAceJar(bytes.NewReader(tarBuf.Bytes()), t.TempDir())
	r.Error(err)
	r.ErrorContains(err, "unable to find ace.jar")
}

// TestExtractAceJar_HappyPath finds ace.jar inside a tar stream and writes it to
// outputDir, returning the created file's path.
func TestExtractAceJar_HappyPath(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	jarBytes := buildAceJar(t, map[string]string{"api/fields/Device.json": `{"k":"v"}`})
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	r.NoError(tw.WriteHeader(&tar.Header{Name: "./usr/lib/unifi/lib/ace.jar", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(jarBytes))}))
	_, err := tw.Write(jarBytes)
	r.NoError(err)
	r.NoError(tw.Close())

	outputDir := t.TempDir()
	jarPath, err := extractAceJar(bytes.NewReader(tarBuf.Bytes()), outputDir)
	r.NoError(err)
	a.Equal(filepath.Join(outputDir, "ace.jar"), jarPath)

	written, err := os.ReadFile(jarPath)
	r.NoError(err)
	a.Equal(jarBytes, written)
}

// TestDownloadJar_HappyPath drives downloadJar against an httptest server
// serving a tiny hand-built ar(data.tar.xz(ace.jar)) fixture, fully offline.
func TestDownloadJar_HappyPath(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	deb := buildControllerDeb(t, map[string]string{"api/fields/Device.json": `{"k":"v"}`})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	outputDir := t.TempDir()
	jarPath, err := downloadJar(context.Background(), server.Client(), *u, outputDir)
	r.NoError(err)
	a.Equal(filepath.Join(outputDir, "ace.jar"), jarPath)

	_, err = os.Stat(jarPath)
	r.NoError(err)
}

// TestDownloadJar_NotFound asserts the non-200 branch returns the HTTP%d error.
func TestDownloadJar_NotFound(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	_, err = downloadJar(context.Background(), server.Client(), *u, t.TempDir())
	r.Error(err)
	r.ErrorContains(err, "HTTP404")
}

// TestDownloadJar_NilClientDefaults verifies the nil-client default does not
// panic and still issues the request (against a local server here).
func TestDownloadJar_NilClientDefaults(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	// nil -> a default client with a timeout; the local httptest URL is reachable offline.
	_, err = downloadJar(context.Background(), nil, *u, t.TempDir())
	r.Error(err)
	r.ErrorContains(err, "HTTP404")
}

// TestDownloadAndExtract_FullChainOffline exercises the full
// download -> ar -> xz -> tar -> ace.jar -> extractJSON chain offline using the
// injected client seam and a tiny in-memory .deb fixture.
func TestDownloadAndExtract_FullChainOffline(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	deb := buildControllerDeb(t, map[string]string{"api/fields/Device.json": `{"key":"value"}`})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	// Use a not-yet-created subdirectory so DownloadAndExtract performs the download+extract.
	outputDir := filepath.Join(t.TempDir(), "fields")
	err = DownloadAndExtract(context.Background(), server.Client(), *u, outputDir)
	r.NoError(err)

	data, err := os.ReadFile(filepath.Join(outputDir, "Device.json"))
	r.NoError(err)
	a.JSONEq(`{"key":"value"}`, string(data))

	// A successful extract drops the completion sentinel and removes the
	// intermediate ace.jar from the published directory.
	_, err = os.Stat(filepath.Join(outputDir, extractCompleteSentinel))
	r.NoError(err, "completion sentinel must be present after a successful extract")
	_, err = os.Stat(filepath.Join(outputDir, "ace.jar"))
	r.ErrorIs(err, os.ErrNotExist, "intermediate ace.jar must not be left behind")
}

// TestDownloadAndExtract_NotFoundOffline drives the 404 path through the public
// DownloadAndExtract entrypoint using the injected client seam.
func TestDownloadAndExtract_NotFoundOffline(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	outputDir := filepath.Join(t.TempDir(), "fields")
	err = DownloadAndExtract(context.Background(), server.Client(), *u, outputDir)
	r.Error(err)
	r.ErrorContains(err, "HTTP404")

	// A failed download must not leave a sentinel-bearing (or even
	// existing) output dir behind.
	_, statErr := os.Stat(filepath.Join(outputDir, extractCompleteSentinel))
	r.ErrorIs(statErr, os.ErrNotExist, "no sentinel after a failed download")
}

// TestDownloadAndExtract_ContextCancelled asserts that a pre-cancelled
// context aborts the download before (or during) the request and surfaces a
// context error, leaving no completed output dir.
func TestDownloadAndExtract_ContextCancelled(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	var hits atomic.Int32
	deb := buildControllerDeb(t, map[string]string{"api/fields/Device.json": `{"k":"v"}`})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel up front

	outputDir := filepath.Join(t.TempDir(), "fields")
	err = DownloadAndExtract(ctx, server.Client(), *u, outputDir)
	r.Error(err, "a cancelled context must abort the download")
	r.ErrorIs(err, context.Canceled)

	_, statErr := os.Stat(outputDir)
	r.ErrorIs(statErr, os.ErrNotExist, "cancelled download must not leave an output dir")
}

// TestValidateDownloadURL pins the host/scheme guard: only https on a
// Ubiquiti host (or any loopback host, for the offline test seam) is allowed.
func TestValidateDownloadURL(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		rawURL  string
		wantErr string
	}{
		"https dl.ui.com allowed": {
			rawURL: "https://dl.ui.com/unifi/9.5.21/unifi_sysvinit_all.deb",
		},
		"https fw-download.ubnt.com allowed": {
			rawURL: "https://fw-download.ubnt.com/data/unifi-controller/x-debian-9.5.21.deb",
		},
		"https bare ui.com allowed": {
			rawURL: "https://ui.com/file.deb",
		},
		"loopback ip over http allowed (test seam)": {
			rawURL: "http://127.0.0.1:8080/test.deb",
		},
		"localhost over http allowed (test seam)": {
			rawURL: "http://localhost:9000/test.deb",
		},
		"http on ubiquiti host rejected": {
			rawURL:  "http://dl.ui.com/unifi/x.deb",
			wantErr: "must use https",
		},
		"https on unknown host rejected": {
			rawURL:  "https://evil.example.com/x.deb",
			wantErr: "not an allowed Ubiquiti host",
		},
		"lookalike suffix host rejected": {
			rawURL:  "https://ui.com.evil.example/x.deb",
			wantErr: "not an allowed Ubiquiti host",
		},
		"missing host rejected": {
			rawURL:  "https:///x.deb",
			wantErr: "no host",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			u, err := url.Parse(tc.rawURL)
			r.NoError(err)

			err = validateDownloadURL(*u)
			if tc.wantErr != "" {
				r.ErrorContains(err, tc.wantErr)
				return
			}
			r.NoError(err)
		})
	}
}

// TestDownloadAndExtract_RejectsBadURL drives the host guard through the public
// entrypoint: a non-allowed host is rejected before any HTTP request is made.
func TestDownloadAndExtract_RejectsBadURL(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	u, err := url.Parse("http://dl.ui.com/unifi/x.deb") // http, not https
	r.NoError(err)

	outputDir := filepath.Join(t.TempDir(), "fields")
	err = DownloadAndExtract(context.Background(), http.DefaultClient, *u, outputDir)
	r.Error(err)
	r.ErrorContains(err, "must use https")

	_, statErr := os.Stat(outputDir)
	r.ErrorIs(statErr, os.ErrNotExist, "rejected URL must not create an output dir")
}

// TestDownloadAndExtract_MidExtractFailureReExtracts pins the core
// guarantee: an extract that fails partway (here, an oversize JSON entry trips
// the decompression-bomb cap) leaves NO completed output dir, and a subsequent
// run against a healthy server re-extracts successfully.
func TestDownloadAndExtract_MidExtractFailureReExtracts(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	oversize := strings.Repeat("a", maxJSONSize+1)
	badDeb := buildControllerDeb(t, map[string]string{
		"api/fields/Device.json": `{"k":"v"}`,
		"api/fields/Big.json":    oversize,
	})
	goodDeb := buildControllerDeb(t, map[string]string{
		"api/fields/Device.json": `{"key":"value"}`,
	})

	// First requests get the corrupting deb; later requests get the good one.
	var serveGood atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		if serveGood.Load() {
			_, _ = rw.Write(goodDeb)
		} else {
			_, _ = rw.Write(badDeb)
		}
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	r.NoError(err)

	outputDir := filepath.Join(t.TempDir(), "fields")

	// Run 1: extraction fails on the oversize entry.
	err = DownloadAndExtract(context.Background(), server.Client(), *u, outputDir)
	r.Error(err, "mid-extract failure must surface an error")
	r.ErrorContains(err, "decompression bomb")

	// The failed run must not leave a directory that a re-run treats as complete.
	complete, cerr := extractionComplete(outputDir)
	r.NoError(cerr)
	a.False(complete, "partial extract must not be marked complete")
	// And no sibling temp dir should be left lying around.
	entries, derr := os.ReadDir(filepath.Dir(outputDir))
	r.NoError(derr)
	for _, e := range entries {
		a.NotContains(e.Name(), ".tmp-", "temp extraction dir must be cleaned up on failure")
	}

	// Run 2: healthy server -> the re-run actually re-extracts to success.
	serveGood.Store(true)
	err = DownloadAndExtract(context.Background(), server.Client(), *u, outputDir)
	r.NoError(err, "re-run after a failed extract must re-extract")

	data, err := os.ReadFile(filepath.Join(outputDir, "Device.json"))
	r.NoError(err)
	a.JSONEq(`{"key":"value"}`, string(data))
	complete, cerr = extractionComplete(outputDir)
	r.NoError(cerr)
	a.True(complete, "successful re-run must be marked complete")
}
