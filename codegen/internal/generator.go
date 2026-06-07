package internal

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/filipowm/go-unifi/codegen/shared"
	"github.com/iancoleman/strcase"
)

// Generatable is the interface for generation sources.
type Generatable interface {
	Name() string
	GenerateCode() (string, error)
}

// commonTemplate holds the shared template partials (the package header/imports
// and the field / field-customUnmarshalType / typecast / header defines) that the
// api.go.tmpl and apiv2.go.tmpl templates invoke. Factoring it here keeps the two
// resource templates from duplicating an identical top block.
//
//go:embed common.tmpl
var commonTemplate string

// generateCodeFromTemplate renders a template with provided content and formats the source.
func generateCodeFromTemplate(templateName, templateContent string, toWrite any) (string, error) {
	var err error
	var buf bytes.Buffer
	writer := io.Writer(&buf)

	// Parse the shared partials first so the named templates they define (header,
	// field, field-customUnmarshalType, typecast) are available to every template
	// that invokes them. Templates that do not reference the partials are unaffected.
	tpl := template.Must(template.New(templateName).Parse(commonTemplate))
	tpl = template.Must(tpl.Parse(templateContent))

	err = tpl.Execute(writer, toWrite)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format source: %w", err)
	}

	return string(src), err
}

// generateCode generates code for each generation source and writes it to file.
//
// v2BaseDir is the directory holding the V2-API field definitions (the
// "codegen/v2" tree). It is injected by the caller rather than discovered via
// findCodegenDir at runtime, so generation is unit-testable against a fixture
// without the real repo layout. logger receives all pipeline output;
// when nil it falls back to the package-global logger.
func generateCode(fieldsDir, v2BaseDir, outDir string, customizer CodeCustomizer, logger shared.Logger) error {
	logger = orDefaultLogger(logger)
	customizer.logger = logger

	if _, err := shared.EnsurePath(outDir); err != nil {
		return fmt.Errorf("unable to create output directory %s: %w", outDir, err)
	}

	resources, err := buildResourcesFromDownloadedFields(fieldsDir, customizer, false, logger)
	if err != nil {
		return fmt.Errorf("failed to build resources from downloaded fields: %w", err)
	}

	resourcesCustomV2, err := buildCustomResources(v2BaseDir, customizer, true, logger)
	if err != nil {
		return fmt.Errorf("failed to build resources from downloaded fields: %w", err)
	}
	resources = append(resources, resourcesCustomV2...)
	generators := collectResourceGenerators(resources, customizer, logger)

	for _, g := range generators {
		var code string
		if code, err = g.GenerateCode(); err != nil {
			logger.Errorf("failed to generate code for %s: %s", g.Name(), err)
			continue
		}

		filename, err := writeGeneratedFile(outDir, g.Name(), code)
		if err != nil {
			logger.Errorf("failed to write file %s: %s", g.Name(), err)
			continue
		}
		logger.Debugf("Generated %s with resource %s\n\n", filename, g.Name())
	}
	return nil
}

// collectResourceGenerators filters resources, wires the eligible ones into the
// Client interface builder, and returns the per-resource generators followed by
// the built client generator. Resources excluded from generation produce no
// .generated.go file and no Client interface methods at all — they are
// unsupported and have no hand-written wrapper, so emitting them would only ship
// dead code.
//
// Customizations are NOT (re)applied here: buildResourcesFromDownloadedFields
// already calls ApplyToResource on every resource before Resource.processJSON,
// which is the only consumer of the composed FieldProcessor. Re-applying at this
// point was dead work — processJSON has already run, so the re-wrapped processor
// would never be invoked again, and resourcePath would be re-set to the same
// value.
func collectResourceGenerators(resources []*Resource, customizer CodeCustomizer, logger shared.Logger) []Generatable {
	logger = orDefaultLogger(logger)
	cb := NewClientInfoBuilder()
	customizer.ApplyToClient(cb)
	generators := make([]Generatable, 0, len(resources)+1)
	for _, resource := range resources {
		if customizer.IsExcludedFromGeneration(resource.Name()) {
			logger.Debugf("Skipping generation for excluded resource %s\n", resource.Name())
			continue
		}
		if !customizer.IsExcludedFromClient(resource.Name()) {
			cb.AddResource(resource, customizer.ExcludedClientFunctions(resource))
		}
		generators = append(generators, resource)
	}
	return append(generators, cb.Build())
}

// writeGeneratedFile writes generated file content to a file.
func writeGeneratedFile(outDir string, name string, content string) (string, error) {
	goFile := strcase.ToSnake(name) + ".generated.go"
	goFilePath := filepath.Join(outDir, goFile)
	_ = os.Remove(goFilePath)
	if err := os.WriteFile(goFilePath, []byte(content), 0o644); err != nil { //nolint:gosec
		return goFile, fmt.Errorf("failed to write file %s: %w", goFile, err)
	}
	return goFile, nil
}
