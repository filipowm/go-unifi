package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-version"
)

const (
	LatestVersionMarker = "latest"
	baseDownloadUrl     = "https://dl.ui.com/unifi/%s/unifi_sysvinit_all.deb"

	// officialSpecDownloadUrl is the UniFi OS Server package, served on the same
	// dl.ui.com path as the internal deb but under a different filename; its data
	// tar carries the Official-API OpenAPI spec (integration.json).
	officialSpecDownloadUrl = "https://dl.ui.com/unifi/%s/unifi-uos_sysvinit.deb"

	// firmwareApiTimeout bounds the firmware-latest JSON call: a slow
	// or hung fw-update.ubnt.com must fail cleanly rather than stall codegen.
	firmwareApiTimeout = 30 * time.Second
)

// minOfficialSpecVersion is the first controller that ships integration.json in
// its UniFi OS Server package. Packages before this version are silently skipped.
var minOfficialSpecVersion = version.Must(version.NewVersion("10.1.78"))

// resolveOfficialSpecVersion resolves the version to use for the Official-API
// OpenAPI spec snapshot, decoupled from the internal (resource-gen) version:
//   - explicit non-empty marker: resolve exactly that version
//   - internal >= 10.1.78: reuse the same version (spec is present in that package)
//   - internal < 10.1.78: resolve latest (old packages predate the Official API)
//
// Callers pass `opts.officialSpecVersion` as the explicit marker; an empty string
// activates the auto-detect logic above so `make generate VERSION=9.5.21` fetches
// the Official spec from the most recent release without rewriting internal resources.
func resolveOfficialSpecVersion(p UnifiVersionProvider, internal *UnifiVersion, explicitMarker string) (*UnifiVersion, error) {
	if explicitMarker != "" {
		return p.ByVersionMarker(explicitMarker)
	}
	if internal.Version.GreaterThanOrEqual(minOfficialSpecVersion) {
		return internal, nil
	}
	return p.Latest()
}

type UnifiVersion struct {
	Version     *version.Version
	DownloadUrl *url.URL
}

func NewUnifiVersion(unifiVersion *version.Version, downloadUrl *url.URL) *UnifiVersion {
	return &UnifiVersion{
		Version:     unifiVersion,
		DownloadUrl: downloadUrl,
	}
}

type UnifiVersionProvider interface {
	Latest() (*UnifiVersion, error)
	ByVersionMarker(versionMarker string) (*UnifiVersion, error)
}

type defaultUnifiVersionProvider struct {
	firmwareUpdateApi string
}

func NewUnifiVersionProvider(firmwareUpdateApi string) UnifiVersionProvider {
	return &defaultUnifiVersionProvider{
		firmwareUpdateApi: firmwareUpdateApi,
	}
}

func (p *defaultUnifiVersionProvider) Latest() (*UnifiVersion, error) {
	url, err := url.Parse(p.firmwareUpdateApi)
	if err != nil {
		return nil, err
	}

	query := url.Query()
	query.Add("filter", firmwareUpdateApiFilter("channel", releaseChannel))
	query.Add("filter", firmwareUpdateApiFilter("product", unifiControllerProduct))
	url.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), firmwareApiTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: firmwareApiTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respData firmwareUpdateApiResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return nil, err
	}

	for _, firmware := range respData.Embedded.Firmware {
		if firmware.Platform != debianPlatform || firmware.Version == nil {
			continue
		}
		// Re-validate the channel/product locally
		// instead of trusting the server-side filter on the firmware API.
		if firmware.Channel != releaseChannel || firmware.Product != unifiControllerProduct {
			continue
		}
		return NewUnifiVersion(firmware.Version.Core(), firmware.Links.Data.Href), nil
	}

	return nil, errors.New("no Unifi Controller firmware found")
}

func (p *defaultUnifiVersionProvider) ByVersionMarker(versionMarker string) (*UnifiVersion, error) {
	if versionMarker == LatestVersionMarker {
		return p.Latest()
	} else {
		unifiVersion, err := version.NewVersion(versionMarker)
		if err != nil {
			return nil, err
		}
		unifiVersion = unifiVersion.Core()
		downloadUrl := fmt.Sprintf(baseDownloadUrl, unifiVersion)
		unifiDownloadUrl, err := url.Parse(downloadUrl)
		if err != nil {
			return nil, err
		}
		return NewUnifiVersion(unifiVersion, unifiDownloadUrl), nil
	}
}

// OfficialSpecURL returns the dl.ui.com URL of the UniFi OS Server package for
// this version, whose data tar carries the Official-API OpenAPI spec.
func (v *UnifiVersion) OfficialSpecURL() (*url.URL, error) {
	return url.Parse(fmt.Sprintf(officialSpecDownloadUrl, v.Version.Core()))
}

// officialSpecSnapshotPath returns the committed snapshot path for the Official
// OpenAPI spec of the given version: <baseDir>/openapi/integration-<ver>.json.
// The versioned filename is the pin, mirroring the .unifi-version marker.
func officialSpecSnapshotPath(baseDir string, version *version.Version) string {
	return filepath.Join(baseDir, "openapi", fmt.Sprintf("integration-%s.json", version.Core()))
}

func writeVersionFile(version *version.Version, outDir string) error {
	versionGo := fmt.Appendf(nil, `
// Generated code. DO NOT EDIT.

package unifi

const UnifiVersion = %q
`, version.Core())

	versionGo, err := format.Source(versionGo)
	if err != nil {
		return err
	}

	_, err = writeGeneratedFile(outDir, "version", string(versionGo))
	return err
}

func writeVersionRepoMarkerFile(version *version.Version, outDir string) error {
	versionRepoMarker := []byte(version.Core().String())
	return os.WriteFile(filepath.Join(outDir, ".unifi-version"), versionRepoMarker, 0o644) //nolint:gosec
}
