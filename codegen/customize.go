package main

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	AllFieldsCustomizationKeyword = "_all"
	defaultCustomizationsPath     = "customizations.yml"
)

type Generate struct {
	Customizations struct {
		Resources map[string]*ResourceCustomization `yaml:"resources"`
	} `yaml:"customizations"`
}

type ResourceCustomization struct {
	ResourceName string                         `yaml:"-"`
	Fields       map[string]*FieldCustomization `yaml:"fields"`
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
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				err := compositeCustomizationsProcessor(customizationsProcessor)(name, f)
				if err != nil {
					return err
				}
				return currentProcessor(name, f)
			}
		} else {
			resource.FieldProcessor = compositeCustomizationsProcessor(customizationsProcessor)
		}
	}
}

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
	err = yaml.Unmarshal(customizationsYml, &generate)
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

type YamlConfigCodeCustomizer struct {
	Customizations map[string]*ResourceCustomization
}

type CodeCustomizer interface {
	ApplyToResource(resource *Resource)
}

type noopCustomizer struct{}

func (noopCustomizer) ApplyToResource(resource *Resource) {}

func NewCodeCustomizer(customizationsPath string) (CodeCustomizer, error) { //nolint: ireturn
	generate, err := unmarshalCustomizationYaml(customizationsPath)
	if err != nil {
		return nil, err
	}
	return &YamlConfigCodeCustomizer{generate.Customizations.Resources}, nil
}

func (r *YamlConfigCodeCustomizer) ApplyToResource(resource *Resource) {
	for resourceName, resourceCustomization := range r.Customizations {
		if resource.StructName == resourceName {
			resourceCustomization.ApplyTo(resource)
		}
	}
}
