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

func TestFrozenLegacyFieldsDir(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	v, err := version.NewVersion("9.5.21+atag-extra")
	require.NoError(t, err)

	// Core() strips the build metadata so the directory name pins the bare version.
	a.Equal(filepath.Join("/base", "v9.5.21"), frozenLegacyFieldsDir("/base", v))
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
	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	err = writeVersionFile(v, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "version.generated.go"))
	require.NoError(t, err)
	a.Contains(string(content), `const UnifiVersion = "7.3.83"`)
}

func TestWriteVersionRepoMarkerFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tmpDir := t.TempDir()
	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	err = writeVersionRepoMarkerFile(v, tmpDir)
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

	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	err = writeVersionFile(v, "/nonexistent/directory")
	require.Error(t, err)
	require.ErrorContains(t, err, "no such file or directory")
}

func TestWriteVersionRepoMarkerFile_InvalidDir(t *testing.T) {
	t.Parallel()

	v, err := version.NewVersion("7.3.83")
	require.NoError(t, err)

	err = writeVersionRepoMarkerFile(v, "/nonexistent/directory")
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
	v, err := version.NewVersion("0.0.0")
	require.NoError(t, err)

	err = writeVersionFile(v, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "version.generated.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `const UnifiVersion = "0.0.0"`)
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

	err = writeVersionRepoMarkerFile(v, readOnlyDir)
	require.Error(t, err)
	require.ErrorContains(t, err, "permission denied")
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
