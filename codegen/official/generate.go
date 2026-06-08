package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// ResolveSnapshot returns the single committed integration-<ver>.json snapshot in
// dir, failing loudly on zero or multiple so the pinned version is unambiguous.
func ResolveSnapshot(dir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "integration-*.json"))
	if err != nil {
		return "", fmt.Errorf("glob integration snapshots in %s: %w", dir, err)
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("expected exactly one integration-*.json in %s, found %d", dir, len(matches))
	}
	return matches[0], nil
}
