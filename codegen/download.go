package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/ulikunitz/xz"
	"github.com/xor-gate/ar"
)

const (
	maxAceJarSize = 128 << 20 // 128 MiB — ace.jar is ~tens of MB; generous headroom
	maxJSONSize   = 5 << 20   // 5 MiB — individual API field JSONs are tiny

	// maxOpenAPISpecSize caps the extracted integration.json (the Official-API
	// OpenAPI spec, ~240 KiB today): generous headroom while still guarding
	// against a decompression bomb.
	maxOpenAPISpecSize = 32 << 20

	// officialSpecTarPath is integration.json's location inside the UniFi OS
	// Server package's data.tar.xz.
	officialSpecTarPath = "./usr/lib/unifi/webapps/ROOT/api-docs/integration.json"

	// defaultDownloadTimeout caps the whole .deb download+stream when the caller
	// injects a client without its own Timeout (or a nil client). Streaming the
	// multi-MB .deb body is the long pole, so this is generous.
	defaultDownloadTimeout = 5 * time.Minute

	// extractCompleteSentinel marks a fully-extracted output directory:
	// a version dir without this file is treated as partial/crashed and is
	// re-extracted, so a run that dies mid-extraction can never be silently
	// accepted on the next invocation.
	extractCompleteSentinel = ".extract-complete"
)

// allowedDownloadHostSuffixes pins the controller download to Ubiquiti-owned
// hosts. The static base URL lives under dl.ui.com and the firmware
// API redirects downloads to fw-download.ubnt.com, so both registrable domains
// are permitted (host == suffix or *.suffix). Loopback hosts are allowed
// separately to keep the offline httptest seam working.
var allowedDownloadHostSuffixes = []string{"ui.com", "ubnt.com"}

// errOfficialSpecNotFound marks a UniFi OS Server package that carries no
// integration.json (controllers predating the Official API, < 10.1.68) so
// callers can skip the snapshot instead of failing the whole generation.
var errOfficialSpecNotFound = errors.New("integration.json (OpenAPI spec) not found in UniFi OS Server package")

// DownloadAndExtract downloads the controller .deb from downloadUrl and extracts
// the API field-definition JSONs into outputDir. ctx bounds the network
// download. Extraction is atomic: work happens in a sibling
// temp dir that is renamed into place only after a fully-successful extract, and
// a non-existent or sentinel-less outputDir is treated as missing and
// re-extracted, so a crashed prior run can never be silently accepted.
func DownloadAndExtract(ctx context.Context, client *http.Client, downloadUrl url.URL, outputDir string) error {
	// ctx must be non-nil; callers (generate(), tests) pass a bounded or
	// background context. A nil ctx panics in http.NewRequestWithContext.
	if complete, err := extractionComplete(outputDir); err != nil {
		return err
	} else if complete {
		log.Debugf("API structures already extracted in %s, skipping download", outputDir)
		return nil
	}

	// Reject anything that is not an https URL on a Ubiquiti host (or a
	// loopback test server) before issuing any request.
	if err := validateDownloadURL(downloadUrl); err != nil {
		return fmt.Errorf("refusing to download controller package: %w", err)
	}

	return downloadAndExtractAtomic(ctx, client, downloadUrl, outputDir)
}

// downloadAndExtractAtomic performs the download+extract into a sibling temp dir
// and renames it into place only after a fully-successful extract, so
// a partial extract never lands at outputDir and a crashed run is re-extracted.
func downloadAndExtractAtomic(ctx context.Context, client *http.Client, downloadUrl url.URL, outputDir string) error {
	parent := filepath.Dir(outputDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("unable to create parent directory %s: %w", parent, err)
	}
	tmpDir, err := os.MkdirTemp(parent, filepath.Base(outputDir)+".tmp-")
	if err != nil {
		return fmt.Errorf("unable to create temp extraction directory: %w", err)
	}
	// Best-effort cleanup of the temp dir on any failure; on success it is gone
	// (renamed away) and the RemoveAll is a no-op.
	defer os.RemoveAll(tmpDir)

	log.Debugf("downloading UniFi Controller package from: %s", downloadUrl.String())
	jarFile, err := downloadJar(ctx, client, downloadUrl, tmpDir)
	if err != nil {
		return err
	}

	log.Debugf("extracting JSON files with API structures from: %s to: %s", jarFile, tmpDir)
	if err = extractJSON(jarFile, tmpDir); err != nil {
		return err
	}

	// Drop the intermediate ace.jar so only the field JSONs remain, then write the
	// completion sentinel last so the dir renamed into place is atomically complete.
	// Remove via the local tmpDir (not jarFile) to keep the path off the taint path.
	_ = os.Remove(filepath.Join(tmpDir, "ace.jar"))
	if err = os.WriteFile(filepath.Join(tmpDir, extractCompleteSentinel), nil, 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("unable to write extraction sentinel: %w", err)
	}

	return publishExtractedDir(tmpDir, outputDir)
}

// publishExtractedDir removes any stale partial dir left by a previous crashed
// run and atomically moves the freshly-extracted temp dir into place.
func publishExtractedDir(tmpDir, outputDir string) error {
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("unable to remove stale output directory %s: %w", outputDir, err)
	}
	if err := os.Rename(tmpDir, outputDir); err != nil {
		return fmt.Errorf("unable to move extracted files into %s: %w", outputDir, err)
	}
	log.Debugf("JSON files extracted to: %s", outputDir)
	return nil
}

// extractionComplete reports whether outputDir already holds a fully-extracted
// field set, identified by the completion sentinel. A missing dir, a dir without
// the sentinel (partial/crashed run), or a non-directory all report false so the
// caller re-extracts; a path that exists but is not a directory is an error.
func extractionComplete(outputDir string) (bool, error) {
	info, err := os.Stat(outputDir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s isn't a directory", outputDir)
	}
	if _, err = os.Stat(filepath.Join(outputDir, extractCompleteSentinel)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debugf("output directory %s exists but is missing the completion sentinel; re-extracting", outputDir)
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// validateDownloadURL enforces the download provenance guard: the URL
// must use https and target a Ubiquiti-owned host. Loopback hosts (the offline
// httptest seam) are exempted from the scheme/host checks.
func validateDownloadURL(downloadUrl url.URL) error {
	host := downloadUrl.Hostname()
	if host == "" {
		return fmt.Errorf("download URL has no host: %s", downloadUrl.String())
	}
	if isLoopbackHost(host) {
		return nil
	}
	if downloadUrl.Scheme != "https" {
		return fmt.Errorf("download URL must use https, got %q in %s", downloadUrl.Scheme, downloadUrl.String())
	}
	if !hostAllowed(host) {
		return fmt.Errorf("download host %q is not an allowed Ubiquiti host %v", host, allowedDownloadHostSuffixes)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func hostAllowed(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, suffix := range allowedDownloadHostSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func downloadJar(ctx context.Context, client *http.Client, downloadUrl url.URL, outputDir string) (string, error) {
	err := withDebDataTar(ctx, client, downloadUrl, func(dataTar io.Reader) error {
		_, err := extractAceJar(dataTar, outputDir)
		return err
	})
	if err != nil {
		return "", err
	}
	// Reconstruct the path from outputDir rather than threading extractAceJar's
	// return through the closure (keeps it a clean, non-tainted path).
	return filepath.Join(outputDir, "ace.jar"), nil
}

// withDebDataTar downloads the .deb at downloadUrl and invokes fn with a reader
// over the decompressed data.tar.xz, keeping the response body open for the
// duration of the call. A nil/timeout-less client gets a default-timeout client
// or a bounded context so a hung server cannot stall the (CI) generate job.
func withDebDataTar(ctx context.Context, client *http.Client, downloadUrl url.URL, fn func(io.Reader) error) error {
	if client == nil {
		// Never use http.DefaultClient (no timeout) for a multi-MB
		// streamed download; construct one with a sane default timeout.
		client = &http.Client{Timeout: defaultDownloadTimeout}
	} else if client.Timeout == 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultDownloadTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadUrl.String(), nil)
	if err != nil {
		return fmt.Errorf("unable to download UniFi Controller deb: %w", err)
	}

	debResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to download UniFi Controller deb: %w", err)
	}
	defer debResp.Body.Close()
	if debResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to download UniFi Controller deb: HTTP%d. Probably it does not exist under %s", debResp.StatusCode, downloadUrl.String())
	}

	uncompressedReader, err := openDebDataTar(debResp.Body)
	if err != nil {
		return err
	}
	return fn(uncompressedReader)
}

// DownloadAndExtractOfficialSpec downloads the UniFi OS Server package from
// downloadUrl, extracts the Official-API OpenAPI spec (integration.json), and
// writes a byte-for-byte pinned snapshot to outputPath. It reuses the internal
// trust model: https + Ubiquiti host pinning + size cap + atomic publish.
func DownloadAndExtractOfficialSpec(ctx context.Context, client *http.Client, downloadUrl url.URL, outputPath string) error {
	if err := validateDownloadURL(downloadUrl); err != nil {
		return fmt.Errorf("refusing to download UniFi OS Server package: %w", err)
	}

	var spec []byte
	err := withDebDataTar(ctx, client, downloadUrl, func(dataTar io.Reader) error {
		b, err := extractOfficialSpec(dataTar)
		spec = b
		return err
	})
	if err != nil {
		return err
	}
	return writeOfficialSpecSnapshot(spec, outputPath)
}

// extractOfficialSpec walks the data tar for integration.json and returns its
// raw bytes, capped against decompression bombs. Returns errOfficialSpecNotFound
// when the package predates the Official API.
func extractOfficialSpec(r io.Reader) ([]byte, error) {
	tarReader := tar.NewReader(r)
	log.Debugln("extracting integration.json (OpenAPI spec) from downloaded UniFi OS Server package")
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("in next: %w", err)
		}
		if header.Typeflag != tar.TypeReg || header.Name != officialSpecTarPath {
			continue
		}

		var buf bytes.Buffer
		if _, err = copyWithLimit(&buf, tarReader, maxOpenAPISpecSize); err != nil {
			return nil, fmt.Errorf("unable to read integration.json: %w", err)
		}
		log.Debugf("integration.json extracted (%d bytes)", buf.Len())
		return buf.Bytes(), nil
	}
	return nil, errOfficialSpecNotFound
}

// writeOfficialSpecSnapshot writes the spec to outputPath atomically — temp file
// in the same dir + rename — so a partial write never publishes. A single rename
// is atomic, so (unlike the multi-file ace.jar path) no sentinel is needed.
func writeOfficialSpecSnapshot(spec []byte, outputPath string) error {
	if !json.Valid(spec) {
		return errors.New("extracted integration.json is not valid JSON")
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("unable to create snapshot directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(outputPath)+".tmp-")
	if err != nil {
		return fmt.Errorf("unable to create snapshot temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed away on success

	if _, err = tmp.Write(spec); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("unable to write snapshot: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("unable to close snapshot temp file: %w", err)
	}
	// os.CreateTemp yields 0o600; normalize to the committed-marker mode.
	if err = os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("unable to set snapshot file mode: %w", err)
	}
	if err = os.Rename(tmpName, outputPath); err != nil {
		return fmt.Errorf("unable to publish snapshot to %s: %w", outputPath, err)
	}
	log.Debugf("Official OpenAPI spec snapshot written to: %s", outputPath)
	return nil
}

// openDebDataTar iterates the entries of a .deb (ar archive) and returns a reader
// over the decompressed contents of the data.tar.xz member.
func openDebDataTar(body io.Reader) (io.Reader, error) {
	arReader := ar.NewReader(body)
	for {
		header, err := arReader.Next()
		if errors.Is(err, io.EOF) || header == nil {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("in ar next: %w", err)
		}
		if header.Name == "data.tar.xz" {
			uncompressedReader, err := xz.NewReader(arReader)
			if err != nil {
				return nil, fmt.Errorf("in xz reader: %w", err)
			}
			return uncompressedReader, nil
		}
	}
	return nil, errors.New("unable to find .deb data file")
}

// extractAceJar walks the tar stream looking for ace.jar, writes it under outputDir
// and returns the path to the created file.
func extractAceJar(r io.Reader, outputDir string) (string, error) {
	tarReader := tar.NewReader(r)
	log.Debugln("extracting ace.jar from downloaded controller package")
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("in next: %w", err)
		}
		if header.Typeflag != tar.TypeReg || header.Name != "./usr/lib/unifi/lib/ace.jar" {
			continue
		}

		aceJar, err := os.Create(filepath.Join(outputDir, "ace.jar"))
		if err != nil {
			return "", fmt.Errorf("unable to create temp file: %w", err)
		}
		defer aceJar.Close()

		if _, err = copyWithLimit(aceJar, tarReader, maxAceJarSize); err != nil {
			return "", fmt.Errorf("unable to write ace.jar temp file: %w", err)
		}
		log.Debugf("ace.jar extracted to: %s", aceJar.Name())
		return aceJar.Name(), nil
	}
	return "", errors.New("unable to find ace.jar")
}

func extractJSON(jarFile, fieldsDir string) error {
	jarZip, err := zip.OpenReader(jarFile)
	if err != nil {
		return fmt.Errorf("unable to open jar: %w", err)
	}
	defer jarZip.Close()

	log.Tracef("opened jar %s with %d files", jarFile, len(jarZip.File))
	for _, f := range jarZip.File {
		if !strings.HasPrefix(f.Name, "api/fields/") || path.Ext(f.Name) != ".json" {
			continue
		}
		if err = extractZipEntry(f, fieldsDir); err != nil {
			return fmt.Errorf("unable to write JSON file: %w", err)
		}
	}

	return splitSettingsFile(fieldsDir)
}

// extractZipEntry copies a single zip entry into fieldsDir, sanitizing its path.
func extractZipEntry(f *zip.File, fieldsDir string) error {
	log.Tracef("extracting %s", f.Name)
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dstPath, err := sanitizeExtractedPath(f.Name, fieldsDir)
	if err != nil {
		return err
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = copyWithLimit(dst, src, maxJSONSize)
	log.Debugf("extracted %s", f.Name)
	if err != nil {
		return err
	}
	return nil
}

// splitSettingsFile reads the extracted Setting.json (if present) and writes one
// Setting<Camel>.json file per top-level setting key.
func splitSettingsFile(fieldsDir string) error {
	settingsData, err := os.ReadFile(filepath.Join(fieldsDir, "Setting.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to open settings file: %w", err)
	}

	var settings map[string]any
	err = json.Unmarshal(settingsData, &settings)
	if err != nil {
		return fmt.Errorf("unable to unmarshal settings: %w", err)
	}

	log.Debugf("splitting Settings.json into individual setting files")
	for settingKey, settingValue := range settings {
		settingName := strcase.ToCamel(settingKey)
		fileName := fmt.Sprintf("Setting%s.json", settingName)
		log.Tracef("splitting %s", fileName)

		data, err := json.MarshalIndent(settingValue, "", "  ")
		if err != nil {
			return fmt.Errorf("unable to marshal setting %q: %w", settingKey, err)
		}

		err = os.WriteFile(filepath.Join(fieldsDir, fileName), data, 0o644) //nolint:gosec
		if err != nil {
			return fmt.Errorf("unable to write new settings file: %w", err)
		}
		log.Tracef("splitted %s into %s", settingKey, fileName)
	}

	return nil
}

func sanitizeExtractedPath(filePath, destinationDir string) (string, error) {
	absDestinationDir, err := filepath.Abs(destinationDir)
	if err != nil {
		return "", err
	}

	absFilePath, err := filepath.Abs(filepath.Join(destinationDir, filepath.Base(filePath)))
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absFilePath, absDestinationDir) {
		return "", fmt.Errorf("invalid file path: %s", filePath)
	}

	return absFilePath, nil
}
