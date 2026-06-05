package main

import (
	"archive/tar"
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/ulikunitz/xz"
	"github.com/xor-gate/ar"
)

const (
	maxAceJarSize = 128 << 20 // 128 MiB — ace.jar is ~tens of MB; generous headroom
	maxJSONSize   = 5 << 20   // 5 MiB — individual API field JSONs are tiny
)

func DownloadAndExtract(client *http.Client, downloadUrl url.URL, outputDir string) error {
	// Check if output directory exists, if not create and perform extraction

	if created, err := ensurePath(outputDir); err != nil {
		return fmt.Errorf("unable to create output directory %s: %w", outputDir, err)
	} else if created {
		log.Debugf("downloading UniFi Controller package from: %s", downloadUrl.String())
		jarFile, err := downloadJar(client, downloadUrl, outputDir)
		if err != nil {
			return err
		}

		log.Debugf("extracting JSON files with API structures from: %s to: %s", jarFile, outputDir)
		if err = extractJSON(jarFile, outputDir); err != nil {
			return err
		}

		log.Debugf("JSON files extracted to: %s", outputDir)
		_, err = os.Stat(outputDir)
		if err != nil {
			return err
		}
	}
	if targetInfo, err := os.Stat(outputDir); err != nil {
		return err
	} else if !targetInfo.IsDir() {
		return errors.New("fields info isn't a directory")
	}
	return nil
}

func downloadJar(client *http.Client, downloadUrl url.URL, outputDir string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadUrl.String(), nil)
	if err != nil {
		return "", fmt.Errorf("unable to download UniFi Controller deb: %w", err)
	}

	debResp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to download UniFi Controller deb: %w", err)
	}
	if debResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to download UniFi Controller deb: HTTP%d. Probably it does not exist under %s", debResp.StatusCode, downloadUrl.String())
	}
	defer debResp.Body.Close()

	uncompressedReader, err := openDebDataTar(debResp.Body)
	if err != nil {
		return "", err
	}

	return extractAceJar(uncompressedReader, outputDir)
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
