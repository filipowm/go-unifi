package main

import (
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	AllFieldsCustomizationKeyword = "_all"
	defaultCustomizationsPath     = "customizations.yml"
)

type Customizations struct {
	Resources map[string]*ResourceCustomization `yaml:"resources"`
	Client    *ClientCustomization              `yaml:"client"`
}

type Generate struct {
	Customizations *Customizations `yaml:"customizations"`
}

type ResourceCustomization struct {
	ResourceName string                         `yaml:"-"`
	Fields       map[string]*FieldCustomization `yaml:"fields"`
	ResourcePath string                         `yaml:"resourcePath"`
	// QueryParams declares query-string parameters appended to every emitted URL
	// for this resource (after the id segment on id-suffixed get/update/delete
	// URLs, and after the bare path on list/create). This is the first-class
	// alternative to smuggling a "?foo=bar" suffix into resourcePath, which would
	// otherwise produce malformed id-suffixed URLs like ".../x?q=1/%s". See
	// ARCH-19. Keys are rendered in deterministic (sorted) order and URL-encoded.
	QueryParams      map[string]string `yaml:"queryParams"`
	ExcludeFunctions []string          `yaml:"excludeFunctions"`
}

type ClientCustomization struct {
	Imports   []string               `yaml:"imports"`
	Functions []CustomClientFunction `yaml:"functions"`
	// ExcludeResources omits a resource from the generated Client interface only;
	// its <resource>.generated.go (types + private CRUD) is still emitted so a
	// hand-written wrapper (like FirewallZoneMatrix) can wire it up.
	ExcludeResources []string `yaml:"excludeResources"`
	// ExcludeGeneration omits a resource from generation entirely: no
	// <resource>.generated.go file is written at all. Use this for resources that
	// are unsupported and have no hand-written wrapper, so no dead generated code
	// ships. Glob patterns follow the same rules as ExcludeResources.
	ExcludeGeneration []string `yaml:"excludeGeneration"`
}

type FieldCustomization struct {
	FieldName   string             `yaml:"-"`
	Overrides   *FieldInfoOverride `yaml:",inline"`
	IfFieldType string             `yaml:"ifFieldType"`
}

type FieldInfoOverride struct {
	FieldName           *string `yaml:"fieldName"`
	FieldType           *string `yaml:"fieldType"`
	OmitEmpty           *bool   `yaml:"omitEmpty"`
	CustomUnmarshalType *string `yaml:"customUnmarshalType"`
	CustomUnmarshalFunc *string `yaml:"customUnmarshalFunc"`
	JsonPath            *string `yaml:"jsonPath"`
}

func compositeCustomizationsProcessor(customizationsProcessor FieldProcessor) FieldProcessor {
	return func(name string, f *FieldInfo) error {
		err := customizationsProcessor(AllFieldsCustomizationKeyword, f)
		if err != nil {
			return fmt.Errorf("failed applying all fields customization to %s field: %w", name, err)
		}
		err = customizationsProcessor(name, f)
		if err != nil {
			return fmt.Errorf("failed applying customization to %s fields: %w", name, err)
		}
		return nil
	}
}

// ApplyTo applies this resource's customizations to resource exactly once.
//
// It is the single entry point that mutates resource, and it cleanly separates
// the two kinds of override:
//
//   - resource-level overrides (resourcePath) are applied by applyResourceOverrides;
//   - field-level overrides (the per-field FieldProcessor) are composed by
//     applyFieldOverrides.
//
// Ordering contract for the field processor: the YAML field customizations run
// FIRST (the _all keyword, then the named field), then any pre-installed
// processor from customizeResource (the SettingGlobalAp / SettingMgmt /
// SettingUsg special cases). The composed processor is invoked once per field
// during Resource.processJSON; ApplyTo itself never runs it. Because processJSON
// is the only consumer, ApplyTo must be called before processJSON and exactly
// once per resource — collectResourceGenerators no longer re-applies it (that
// second call was dead: it re-wrapped a processor nobody invoked again and
// re-set resourcePath to the same value). See ARCH-21.
func (r *ResourceCustomization) ApplyTo(resource *Resource) {
	if resource.StructName != r.ResourceName {
		return
	}
	r.applyResourceOverrides(resource)
	r.applyFieldOverrides(resource)
}

// applyResourceOverrides applies the resource-level overrides (resourcePath and
// queryParams). excludeFunctions is a resource-level override too, but it is
// consumed directly at client-build time via CodeCustomizer.ExcludedClientFunctions
// rather than mutating the resource here.
func (r *ResourceCustomization) applyResourceOverrides(resource *Resource) {
	if r.ResourcePath != "" {
		resource.ResourcePath = r.ResourcePath
	}
	if len(r.QueryParams) > 0 {
		resource.QueryString = buildQueryString(r.QueryParams)
	}
}

// buildQueryString renders params into a deterministic, URL-encoded query string
// WITHOUT a leading "?", e.g. {"a":"1","b":"2"} -> "a=1&b=2". Keys are sorted so
// the generated output is reproducible run-to-run. Templates prepend the "?".
func buildQueryString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	values := url.Values{}
	for _, k := range keys {
		values.Set(k, params[k])
	}
	return values.Encode()
}

// applyFieldOverrides composes the YAML field customizations with any processor
// already installed on the resource, preserving the ordering contract documented
// on ApplyTo (YAML customizations first, then the pre-installed processor).
func (r *ResourceCustomization) applyFieldOverrides(resource *Resource) {
	customizationsProcessor := compositeCustomizationsProcessor(r.toFieldProcessor())
	currentProcessor := resource.FieldProcessor
	if currentProcessor == nil {
		resource.FieldProcessor = customizationsProcessor
		return
	}
	resource.FieldProcessor = func(name string, f *FieldInfo) error {
		if err := customizationsProcessor(name, f); err != nil {
			return err
		}
		return currentProcessor(name, f)
	}
}

//nolint:nestif,cyclop
func (r *ResourceCustomization) toFieldProcessor() FieldProcessor {
	return func(name string, f *FieldInfo) error {
		if fc, ok := r.Fields[name]; ok && fc.Overrides != nil && (fc.IfFieldType == "" || fc.IfFieldType == f.FieldType) {
			if fc.Overrides.FieldType != nil {
				f.FieldType = *fc.Overrides.FieldType
			}
			if fc.Overrides.CustomUnmarshalType != nil {
				f.CustomUnmarshalType = *fc.Overrides.CustomUnmarshalType
			}
			if fc.Overrides.OmitEmpty != nil {
				f.OmitEmpty = *fc.Overrides.OmitEmpty
			}
			if fc.Overrides.CustomUnmarshalFunc != nil {
				f.CustomUnmarshalFunc = *fc.Overrides.CustomUnmarshalFunc
			}
			if fc.Overrides.FieldName != nil {
				f.FieldName = *fc.Overrides.FieldName
			}
			if fc.Overrides.JsonPath != nil {
				f.JSONName = *fc.Overrides.JsonPath
			}
		}
		return nil
	}
}

//go:embed customizations.yml
var defaultCustomizationYml []byte

func readCustomizationsYml(customizationsPath string) ([]byte, error) {
	if customizationsPath == "" || customizationsPath == defaultCustomizationsPath {
		return defaultCustomizationYml, nil
	}
	customizations, err := os.ReadFile(customizationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed reading customizations file %s: %w", customizationsPath, err)
	}
	return customizations, nil
}

func unmarshalCustomizationYaml(customizationsPath string) (*Generate, error) {
	var generate Generate
	customizationsYml, err := readCustomizationsYml(customizationsPath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(customizationsYml, &generate) //nolint: musttag
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling YAML to Generate structure: %w", err)
	}
	// Assign ResourceName and FieldName based on the map keys
	for resourceName, resource := range generate.Customizations.Resources {
		resource.ResourceName = resourceName
		for fieldName, field := range resource.Fields {
			field.FieldName = fieldName
		}
	}

	return &generate, nil
}

type CodeCustomizer struct {
	Customizations Customizations
	// logger receives customizer diagnostics (the unknown-excludeFunctions
	// warning). It is injected by generate()/generateCode; when nil, log()
	// falls back to the package-global logger so a directly-built customizer
	// (tests) still works. See TEST-13.
	logger Logger
}

func NewCodeCustomizer(customizationsPath string) (*CodeCustomizer, error) {
	generate, err := unmarshalCustomizationYaml(customizationsPath)
	if err != nil {
		return nil, err
	}
	if generate.Customizations == nil {
		generate.Customizations = &Customizations{}
	}
	return &CodeCustomizer{Customizations: *generate.Customizations}, nil
}

func (r *CodeCustomizer) IsExcludedFromClient(resourceName string) bool {
	if r.Customizations.Client == nil {
		return false
	}
	for _, pattern := range r.Customizations.Client.ExcludeResources {
		if matchesExcludePattern(pattern, resourceName) {
			return true
		}
	}
	return false
}

// IsExcludedFromGeneration reports whether the resource must be skipped entirely
// at the generation step, so no <resource>.generated.go file is written for it.
func (r *CodeCustomizer) IsExcludedFromGeneration(resourceName string) bool {
	if r.Customizations.Client == nil {
		return false
	}
	for _, pattern := range r.Customizations.Client.ExcludeGeneration {
		if matchesExcludePattern(pattern, resourceName) {
			return true
		}
	}
	return false
}

// ExcludedClientFunctions returns the standard CRUD action names excluded from
// client generation for res (nil if none configured). Unknown action names are
// warned and ignored so a typo can't silently leave a method in place.
func (r *CodeCustomizer) ExcludedClientFunctions(res *Resource) []string {
	if r.Customizations.Resources == nil {
		return nil
	}
	rc, ok := r.Customizations.Resources[res.Name()]
	if !ok || rc == nil || len(rc.ExcludeFunctions) == 0 {
		return nil
	}
	valid := standardActionNames(res)
	for _, a := range rc.ExcludeFunctions {
		if !valid[a] {
			r.log().Warnf("excludeFunctions: unknown action %q for resource %s (ignored)", a, res.Name())
		}
	}
	return rc.ExcludeFunctions
}

// matchesExcludePattern reports whether name matches a glob-style pattern,
// where a leading and/or trailing "*" acts as a wildcard:
//
//	"*x*" -> contains, "x*" -> prefix, "*x" -> suffix, "x" -> exact.
//
// A bare "*" (or "**") has no inner content and matches everything.
func matchesExcludePattern(pattern, name string) bool {
	prefixWildcard := strings.HasPrefix(pattern, "*")
	suffixWildcard := strings.HasSuffix(pattern, "*")
	switch {
	case prefixWildcard && suffixWildcard:
		if len(pattern) <= 2 {
			return true
		}
		return strings.Contains(name, pattern[1:len(pattern)-1])
	case prefixWildcard:
		return strings.HasSuffix(name, pattern[1:])
	case suffixWildcard:
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	default:
		return name == pattern
	}
}

func (r *CodeCustomizer) ApplyToResource(resource *Resource) {
	for resourceName, resourceCustomization := range r.Customizations.Resources {
		if resource.StructName == resourceName {
			resourceCustomization.ApplyTo(resource)
		}
	}
}

func (r *CodeCustomizer) ApplyToClient(client *ClientInfoBuilder) {
	if client == nil || r.Customizations.Client == nil {
		return
	}
	client.AddFunctions(r.Customizations.Client.Functions)
	client.AddImports(r.Customizations.Client.Imports)
}

// log returns the customizer's injected logger, or the package-global fallback
// when none was set. See TEST-13.
func (r *CodeCustomizer) log() Logger {
	return orDefaultLogger(r.logger)
}
