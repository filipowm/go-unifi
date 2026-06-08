package main

// This file holds download-related root-orchestration tests that cannot live in
// codegen/internal because they exercise package-main functions
// (downloadOfficialSpecSnapshot) or use root-only types (UnifiVersionProvider).

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filipowm/go-unifi/v2/codegen/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
	"github.com/xor-gate/ar"
)

// rootBuildDataTarXz wraps the given files in a tar archive compressed with xz,
// mimicking the data.tar.xz member of a .deb. Duplicated from the internal test
// helper because test helpers cannot be imported across packages.
func rootBuildDataTarXz(t *testing.T, files map[string][]byte) []byte {
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
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())

	var xzBuf bytes.Buffer
	xw, err := xz.NewWriter(&xzBuf)
	require.NoError(t, err)
	_, err = xw.Write(tarBuf.Bytes())
	require.NoError(t, err)
	require.NoError(t, xw.Close())
	return xzBuf.Bytes()
}

// rootBuildDeb builds an ar archive (the .deb container) holding the supplied members.
func rootBuildDeb(t *testing.T, members map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	aw := ar.NewWriter(&buf)
	require.NoError(t, aw.WriteGlobalHeader())
	for name, content := range members {
		hdr := &ar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0o644,
		}
		require.NoError(t, aw.WriteHeader(hdr))
		_, err := aw.Write(content)
		require.NoError(t, err)
	}
	return buf.Bytes()
}

// TestDownloadOfficialSpecSnapshot_OldVersionSkips pins the <10.1.78 regression-safety
// path in downloadOfficialSpecSnapshot: a UOS package without integration.json must
// yield nil (generation continues) and write no snapshot.
func TestDownloadOfficialSpecSnapshot_OldVersionSkips(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// Build a UOS deb without integration.json, mimicking pre-10.1.78 packages.
	dataTarXz := rootBuildDataTarXz(t, map[string][]byte{"./usr/lib/unifi/other.json": []byte("{}")})
	deb := rootBuildDeb(t, map[string][]byte{"data.tar.xz": dataTarXz})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write(deb)
	}))
	defer server.Close()

	specURL, err := url.Parse(server.URL)
	r.NoError(err)

	specPath := filepath.Join(t.TempDir(), "integration-9.5.21.json")
	logger := setupLogging(false, false)

	// downloadOfficialSpecSnapshot must return nil (non-fatal skip), not propagate ErrOfficialSpecNotFound.
	err = downloadOfficialSpecSnapshot(context.Background(), server.Client(), *specURL, specPath, logger)
	r.NoError(err, "pre-Official-API package must skip without error; generation must continue")

	// No snapshot file must be written for old packages.
	_, statErr := os.Stat(specPath)
	r.ErrorIs(statErr, os.ErrNotExist, "no snapshot must be written for packages lacking integration.json")
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
	err = internal.DownloadAndExtractOfficialSpec(context.Background(), http.DefaultClient, *specURL, outPath)
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
