package main

import (
	_ "embed"
	"fmt"
	"os"
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
	ResourceName     string                         `yaml:"-"`
	Fields           map[string]*FieldCustomization `yaml:"fields"`
	ResourcePath     string                         `yaml:"resourcePath"`
	ExcludeFunctions []string                       `yaml:"excludeFunctions"`
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

func (r *ResourceCustomization) ApplyTo(resource *Resource) {
	if resource.StructName == r.ResourceName {
		currentProcessor := resource.FieldProcessor
		customizationsProcessor := r.toFieldProcessor()
		if currentProcessor != nil {
			// create composite processor with existing processor, first running pre-defined customizations, then user-defined
			r.applyCurrentProcessor(resource, customizationsProcessor, currentProcessor)
		} else {
			resource.FieldProcessor = compositeCustomizationsProcessor(customizationsProcessor)
		}
	}
}

func (r *ResourceCustomization) applyCurrentProcessor(resource *Resource, customizationsProcessor FieldProcessor, currentProcessor FieldProcessor) {
	resource.FieldProcessor = func(name string, f *FieldInfo) error {
		err := compositeCustomizationsProcessor(customizationsProcessor)(name, f)
		if err != nil {
			return err
		}
		return currentProcessor(name, f)
	}
	if r.ResourcePath != "" {
		resource.ResourcePath = r.ResourcePath
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
}

func NewCodeCustomizer(customizationsPath string) (*CodeCustomizer, error) {
	generate, err := unmarshalCustomizationYaml(customizationsPath)
	if err != nil {
		return nil, err
	}
	if generate.Customizations == nil {
		generate.Customizations = &Customizations{}
	}
	return &CodeCustomizer{*generate.Customizations}, nil
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
			log.Warnf("excludeFunctions: unknown action %q for resource %s (ignored)", a, res.Name())
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
