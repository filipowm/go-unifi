package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] version\n", path.Base(os.Args[0]))
	fmt.Printf("version can be a specific version or '%s' (default) for the latest UniFi Controller version\n", LatestVersionMarker)
	flag.PrintDefaults()
}

func setupLogging(debugEnabled, traceEnabled bool) {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		FullTimestamp:          false,
	})
	if traceEnabled {
		log.SetLevel(log.TraceLevel)
	} else if debugEnabled {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	flag.Usage = usage

	versionBaseDirFlag := flag.String("version-base-dir", ".", "The base directory for version JSON files")
	outputDirFlag := flag.String("output-dir", ".", "The output directory of the generated Go code")
	downloadOnly := flag.Bool("download-only", false, "Only download and build the API structures JSON directory, do not generate")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	traceFlag := flag.Bool("trace", false, "Enable trace logging")

	flag.Parse()
	setupLogging(*debugFlag, *traceFlag)
	specifiedVersion := strings.TrimSpace(flag.Arg(0))
	if specifiedVersion == "" {
		specifiedVersion = LatestVersionMarker // default to latest version
	}
	unifiVersion, err := determineUnifiVersion(specifiedVersion)
	if err != nil {
		log.Fatalf("unable to determine version and download URL for Unifi version %s: %s", specifiedVersion, err)
		panic(err)
	}

	log.Infof("UniFi Controller version: %s", unifiVersion.Version)
	log.Infof("UniFi Controller download URL: %s", unifiVersion.DownloadUrl.String())

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("unable to determine working directory: %s", err)
		panic(err)
	}

	structuresDir := filepath.Join(wd, *versionBaseDirFlag, fmt.Sprintf("v%s", unifiVersion.Version))
	log.Infoln("Downloading UniFi Controller API structures definitions...")
	err = DownloadAndExtract(*unifiVersion.DownloadUrl, structuresDir)
	if err != nil {
		log.Fatalf("unable to download and extract UniFi Controller API structures definitions: %s", err)
		panic(err)
	}
	log.Infof("Downloaded UniFi Controller API structures definitions in %s", structuresDir)

	if *downloadOnly {
		log.Infoln("Structure JSONs ready!")
		os.Exit(0)
	}

	log.Infoln("Generating resources code...")
	outDir := filepath.Join(wd, *outputDirFlag)
	if err = generateCode(structuresDir, outDir); err != nil {
		log.Fatalf("unable to generate resources code: %s", err)
		panic(err)
	}

	log.Infof("Writing version file...")
	if err = writeVersionFile(unifiVersion.Version, outDir); err != nil {
		log.Fatalf("failed to write version file to %s: %s", outDir, err)
		panic(err)
	}

	basepath := filepath.Dir(wd)
	if err = writeVersionRepoMarkerFile(unifiVersion.Version, basepath); err != nil {
		log.Fatalf("failed to write version file to %s: %s", basepath, err)
		panic(err)
	}

	log.Infof("Generated resources in %s", outDir)
}
