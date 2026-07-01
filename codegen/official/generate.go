package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
)

const (
	// defaultPackageName is the Go package the Official models live in.
	defaultPackageName = "official"
	// generatorVersion pins the header's version string so regeneration is
	// byte-identical no matter how the tool is launched (go run vs go test);
	// the codegen submodule carries no VCS tag of its own.
	generatorVersion = "(devel)"
)

// generatedFile names a generated file alongside its rendered source.
type generatedFile struct {
	name string
	code string
}

// GenerateAll reads the committed OpenAPI snapshot at specPath and writes the
// full Official surface into outDir: the oapi-codegen models plus the tri-shape
// wrappers, the Client interface, and its mock. Fully offline.
func GenerateAll(specPath, outDir, pkgName string) error {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("reading spec %s: %w", specPath, err)
	}
	files, err := generateFiles(raw, pkgName)
	if err != nil {
		return err
	}
	for _, f := range files {
		out := filepath.Join(outDir, f.name)
		if err := os.WriteFile(out, []byte(f.code), 0o644); err != nil { //nolint:gosec
			return fmt.Errorf("writing %s: %w", out, err)
		}
	}
	return nil
}

// generateFiles renders every generated file from raw spec bytes: models
// (oapi-codegen) + the parent Client interface/mock + one file per resource group
// (group interface, accessor, wrapper impls, per-group mock). Disk-free so tests
// can assert determinism in-process.
func generateFiles(raw []byte, pkgName string) ([]generatedFile, error) {
	models, err := GenerateModels(raw, pkgName)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing spec JSON: %w", err)
	}
	ops, err := buildOperations(doc)
	if err != nil {
		return nil, fmt.Errorf("building operations: %w", err)
	}
	groups, err := buildGroups(ops)
	if err != nil {
		return nil, fmt.Errorf("building groups: %w", err)
	}
	client, err := generateClient(groups, pkgName)
	if err != nil {
		return nil, fmt.Errorf("generating client: %w", err)
	}
	files := []generatedFile{
		{"models.generated.go", models},
		{"client.generated.go", client},
	}
	for _, g := range groups {
		code, err := generateGroupFile(g, pkgName)
		if err != nil {
			return nil, fmt.Errorf("generating group %s: %w", g.Name, err)
		}
		files = append(files, generatedFile{g.file(), code})
	}
	return files, nil
}

// GenerateModels runs the transform and oapi-codegen against raw spec bytes,
// returning the generated source. Disk-free so tests can assert determinism and
// round-tripping in-process.
func GenerateModels(raw []byte, pkgName string) (string, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("parsing spec JSON: %w", err)
	}
	exclude, err := Transform(doc)
	if err != nil {
		return "", fmt.Errorf("transforming spec: %w", err)
	}
	transformed, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("re-encoding spec: %w", err)
	}

	// Load from memory rather than a temp file: no fixed-path or cleanup concern,
	// and internal $refs resolve since the spec has no external references.
	spec, err := openapi3.NewLoader().LoadFromData(transformed)
	if err != nil {
		return "", fmt.Errorf("loading transformed spec: %w", err)
	}

	version := generatorVersion
	code, err := codegen.Generate(spec, codegen.Configuration{
		PackageName:          pkgName,
		Generate:             codegen.GenerateOptions{Models: true},
		NoVCSVersionOverride: &version,
		OutputOptions:        codegen.OutputOptions{ExcludeSchemas: exclude},
	})
	if err != nil {
		return "", fmt.Errorf("generating models: %w", err)
	}
	rewritten := rewritePlaceholderDocs(code)
	return normalizeHeader(rewritten)
}

// rewritePlaceholderDocs replaces oapi-codegen's zero-info placeholder godoc
// lines with short, idiomatic equivalents. Only exact machine-generated phrases
// are matched — spec-supplied docs always start with uppercase "Defines" and
// are therefore excluded by the lowercase-phrase anchors below.
func rewritePlaceholderDocs(code string) string {
	// All oapi-codegen placeholders use lowercase "defines"; spec-supplied enum
	// docs use uppercase "Defines values for …" so they can't match these patterns.
	// Go's RE2 engine has no backreference support; case is the safe discriminant.
	modelRe := regexp.MustCompile(`^// (\w+) defines model for .+\.$`)
	paramsRe := regexp.MustCompile(`^// (\w+) defines parameters for .+\.$`)
	// Body lines carry extra content ("for application/json ContentType.").
	bodyRe := regexp.MustCompile(`^// (\w+) defines body for .+\.$`)

	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if m := modelRe.FindStringSubmatch(line); m != nil {
			lines[i] = "// " + m[1] + " is a generated model for the UniFi Official API."
		} else if m := paramsRe.FindStringSubmatch(line); m != nil {
			lines[i] = "// " + m[1] + " holds query parameters for the UniFi Official API."
		} else if m := bodyRe.FindStringSubmatch(line); m != nil {
			lines[i] = "// " + m[1] + " is a generated request body for the UniFi Official API."
		}
	}
	return strings.Join(lines, "\n")
}

// normalizeHeader collapses oapi-codegen's "// Package X ..." doc into a bare
// DO-NOT-EDIT banner (the repo's other generated files use this form), so the
// hand-written package doc in official.go stays the single package godoc.
func normalizeHeader(code string) (string, error) {
	pkg := strings.Index(code, "\npackage ")
	if pkg < 0 {
		return "", errors.New("generated code has no package clause")
	}
	preamble, body := code[:pkg], code[pkg+1:]
	start := strings.Index(preamble, "// Code generated")
	if start < 0 {
		return "", errors.New("generated code has no DO NOT EDIT banner")
	}
	banner := preamble[start:]
	if nl := strings.IndexByte(banner, '\n'); nl >= 0 {
		banner = banner[:nl]
	}
	return banner + "\n\n" + body, nil
}

// specNameRe matches a committed OpenAPI snapshot filename, capturing its version.
var specNameRe = regexp.MustCompile(`^integration-(.+)\.json$`)

// specFile is a committed OpenAPI snapshot with its parsed version.
type specFile struct {
	path    string
	version string
	parts   []int
}

// listSpecs returns every committed integration-<ver>.json snapshot in dir,
// sorted ascending by numeric version so the newest is last. Supporting multiple
// snapshots lets the codegen pin one version while the website renders latest.
func listSpecs(dir string) ([]specFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading openapi dir %s: %w", dir, err)
	}
	var specs []specFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := specNameRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		specs = append(specs, specFile{
			path:    filepath.Join(dir, e.Name()),
			version: m[1],
			parts:   parseVersionParts(m[1]),
		})
	}
	sort.Slice(specs, func(i, j int) bool {
		return compareVersionParts(specs[i].parts, specs[j].parts) < 0
	})
	return specs, nil
}

// parseVersionParts splits a version string into its numeric components,
// tolerating any non-digit separator. Non-numeric tail segments (e.g. "-rc1")
// contribute nothing, mirroring the website's numeric-collation ordering.
func parseVersionParts(v string) []int {
	fields := strings.FieldsFunc(v, func(r rune) bool { return r < '0' || r > '9' })
	parts := make([]int, len(fields))
	for i, f := range fields {
		parts[i], _ = strconv.Atoi(f)
	}
	return parts
}

// compareVersionParts orders two version-component slices: shorter sorts first
// when a common prefix is equal (10.1 < 10.1.5).
func compareVersionParts(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	default:
		return 0
	}
}

// ResolveSnapshot returns the committed integration-<ver>.json snapshot in dir to
// generate from. Multiple snapshots may be committed side by side; version selects
// which one:
//   - non-empty: the exact integration-<version>.json (fails loudly if absent),
//   - empty: the newest committed snapshot by numeric version — the same choice
//     the website makes, so a standalone default reproduces latest.
func ResolveSnapshot(dir, version string) (string, error) {
	specs, err := listSpecs(dir)
	if err != nil {
		return "", err
	}
	if len(specs) == 0 {
		return "", fmt.Errorf("no integration-*.json OpenAPI spec found in %s", dir)
	}
	if version == "" {
		return specs[len(specs)-1].path, nil
	}
	for _, s := range specs {
		if s.version == version {
			return s.path, nil
		}
	}
	available := make([]string, len(specs))
	for i, s := range specs {
		available[i] = s.version
	}
	return "", fmt.Errorf("no committed OpenAPI spec for version %s in %s (available: %s)", version, dir, strings.Join(available, ", "))
}

// pinnedOfficialVersion walks up from dir to find the repo's
// .unifi-version-official marker and returns its trimmed contents, or "" (no
// error) when it cannot be located — callers then fall back to the newest snapshot.
func pinnedOfficialVersion(dir string) string {
	d, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		if b, err := os.ReadFile(filepath.Join(d, ".unifi-version-official")); err == nil {
			return strings.TrimSpace(string(b))
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// resolveSnapshotVersion picks the snapshot version to generate from: the explicit
// flag when set, else the committed .unifi-version-official pin, else "" so
// ResolveSnapshot selects the newest snapshot.
func resolveSnapshotVersion(openapiDir, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return pinnedOfficialVersion(openapiDir)
}
