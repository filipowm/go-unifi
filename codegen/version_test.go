package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertLatestVersionUsingProvider(t *testing.T, provider func(p UnifiVersionProvider) (*UnifiVersion, error)) {
	t.Helper()
	assert := assert.New(t)
	require := require.New(t)

	fwVersion, err := version.NewVersion("7.3.83+atag-7.3.83-19645")
	require.NoError(err)

	fwDownload, err := url.Parse("https://fw-download.ubnt.com/data/unifi-controller/c31c-debian-7.3.83-c9249c913b91416693b869b9548850c3.deb")
	require.NoError(err)

	respData := firmwareUpdateApiResponse{
		Embedded: firmwareUpdateApiResponseEmbedded{
			Firmware: []firmwareUpdateApiResponseEmbeddedFirmware{
				{
					Channel:  releaseChannel,
					Created:  "2023-02-06T08:55:31+00:00",
					Id:       "c9249c91-3b91-4166-93b8-69b9548850c3",
					Platform: debianPlatform,
					Product:  unifiControllerProduct,
					Version:  fwVersion,
					Links: firmwareUpdateApiResponseEmbeddedFirmwareLinks{
						Data: firmwareUpdateApiResponseEmbeddedFirmwareDataLink{
							Href: fwDownload,
						},
					},
				},
				{
					Channel:  releaseChannel,
					Created:  "2023-02-06T08:51:36+00:00",
					Id:       "2a600108-7f79-4b3e-b6e0-4dd262460457",
					Platform: "document",
					Product:  unifiControllerProduct,
					Version:  fwVersion,
					Links: firmwareUpdateApiResponseEmbeddedFirmwareLinks{
						Data: firmwareUpdateApiResponseEmbeddedFirmwareDataLink{
							Href: nil,
						},
					},
				},
				{
					Channel:  releaseChannel,
					Created:  "2023-02-06T08:51:37+00:00",
					Id:       "9d2d413d-36ce-4742-a10d-4351aac6f08d",
					Platform: "windows",
					Product:  unifiControllerProduct,
					Version:  fwVersion,
					Links: firmwareUpdateApiResponseEmbeddedFirmwareLinks{
						Data: firmwareUpdateApiResponseEmbeddedFirmwareDataLink{
							Href: nil,
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		assert.Contains(query["filter"], firmwareUpdateApiFilter("channel", releaseChannel))
		assert.Contains(query["filter"], firmwareUpdateApiFilter("product", unifiControllerProduct))

		resp, err := json.Marshal(respData)
		assert.NoError(err)

		_, err = rw.Write(resp)
		assert.NoError(err)
	}))
	defer server.Close()

	p := NewUnifiVersionProvider(server.URL)

	gotVersion, err := provider(p)
	require.NoError(err)

	assert.Equal(fwVersion.Core(), gotVersion.Version)
	assert.Equal(fwDownload, gotVersion.DownloadUrl)
}

func TestLatestUnifiVersion(t *testing.T) {
	t.Parallel()
	assertLatestVersionUsingProvider(t, func(p UnifiVersionProvider) (*UnifiVersion, error) {
		return p.Latest()
	})
}

func TestDetermineUnifiVersion_latest(t *testing.T) {
	t.Parallel()
	assertLatestVersionUsingProvider(t, func(p UnifiVersionProvider) (*UnifiVersion, error) {
		return p.ByVersionMarker(LatestVersionMarker)
	})
}

func TestDetermineUnifiVersion_provided(t *testing.T) {
	t.Parallel()
	testCases := map[string]string{
		"7.3.83+atag-7.3.83-19645": "7.3.83",
		"7.3.83":                   "7.3.83",
		"7.3":                      "7.3.0",
		"7":                        "7.0.0",
	}

	for providedVersion, expectedVersion := range testCases {
		t.Run(providedVersion, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			unifiVersion, err := NewUnifiVersionProvider(defaultFirmwareUpdateApi).ByVersionMarker(providedVersion)
			require.NoError(t, err)

			a.Equal(expectedVersion, unifiVersion.Version.String())
			a.Equal(fmt.Sprintf(baseDownloadUrl, expectedVersion), unifiVersion.DownloadUrl.String())
		})
	}
}

func TestDetermineUnifiVersion_invalid(t *testing.T) {
	t.Parallel()
	testCases := []string{
		"a7.3.83",
		"7.3.83 ",
		"invalid",
		"-1",
		"",
	}

	for _, providedVersion := range testCases {
		t.Run(providedVersion, func(t *testing.T) {
			t.Parallel()
			_, err := NewUnifiVersionProvider(defaultFirmwareUpdateApi).ByVersionMarker(providedVersion)
			require.ErrorContains(t, err, providedVersion)
		})
	}
}

func TestNewUnifiVersion(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)
	downloadUrl, err := url.Parse("https://example.com/download")
	require.NoError(t, err)

	unifiVersion := NewUnifiVersion(v, downloadUrl)
	a.Equal(v, unifiVersion.Version)
	a.Equal(downloadUrl, unifiVersion.DownloadUrl)
}

func TestOfficialSpecURL(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	v, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	got, err := NewUnifiVersion(v, nil).OfficialSpecURL()
	require.NoError(t, err)
	a.Equal("https://dl.ui.com/unifi/10.1.78/unifi-uos_sysvinit.deb", got.String())
}

func TestOfficialSpecSnapshotPath(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	v, err := version.NewVersion("10.1.78+atag-extra")
	require.NoError(t, err)

	// Core() strips the build metadata so the filename pins the bare version.
	a.Equal(filepath.Join("/base", "openapi", "integration-10.1.78.json"), officialSpecSnapshotPath("/base", v))
}

func TestLegacyFieldsDir(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	v, err := version.NewVersion("9.5.21+atag-extra")
	require.NoError(t, err)

	// Core() strips the build metadata so the directory name pins the bare version.
	a.Equal(filepath.Join("/base", "v9.5.21"), legacyFieldsDir("/base", v))
}

func TestLatestUnifiVersion_HttpError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := NewUnifiVersionProvider(server.URL).Latest()
	require.Error(t, err)
}

func TestLatestUnifiVersion_InvalidJson(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, err := rw.Write([]byte("invalid json"))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	_, err := NewUnifiVersionProvider(server.URL).Latest()

	require.Error(t, err)
	require.ErrorContains(t, err, "invalid")
}

func TestLatestUnifiVersion_NoDebianFirmware(t *testing.T) {
	t.Parallel()

	fwVersion, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	respData := firmwareUpdateApiResponse{
		Embedded: firmwareUpdateApiResponseEmbedded{
			Firmware: []firmwareUpdateApiResponseEmbeddedFirmware{
				{
					Channel:  releaseChannel,
					Platform: "windows",
					Product:  unifiControllerProduct,
					Version:  fwVersion,
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		resp, err := json.Marshal(respData)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
		_, err = rw.Write(resp)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	_, err = NewUnifiVersionProvider(server.URL).Latest()

	require.Error(t, err)
	require.ErrorContains(t, err, "no Unifi Controller firmware found")
}

func TestWriteVersionFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tmpDir := t.TempDir()
	internal, err := version.NewVersion("7.3.83")
	require.NoError(t, err)
	official, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	err = writeVersionFile(internal, official, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "version.generated.go"))
	require.NoError(t, err)
	a.Contains(string(content), `const UnifiVersion = "7.3.83"`)
	a.Contains(string(content), `const OfficialAPIVersion = "10.1.78"`)
}

func TestWriteVersionFile_BothConstsDistinct(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	internal, err := version.NewVersion("9.5.21")
	require.NoError(t, err)
	official, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	require.NoError(t, writeVersionFile(internal, official, tmpDir))

	content, err := os.ReadFile(filepath.Join(tmpDir, "version.generated.go"))
	require.NoError(t, err)
	// Internal and Official versions legitimately diverge — verify each pin is written.
	assert.Contains(t, string(content), `const UnifiVersion = "9.5.21"`)
	assert.Contains(t, string(content), `const OfficialAPIVersion = "10.1.78"`)
}

func TestWriteVersionRepoMarkerFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tmpDir := t.TempDir()
	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	err = writeVersionMarker(v, tmpDir, ".unifi-version")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, ".unifi-version"))
	require.NoError(t, err)
	a.Equal("7.3.83", string(content))
}

func TestLatestUnifiVersion_InvalidUrl(t *testing.T) {
	t.Parallel()

	_, err := NewUnifiVersionProvider(":\\invalid").Latest()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid")
}

func TestWriteVersionFile_InvalidDir(t *testing.T) {
	t.Parallel()

	internal, err := version.NewVersion("7.3.83")
	require.NoError(t, err)
	official, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	err = writeVersionFile(internal, official, "/nonexistent/directory")
	require.Error(t, err)
	require.ErrorContains(t, err, "no such file or directory")
}

func TestWriteVersionRepoMarkerFile_InvalidDir(t *testing.T) {
	t.Parallel()

	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	err = writeVersionMarker(v, "/nonexistent/directory", ".unifi-version")
	require.Error(t, err)
	require.ErrorContains(t, err, "no such file or directory")
}

func TestLatestUnifiVersion_NilVersion(t *testing.T) {
	t.Parallel()

	respData := firmwareUpdateApiResponse{
		Embedded: firmwareUpdateApiResponseEmbedded{
			Firmware: []firmwareUpdateApiResponseEmbeddedFirmware{
				{
					Channel:  releaseChannel,
					Platform: debianPlatform,
					Product:  unifiControllerProduct,
					Version:  nil,
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		resp, err := json.Marshal(respData)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
		_, err = rw.Write(resp)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	_, err := NewUnifiVersionProvider(server.URL).Latest()
	require.Error(t, err)
}

func TestWriteVersionFile_EmptyVersion(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	internal, err := version.NewVersion("0.0.0")
	require.NoError(t, err)
	official, err := version.NewVersion("0.0.0")
	require.NoError(t, err)

	err = writeVersionFile(internal, official, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "version.generated.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `const UnifiVersion = "0.0.0"`)
	assert.Contains(t, string(content), `const OfficialAPIVersion = "0.0.0"`)
}

func TestWriteVersionRepoMarkerFile_Permissions(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0o555)
	require.NoError(t, err)

	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	err = writeVersionMarker(v, readOnlyDir, ".unifi-version")
	require.Error(t, err)
	require.ErrorContains(t, err, "permission denied")
}

func TestWriteOfficialVersionRepoMarkerFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tmpDir := t.TempDir()
	v, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	err = writeVersionMarker(v, tmpDir, ".unifi-version-official")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, ".unifi-version-official"))
	require.NoError(t, err)
	a.Equal("10.1.78", string(content))
}

func TestWriteOfficialVersionRepoMarkerFile_InvalidDir(t *testing.T) {
	t.Parallel()

	v, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	err = writeVersionMarker(v, "/nonexistent/directory", ".unifi-version-official")
	require.Error(t, err)
	require.ErrorContains(t, err, "no such file or directory")
}

func TestWriteOfficialVersionRepoMarkerFile_Permissions(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0o555)
	require.NoError(t, err)

	v, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	err = writeVersionMarker(v, readOnlyDir, ".unifi-version-official")
	require.Error(t, err)
	require.ErrorContains(t, err, "permission denied")
}

// TestWriteVersionRepoMarkersIndependent verifies Internal and Official markers
// are written to separate files and their content is independent — the two API
// surfaces can legitimately pin different versions.
func TestWriteVersionRepoMarkersIndependent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	internal, err := version.NewVersion("9.5.21")
	require.NoError(t, err)
	official, err := version.NewVersion("10.1.78")
	require.NoError(t, err)

	require.NoError(t, writeVersionMarker(internal, tmpDir, ".unifi-version"))
	require.NoError(t, writeVersionMarker(official, tmpDir, ".unifi-version-official"))

	internalContent, err := os.ReadFile(filepath.Join(tmpDir, ".unifi-version"))
	require.NoError(t, err)
	officialContent, err := os.ReadFile(filepath.Join(tmpDir, ".unifi-version-official"))
	require.NoError(t, err)

	assert.Equal(t, "9.5.21", string(internalContent))
	assert.Equal(t, "10.1.78", string(officialContent))
	assert.NotEqual(t, string(internalContent), string(officialContent))
}

// recordingVersionProvider records which provider methods resolveOfficialSpecVersion
// invokes, so each branch can be asserted without hitting the network.
type recordingVersionProvider struct {
	latestCalled   bool
	byMarkerCalled bool
	byMarkerArg    string
	latestResult   *UnifiVersion
	byMarkerResult *UnifiVersion
}

func (p *recordingVersionProvider) Latest() (*UnifiVersion, error) {
	p.latestCalled = true
	return p.latestResult, nil
}

func (p *recordingVersionProvider) ByVersionMarker(marker string) (*UnifiVersion, error) {
	p.byMarkerCalled = true
	p.byMarkerArg = marker
	return p.byMarkerResult, nil
}

func mustUnifiVersion(t *testing.T, v string) *UnifiVersion {
	t.Helper()
	ver, err := version.NewVersion(v)
	require.NoError(t, err)
	return NewUnifiVersion(ver, nil)
}

// An explicit marker is resolved verbatim via ByVersionMarker; Latest() is untouched.
func TestResolveOfficialSpecVersion_ExplicitMarker(t *testing.T) {
	t.Parallel()

	want := mustUnifiVersion(t, "10.1.78")
	p := &recordingVersionProvider{byMarkerResult: want}

	got, err := resolveOfficialSpecVersion(p, mustUnifiVersion(t, "9.5.21"), "10.1.78")
	require.NoError(t, err)
	assert.Same(t, want, got)
	assert.True(t, p.byMarkerCalled)
	assert.Equal(t, "10.1.78", p.byMarkerArg)
	assert.False(t, p.latestCalled)
}

// internal >= floor reuses the internal version as-is; neither provider call fires.
func TestResolveOfficialSpecVersion_InternalAtFloor(t *testing.T) {
	t.Parallel()

	p := &recordingVersionProvider{}
	internal := mustUnifiVersion(t, minOfficialSpecVersion.String())

	got, err := resolveOfficialSpecVersion(p, internal, "")
	require.NoError(t, err)
	assert.Same(t, internal, got)
	assert.False(t, p.byMarkerCalled)
	assert.False(t, p.latestCalled)
}

// internal < floor falls back to Latest() (old packages predate the Official API).
func TestResolveOfficialSpecVersion_InternalBelowFloorResolvesLatest(t *testing.T) {
	t.Parallel()

	want := mustUnifiVersion(t, "10.1.78")
	p := &recordingVersionProvider{latestResult: want}

	got, err := resolveOfficialSpecVersion(p, mustUnifiVersion(t, "9.5.21"), "")
	require.NoError(t, err)
	assert.Same(t, want, got)
	assert.True(t, p.latestCalled)
	assert.False(t, p.byMarkerCalled)
}

// newUOSFirmwareServer returns an httptest server that reports the given version
// (simulating a UOS-era firmware API response) for the debian/release product filter.
func newUOSFirmwareServer(t *testing.T, reportedVersion string) *httptest.Server {
	t.Helper()
	fw, err := version.NewVersion(reportedVersion)
	require.NoError(t, err)
	dl, err := url.Parse(fmt.Sprintf("https://dl.ui.com/unifi/%s/unifi_sysvinit_all.deb", fw.Core()))
	require.NoError(t, err)

	respData := firmwareUpdateApiResponse{
		Embedded: firmwareUpdateApiResponseEmbedded{
			Firmware: []firmwareUpdateApiResponseEmbeddedFirmware{
				{
					Channel:  releaseChannel,
					Platform: debianPlatform,
					Product:  unifiControllerProduct,
					Version:  fw,
					Links: firmwareUpdateApiResponseEmbeddedFirmwareLinks{
						Data: firmwareUpdateApiResponseEmbeddedFirmwareDataLink{Href: dl},
					},
				},
			},
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Prove the firmware API is queried with the correct channel/product filters.
		query := req.URL.Query()
		assert.Contains(t, query["filter"], firmwareUpdateApiFilter("channel", releaseChannel))
		assert.Contains(t, query["filter"], firmwareUpdateApiFilter("product", unifiControllerProduct))

		resp, err := json.Marshal(respData)
		assert.NoError(t, err)
		_, err = rw.Write(resp)
		assert.NoError(t, err)
	}))
}

// TestResolveInternalVersion_LatestClampedWhenAPIReportsUOS verifies that when the
// firmware API reports a UOS-era version (> 9.5.21), resolveInternalVersion clamps
// to maxInternalVersion (9.5.21) — the classic controller is EOL past that point.
// Critically, the clamp re-invokes ByVersionMarker("9.5.21") so the returned
// DownloadUrl points to the 9.5.21 .deb, not the UOS package.
func TestResolveInternalVersion_LatestClampedWhenAPIReportsUOS(t *testing.T) {
	t.Parallel()

	server := newUOSFirmwareServer(t, "10.1.78")
	defer server.Close()

	p := NewUnifiVersionProvider(server.URL)
	got, err := resolveInternalVersion(p, LatestVersionMarker)
	require.NoError(t, err)
	assert.Equal(t, maxInternalVersion.String(), got.Version.String())
	// Verify the download URL was resolved for 9.5.21, not the UOS (10.x) package.
	assert.Equal(t, fmt.Sprintf(baseDownloadUrl, maxInternalVersion.String()), got.DownloadUrl.String())
}

// TestResolveInternalVersion_LatestBelowCapPassthrough verifies that when the
// firmware API reports a classic-era version (<= 9.5.21), resolveInternalVersion
// returns it as-is — no spurious clamping.
func TestResolveInternalVersion_LatestBelowCapPassthrough(t *testing.T) {
	t.Parallel()

	server := newUOSFirmwareServer(t, "9.3.45")
	defer server.Close()

	p := NewUnifiVersionProvider(server.URL)
	got, err := resolveInternalVersion(p, LatestVersionMarker)
	require.NoError(t, err)
	assert.Equal(t, "9.3.45", got.Version.String())
}

// TestResolveInternalVersion_LatestAtCapPassthrough verifies that when the firmware
// API reports exactly 9.5.21 (the cap boundary), no clamp fires and the version is
// returned as-is — min(9.5.21, 9.5.21) must equal 9.5.21 unchanged.
func TestResolveInternalVersion_LatestAtCapPassthrough(t *testing.T) {
	t.Parallel()

	server := newUOSFirmwareServer(t, "9.5.21")
	defer server.Close()

	p := NewUnifiVersionProvider(server.URL)
	got, err := resolveInternalVersion(p, LatestVersionMarker)
	require.NoError(t, err)
	assert.Equal(t, "9.5.21", got.Version.String())
	// URL comes from the firmware API response, not a re-resolved clamp.
	assert.Equal(t, fmt.Sprintf(baseDownloadUrl, "9.5.21"), got.DownloadUrl.String())
}

// TestResolveVersions_InternalCapOfficialUncapped guards the resolveVersions wiring:
// internal goes through resolveInternalVersion (cap-enforcing) while official goes
// through resolveOfficialSpecVersion (uncapped). A silent re-wire would break this.
func TestResolveVersions_InternalCapOfficialUncapped(t *testing.T) {
	t.Parallel()

	// Internal: explicit 9.3.45 -> ByVersionMarker is invoked by resolveInternalVersion.
	// Official: auto-detect (9.3.45 < minOfficialSpecVersion) -> Latest() is invoked by resolveOfficialSpecVersion.
	internal935 := mustUnifiVersion(t, "9.3.45")
	official1078 := mustUnifiVersion(t, "10.1.78")
	rec := &recordingVersionProvider{byMarkerResult: internal935, latestResult: official1078}

	internalVer, officialVer, err := resolveVersions(rec, options{
		version:             "9.3.45",
		officialSpecVersion: "", // auto: internal < minOfficialSpecVersion -> Latest()
	})
	require.NoError(t, err)

	// Internal: routed through resolveInternalVersion, below cap, ByVersionMarker invoked.
	assert.True(t, rec.byMarkerCalled, "resolveInternalVersion must call ByVersionMarker")
	assert.Equal(t, "9.3.45", rec.byMarkerArg)
	assert.Equal(t, "9.3.45", internalVer.Version.String())

	// Official: routed through resolveOfficialSpecVersion, Latest() invoked, no cap applied.
	assert.True(t, rec.latestCalled, "resolveOfficialSpecVersion must call Latest() for auto-detect path")
	assert.Equal(t, "10.1.78", officialVer.Version.String())
}

// TestResolveInternalVersion_ExplicitAtCap verifies that an explicit version exactly
// at maxInternalVersion (9.5.21) resolves normally without error.
func TestResolveInternalVersion_ExplicitAtCap(t *testing.T) {
	t.Parallel()

	// ByVersionMarker is called but performs no network I/O for explicit <= cap.
	rec := &recordingVersionProvider{byMarkerResult: mustUnifiVersion(t, "9.5.21")}
	got, err := resolveInternalVersion(rec, "9.5.21")
	require.NoError(t, err)
	assert.Equal(t, "9.5.21", got.Version.String())
	assert.True(t, rec.byMarkerCalled)
	assert.Equal(t, "9.5.21", rec.byMarkerArg)
}

// TestResolveInternalVersion_ExplicitBelowCap verifies that an explicit version
// below maxInternalVersion resolves normally — backward compat for make generate VERSION=<x>.
func TestResolveInternalVersion_ExplicitBelowCap(t *testing.T) {
	t.Parallel()

	// ByVersionMarker is called but performs no network I/O for explicit <= cap.
	rec := &recordingVersionProvider{byMarkerResult: mustUnifiVersion(t, "9.3.45")}
	got, err := resolveInternalVersion(rec, "9.3.45")
	require.NoError(t, err)
	assert.Equal(t, "9.3.45", got.Version.String())
	assert.True(t, rec.byMarkerCalled)
	assert.Equal(t, "9.3.45", rec.byMarkerArg)
}

// TestResolveInternalVersion_ExplicitNewerFails verifies that an explicit version
// above maxInternalVersion (9.5.21) returns an actionable error mentioning the
// classic-controller EOL, the Official OpenAPI frontend, and the CLI flag.
func TestResolveInternalVersion_ExplicitNewerFails(t *testing.T) {
	t.Parallel()

	// Fail-loud returns before reaching the provider; verify it is never consulted.
	rec := &recordingVersionProvider{}
	_, err := resolveInternalVersion(rec, "10.1.78")
	require.Error(t, err)
	require.ErrorContains(t, err, "10.1.78")
	require.ErrorContains(t, err, maxInternalVersion.String())
	require.ErrorContains(t, err, "end-of-life")
	require.ErrorContains(t, err, "Official")
	require.ErrorContains(t, err, "-official-spec-version")
	require.ErrorContains(t, err, "#121")
	assert.False(t, rec.byMarkerCalled)
}

// TestResolveInternalVersion_ExplicitMuchNewerFails verifies the fail-loud path for
// a clearly post-EOL explicit version (e.g. 11.x).
func TestResolveInternalVersion_ExplicitMuchNewerFails(t *testing.T) {
	t.Parallel()

	// Fail-loud returns before reaching the provider; verify it is never consulted.
	rec := &recordingVersionProvider{}
	_, err := resolveInternalVersion(rec, "11.0.0")
	require.Error(t, err)
	require.ErrorContains(t, err, "11.0.0")
	require.ErrorContains(t, err, maxInternalVersion.String())
	require.ErrorContains(t, err, "-official-spec-version")
	require.ErrorContains(t, err, "#121")
	assert.False(t, rec.byMarkerCalled)
}

// TestResolveInternalVersion_PrereleaseAtCapAllowed verifies that a prerelease suffix
// on an otherwise-allowed version (e.g. 9.5.21-rc1) passes the cap check — Core()
// strips the pre-release tag so the effective version equals maxInternalVersion.
func TestResolveInternalVersion_PrereleaseAtCapAllowed(t *testing.T) {
	t.Parallel()

	p := NewUnifiVersionProvider(defaultFirmwareUpdateApi)
	got, err := resolveInternalVersion(p, "9.5.21-rc1")
	require.NoError(t, err)
	assert.Equal(t, "9.5.21", got.Version.String())
}

// TestResolveInternalVersion_BuildMetaAtCapAllowed verifies that build metadata
// (e.g. 9.5.21+ci) on an otherwise-allowed version passes the cap check — Core()
// strips build metadata so the effective version equals maxInternalVersion.
func TestResolveInternalVersion_BuildMetaAtCapAllowed(t *testing.T) {
	t.Parallel()

	p := NewUnifiVersionProvider(defaultFirmwareUpdateApi)
	got, err := resolveInternalVersion(p, "9.5.21+ci")
	require.NoError(t, err)
	assert.Equal(t, "9.5.21", got.Version.String())
}

// TestResolveInternalVersion_PostEOLPrereleaseStillFails verifies that a post-EOL
// version with a prerelease suffix (e.g. 10.0.0-beta) still fails loud — Core()
// strips the pre-release tag but 10.0.0 > maxInternalVersion, so the cap fires.
func TestResolveInternalVersion_PostEOLPrereleaseStillFails(t *testing.T) {
	t.Parallel()

	p := NewUnifiVersionProvider(defaultFirmwareUpdateApi)
	_, err := resolveInternalVersion(p, "10.0.0-beta")
	require.Error(t, err)
	require.ErrorContains(t, err, "10.0.0")
	require.ErrorContains(t, err, maxInternalVersion.String())
	require.ErrorContains(t, err, "end-of-life")
}

// TestResolveInternalVersion_OfficialPathUnaffected verifies that resolveOfficialSpecVersion
// and the underlying provider methods (ByVersionMarker, Latest) are unaffected by the
// internal cap — the Official pipeline legitimately resolves >= 10.1.78.
func TestResolveInternalVersion_OfficialPathUnaffected(t *testing.T) {
	t.Parallel()

	// Verify ByVersionMarker("10.1.78") returns the UOS version as-is.
	p := NewUnifiVersionProvider(defaultFirmwareUpdateApi)
	got, err := p.ByVersionMarker("10.1.78")
	require.NoError(t, err)
	assert.Equal(t, "10.1.78", got.Version.String())

	// Verify resolveOfficialSpecVersion with explicit "10.1.78" marker resolves
	// through ByVersionMarker — the recording provider confirms no cap is applied.
	want := mustUnifiVersion(t, "10.1.78")
	rec := &recordingVersionProvider{byMarkerResult: want}
	official, err := resolveOfficialSpecVersion(rec, mustUnifiVersion(t, "9.5.21"), "10.1.78")
	require.NoError(t, err)
	assert.Same(t, want, official)
	assert.True(t, rec.byMarkerCalled)
	assert.Equal(t, "10.1.78", rec.byMarkerArg)

	// Verify the Latest() fallback path used when internal < floor still returns
	// a UOS version (recording provider — no cap applies here).
	want2 := mustUnifiVersion(t, "10.2.0")
	rec2 := &recordingVersionProvider{latestResult: want2}
	official2, err := resolveOfficialSpecVersion(rec2, mustUnifiVersion(t, "9.5.21"), "")
	require.NoError(t, err)
	assert.Same(t, want2, official2)
	assert.True(t, rec2.latestCalled)
}
