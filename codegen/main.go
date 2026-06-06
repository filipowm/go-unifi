package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger is the minimal logging surface the generation pipeline depends on. It
// is satisfied by *logrus.Logger (and *logrus.Entry), so production code can
// inject a real logger while tests inject their own instance with a local hook,
// asserting output in parallel without mutating shared state.
type Logger interface {
	Tracef(format string, args ...any)
	Debugf(format string, args ...any)
	Debugln(args ...any)
	Infof(format string, args ...any)
	Infoln(args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Error(args ...any)
}

// log is the package-global logger used by the CLI path and as the default for
// any pipeline component that was not given an explicit logger. Production
// generate() calls thread an injected logger instead, so the global
// is only the CLI fallback — it is no longer the only sink, which is what made
// the previous output-asserting tests racy and forced them to run serially.
var log = logrus.New()

// defaultLogger returns the package-global logger, used as the fallback when a
// pipeline component is constructed without an explicit logger.
func defaultLogger() Logger { return log }

// orDefaultLogger returns logger, or the package-global fallback when it is nil.
// Centralizing the nil-guard keeps the pipeline entry points from each carrying
// their own branch.
func orDefaultLogger(logger Logger) Logger {
	if logger == nil {
		return defaultLogger()
	}
	return logger
}

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] version\n", path.Base(os.Args[0]))
	fmt.Printf("version can be a specific version or '%s' (default) for the latest UniFi Controller version\n", LatestVersionMarker)
	flag.PrintDefaults()
}

// setupLogging configures and returns a *logrus.Logger at the level implied by
// the debug/trace flags. The returned logger is what the CLI injects into
// generate(); it intentionally does NOT mutate the package global so callers
// (and tests) get an isolated, output-assertable instance.
func setupLogging(debugEnabled, traceEnabled bool) *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		ForceColors:            true,
		FullTimestamp:          false,
	})
	if traceEnabled {
		l.SetLevel(logrus.TraceLevel)
	} else if debugEnabled {
		l.SetLevel(logrus.DebugLevel)
	} else {
		l.SetLevel(logrus.InfoLevel)
	}
	return l
}

type options struct {
	versionBaseDir     string
	outputDir          string
	downloadOnly       bool
	version            string
	firmwareUpdateApi  string
	customizationsPath string
	// logger receives the pipeline's structured output. When nil, generate()
	// falls back to the package-global logger so the CLI path is unaffected.
	// Tests inject their own instance to assert output without touching the
	// shared global.
	logger Logger
	// v2BaseDir is the directory holding the hand-maintained V2-API field
	// definitions (the "codegen/v2" tree). When empty, generate() discovers it
	// via findCodegenDir relative to the repo root, preserving the CLI default.
	// Tests inject a fixture path to exercise generation without the real repo
	// layout.
	v2BaseDir string
}

func main() {
	flag.Usage = usage

	versionBaseDirFlag := flag.String("version-base-dir", ".", "The base directory for version JSON files")
	outputDirFlag := flag.String("output-dir", ".", "The output directory of the generated Go code")
	downloadOnly := flag.Bool("download-only", false, "Only download and build the API structures JSON directory, do not generate")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	traceFlag := flag.Bool("trace", false, "Enable trace logging")

	flag.CommandLine.Init(os.Args[0], flag.PanicOnError) // set error handling to panic if parse ends with error
	flag.Parse()
	logger := setupLogging(*debugFlag, *traceFlag)
	specifiedVersion := strings.TrimSpace(flag.Arg(0))
	if specifiedVersion == "" {
		specifiedVersion = LatestVersionMarker // default to latest version
	}
	err := generate(options{
		versionBaseDir:     *versionBaseDirFlag,
		outputDir:          *outputDirFlag,
		downloadOnly:       *downloadOnly,
		version:            specifiedVersion,
		firmwareUpdateApi:  defaultFirmwareUpdateApi,
		customizationsPath: "customizations.yml",
		logger:             logger,
	})
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

// resolveV2BaseDir returns the injected V2-API field-definitions base dir, or
// discovers codegen/v2 relative to the project root when none was injected.
func resolveV2BaseDir(injected string) (string, error) {
	if injected != "" {
		return injected, nil
	}
	codegenPath, err := findCodegenDir()
	if err != nil {
		return "", fmt.Errorf("failed to find codegen directory: %w", err)
	}
	return filepath.Join(codegenPath, "v2"), nil
}

func generate(opts options) error {
	logger := orDefaultLogger(opts.logger)

	p := NewUnifiVersionProvider(opts.firmwareUpdateApi)
	unifiVersion, err := p.ByVersionMarker(opts.version)
	if err != nil {
		return fmt.Errorf("unable to determine version and download URL for Unifi version %s: %w", opts.version, err)
	}

	logger.Infof("UniFi Controller version: %s", unifiVersion.Version)
	logger.Infof("UniFi Controller download URL: %s", unifiVersion.DownloadUrl.String())

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to determine working directory: %w", err)
	}
	structuresDir := resolveDir(wd, opts.versionBaseDir)
	structuresDir = filepath.Join(structuresDir, fmt.Sprintf("v%s", unifiVersion.Version))
	logger.Infoln("Downloading UniFi Controller API structures definitions...")
	ctx, cancel := context.WithTimeout(context.Background(), defaultDownloadTimeout)
	defer cancel()
	err = DownloadAndExtract(ctx, http.DefaultClient, *unifiVersion.DownloadUrl, structuresDir)
	if err != nil {
		return fmt.Errorf("unable to download and extract UniFi Controller API structures definitions: %w", err)
	}
	logger.Infof("Downloaded UniFi Controller API structures definitions in %s", structuresDir)

	if opts.downloadOnly {
		logger.Infoln("Structure JSONs ready!")
		return nil
	}

	logger.Infoln("Generating resources code...")

	// Resolve the V2-API field-definitions base dir. Tests inject opts.v2BaseDir
	// to avoid depending on the real repo layout; the CLI leaves it empty and we
	// discover codegen/v2 relative to the project root.
	v2BaseDir, err := resolveV2BaseDir(opts.v2BaseDir)
	if err != nil {
		return err
	}

	outDir := resolveDir(wd, opts.outputDir)
	customizer, err := NewCodeCustomizer(opts.customizationsPath)
	if err != nil {
		return fmt.Errorf("unable to create code customizer: %w", err)
	}
	customizer.logger = logger
	if err = generateCode(structuresDir, v2BaseDir, outDir, *customizer, logger); err != nil {
		return fmt.Errorf("unable to generate resources code: %w", err)
	}

	logger.Infof("Writing version file...")
	if err = writeVersionFile(unifiVersion.Version, outDir); err != nil {
		return fmt.Errorf("failed to write version file to %s: %w", outDir, err)
	}

	basepath := filepath.Dir(wd)
	if err = writeVersionRepoMarkerFile(unifiVersion.Version, basepath); err != nil {
		return fmt.Errorf("failed to write version file to %s: %w", basepath, err)
	}

	logger.Infof("Generated resources in %s", outDir)
	return nil
}

// resolveDir returns dir as-is if absolute, otherwise joined with base.
func resolveDir(base, dir string) string {
	if path.IsAbs(dir) {
		return dir
	}
	return filepath.Join(base, dir)
}
