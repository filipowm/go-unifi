package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
)

type replacement struct {
	Old string
	New string
}

var fieldReps = []replacement{
	{"Dhcpdv6", "DHCPDV6"},

	{"Dhcpd", "DHCPD"},
	{"Idx", "IDX"},
	{"Ipsec", "IPSec"},
	{"Ipv6", "IPV6"},
	{"Openvpn", "OpenVPN"},
	{"Tftp", "TFTP"},
	{"Wlangroup", "WLANGroup"},

	{"Bc", "Broadcast"},
	{"Dhcp", "DHCP"},
	{"Dns", "DNS"},
	{"Dpi", "DPI"},
	{"Dtim", "DTIM"},
	{"Firewallgroup", "FirewallGroup"},
	{"Fixedip", "FixedIP"},
	{"Icmp", "ICMP"},
	{"Id", "ID"},
	{"Igmp", "IGMP"},
	{"Ip", "IP"},
	{"Leasetime", "LeaseTime"},
	{"Mac", "MAC"},
	{"Mcastenhance", "MulticastEnhance"},
	{"Minrssi", "MinRSSI"},
	{"Monthdays", "MonthDays"},
	{"Nat", "NAT"},
	{"Networkconf", "Network"},
	{"Networkgroup", "NetworkGroup"},
	{"Pd", "PD"},
	{"Pmf", "PMF"},
	{"Portconf", "PortProfile"},
	{"Qos", "QOS"},
	{"Radiusprofile", "RADIUSProfile"},
	{"Radius", "RADIUS"},
	{"Ssid", "SSID"},
	{"Startdate", "StartDate"},
	{"Starttime", "StartTime"},
	{"Stopdate", "StopDate"},
	{"Stoptime", "StopTime"},
	{"Tcp", "TCP"},
	{"Udp", "UDP"},
	{"Usergroup", "UserGroup"},
	{"Utc", "UTC"},
	{"Vlan", "VLAN"},
	{"Vpn", "VPN"},
	{"Wan", "WAN"},
	{"Wep", "WEP"},
	{"Wlan", "WLAN"},
	{"Wpa", "WPA"},
}

var fileReps = []replacement{
	{"WlanConf", "WLAN"},
	{"Dhcp", "DHCP"},
	{"Wlan", "WLAN"},
	{"NetworkConf", "Network"},
	{"PortConf", "PortProfile"},
	{"RadiusProfile", "RADIUSProfile"},
	{"ApGroups", "APGroup"},
}

type FieldProcessor func(name string, f *FieldInfo) error

type Resource struct {
	StructName     string
	ResourcePath   string
	Types          map[string]*FieldInfo
	FieldProcessor FieldProcessor
	V2             bool
}

func (r *Resource) IsV2() bool {
	return r.V2
}

func (r *Resource) BaseType() *FieldInfo {
	return r.Types[r.StructName]
}

type FieldInfo struct {
	FieldName              string
	JSONName               string
	FieldType              string
	FieldValidation        string
	FieldValidationComment string
	OmitEmpty              bool
	IsArray                bool
	Fields                 map[string]*FieldInfo
	CustomUnmarshalType    string
	CustomUnmarshalFunc    string
}

func NewResource(structName string, resourcePath string) *Resource {
	baseType := NewFieldInfo(structName, resourcePath, "struct", "", "", false, false, "")
	resource := &Resource{
		StructName:   structName,
		ResourcePath: resourcePath,
		Types: map[string]*FieldInfo{
			structName: baseType,
		},
		FieldProcessor: func(name string, f *FieldInfo) error { return nil },
	}

	// Since template files iterate through map keys in sorted order, these initial fields
	// are named such that they stay at the top for consistency. The spacer items create a
	// blank line in the resulting generated file.
	//
	// This hack is here for stability of the generated code, but can be removed if desired.
	baseType.Fields = map[string]*FieldInfo{
		"   ID":      NewFieldInfo("ID", "_id", "string", "", "", true, false, ""),
		"   SiteID":  NewFieldInfo("SiteID", "site_id", "string", "", "", true, false, ""),
		"   _Spacer": nil,

		"  Hidden":   NewFieldInfo("Hidden", "attr_hidden", "bool", "", "", true, false, ""),
		"  HiddenID": NewFieldInfo("HiddenID", "attr_hidden_id", "string", "", "", true, false, ""),
		"  NoDelete": NewFieldInfo("NoDelete", "attr_no_delete", "bool", "", "", true, false, ""),
		"  NoEdit":   NewFieldInfo("NoEdit", "attr_no_edit", "bool", "", "", true, false, ""),
		"  _Spacer":  nil,

		" _Spacer": nil,
	}

	if resource.IsSetting() {
		resource.ResourcePath = strcase.ToSnake(strings.TrimPrefix(structName, "Setting"))
	}
	return resource
}

func NewFieldInfo(fieldName, jsonName, fieldType, fieldValidation, fieldValidationComment string, omitempty bool, isArray bool, customUnmarshalType string) *FieldInfo {
	return &FieldInfo{
		FieldName:              fieldName,
		JSONName:               jsonName,
		FieldType:              fieldType,
		FieldValidation:        fieldValidation,
		FieldValidationComment: fieldValidationComment,
		OmitEmpty:              omitempty,
		IsArray:                isArray,
		CustomUnmarshalType:    customUnmarshalType,
	}
}

func cleanName(name string, reps []replacement) string {
	for _, rep := range reps {
		name = strings.ReplaceAll(name, rep.Old, rep.New)
	}

	return name
}

func (r *Resource) IsSetting() bool {
	return strings.HasPrefix(r.StructName, "Setting")
}

func (r *Resource) Name() string {
	return r.StructName
}

func (r *Resource) processFields(fields map[string]interface{}) {
	t := r.Types[r.StructName]
	for name, validation := range fields {
		fieldInfo, err := r.fieldInfoFromValidation(name, validation)
		if err != nil {
			continue
		}

		t.Fields[fieldInfo.FieldName] = fieldInfo
	}
}

func (r *Resource) fieldInfoFromValidation(name string, validation interface{}) (*FieldInfo, error) {
	fieldName := strcase.ToCamel(name)
	fieldName = cleanName(fieldName, fieldReps)

	empty := &FieldInfo{}
	var fieldInfo *FieldInfo

	switch validation := validation.(type) {
	case []interface{}:
		if len(validation) == 0 {
			fieldInfo = NewFieldInfo(fieldName, name, "string", "", "", false, true, "")
			err := r.FieldProcessor(fieldName, fieldInfo)
			return fieldInfo, err
		}
		if len(validation) > 1 {
			return empty, fmt.Errorf("unknown validation %#v", validation)
		}

		fieldInfo, err := r.fieldInfoFromValidation(name, validation[0])
		if err != nil {
			return empty, err
		}

		fieldInfo.OmitEmpty = true
		fieldInfo.IsArray = true

		err = r.FieldProcessor(fieldName, fieldInfo)
		return fieldInfo, err

	case map[string]interface{}:
		typeName := r.StructName + fieldName

		result := NewFieldInfo(fieldName, name, typeName, "", "", true, false, "")
		result.Fields = make(map[string]*FieldInfo)

		for name, fv := range validation {
			child, err := r.fieldInfoFromValidation(name, fv)
			if err != nil {
				return empty, err
			}

			result.Fields[child.FieldName] = child
		}

		err := r.FieldProcessor(fieldName, result)
		r.Types[typeName] = result
		return result, err

	case string:
		fieldValidationComment := validation
		normalized := normalizeValidation(validation)

		omitEmpty := false

		switch {
		case normalized == "falsetrue" || normalized == "truefalse":
			fieldInfo = NewFieldInfo(fieldName, name, "bool", "", "", omitEmpty, false, "")
			return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
		default:
			if _, err := strconv.ParseFloat(normalized, 64); err == nil {
				if normalized == "09" || normalized == "09.09" {
					fieldValidationComment = ""
				}

				if strings.Contains(normalized, ".") {
					if strings.Contains(validation, "\\.){3}") {
						break
					}

					omitEmpty = true
					fieldInfo = NewFieldInfo(fieldName, name, "float64", "", fieldValidationComment, omitEmpty, false, "")
					return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
				}

				fieldValidation := defineFieldValidation(fieldValidationComment)
				omitEmpty = true
				fieldInfo = NewFieldInfo(fieldName, name, "int", fieldValidation, fieldValidationComment, omitEmpty, false, "")
				fieldInfo.CustomUnmarshalType = "emptyStringInt"
				return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
			}
		}
		if validation != "" && normalized != "" {
			log.Tracef("normalize %q to %q", validation, normalized)
		}

		fieldValidation := defineFieldValidation(fieldValidationComment)
		omitEmpty = omitEmpty || (!strings.Contains(validation, "^$") && !strings.HasSuffix(fieldName, "ID"))
		fieldInfo = NewFieldInfo(fieldName, name, "string", fieldValidation, fieldValidationComment, omitEmpty, false, "")
		return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
	}

	return empty, fmt.Errorf("unable to determine type from validation %q", validation)
}

func (r *Resource) processJSON(b []byte) error {
	var fields map[string]interface{}
	err := json.Unmarshal(b, &fields)
	if err != nil {
		return err
	}

	r.processFields(fields)

	return nil
}

//go:embed api.go.tmpl
var apiGoTemplate string

//go:embed apiv2.go.tmpl
var apiGoV2Template string

func (r *Resource) GenerateCode() (string, error) {
	if r.IsV2() {
		return generateCodeFromTemplate("apiv2.go.tmpl", apiGoV2Template, r)
	}
	return generateCodeFromTemplate("api.go.tmpl", apiGoTemplate, r)
}

func normalizeValidation(re string) string {
	re = strings.ReplaceAll(re, "\\d", "[0-9]")
	re = strings.ReplaceAll(re, "[-+]?", "")
	re = strings.ReplaceAll(re, "[+-]?", "")
	re = strings.ReplaceAll(re, "[-]?", "")
	re = strings.ReplaceAll(re, "\\.", ".")
	re = strings.ReplaceAll(re, "[.]?", ".")

	quants := regexp.MustCompile(`\{\d*,?\d*\}|\*|\+|\?`)
	re = quants.ReplaceAllString(re, "")

	control := regexp.MustCompile(`[\(\[\]\)\|\-\$\^]`)
	re = control.ReplaceAllString(re, "")

	re = strings.TrimPrefix(re, "^")
	re = strings.TrimSuffix(re, "$")

	return re
}

var skippable = []string{"AuthenticationRequest.json", "Setting.json", "Wall.json"}

func buildResourcesFromDownloadedFields(fieldsDir string, customizer CodeCustomizer, v2 bool) ([]*Resource, error) {
	fieldsFiles, err := os.ReadDir(fieldsDir)
	if err != nil {
		return nil, fmt.Errorf("unable to read fields directory %s: %w", fieldsDir, err)
	}

	resources := make([]*Resource, 0)

	for _, fieldsFile := range fieldsFiles {
		name := fieldsFile.Name()
		ext := filepath.Ext(name)

		if slices.Contains(skippable, name) || ext != ".json" {
			continue
		}
		log.Debugf("Processing %s...", fieldsFile.Name())
		name = name[:len(name)-len(ext)]

		urlPath := strings.ToLower(name)
		structName := cleanName(name, fileReps)

		fieldsFilePath := filepath.Join(fieldsDir, fieldsFile.Name())
		b, err := os.ReadFile(fieldsFilePath)
		if err != nil {
			log.Warnf("skipping file %s: %s", fieldsFile.Name(), err)
			continue
		}

		resource := NewResource(structName, urlPath)
		customizeResource(resource, v2)
		customizer.ApplyToResource(resource)

		err = resource.processJSON(b)
		if err != nil {
			log.Warnf("skipping file %s: %s", fieldsFile.Name(), err)
			continue
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func buildCustomResources(dir string, customizer CodeCustomizer, v2 bool) ([]*Resource, error) {
	return buildResourcesFromDownloadedFields(dir, customizer, v2)
}

func customizeBaseType(resource *Resource) {
	baseType := resource.BaseType()
	if resource.IsSetting() {
		baseType.Fields[" Key"] = NewFieldInfo("Key", "key", "string", "", "", false, false, "")

		if resource.StructName == "SettingUsg" {
			// Removed in v7, retaining for backwards compatibility
			baseType.Fields["MdnsEnabled"] = NewFieldInfo("MdnsEnabled", "mdns_enabled", "bool", "", "", false, false, "")
		}
	}
	switch {
	case resource.IsSetting():
		baseType.Fields[" Key"] = NewFieldInfo("Key", "key", "string", "", "", false, false, "")

		if resource.StructName == "SettingUsg" {
			// Removed in v7, retaining for backwards compatibility
			baseType.Fields["MdnsEnabled"] = NewFieldInfo("MdnsEnabled", "mdns_enabled", "bool", "", "", false, false, "")
		}
	case resource.StructName == "Device":
		baseType.Fields[" MAC"] = NewFieldInfo("MAC", "mac", "string", createValidations(validation{v: mac}), "", true, false, "")
		baseType.Fields["Adopted"] = NewFieldInfo("Adopted", "adopted", "bool", "", "", false, false, "")
		baseType.Fields["Model"] = NewFieldInfo("Model", "model", "string", "", "", true, false, "")
		baseType.Fields["State"] = NewFieldInfo("State", "state", "DeviceState", "", "", false, false, "")
		baseType.Fields["Type"] = NewFieldInfo("Type", "type", "string", "", "", true, false, "")
	case resource.StructName == "User":
		baseType.Fields[" IP"] = NewFieldInfo("IP", "ip", "string", createValidations(validation{v: ip}), "non-generated field", true, false, "")
		baseType.Fields[" DevIdOverride"] = NewFieldInfo("DevIdOverride", "dev_id_override", "int", "", "non-generated field", true, false, "")
	case resource.StructName == "WLAN":
		// this field removed in v6, retaining for backwards compatibility
		baseType.Fields["WLANGroupID"] = NewFieldInfo("WLANGroupID", "wlangroup_id", "string", "", "", false, false, "")
	}
}

func customizeResource(resource *Resource, v2 bool) {
	customizeBaseType(resource)
	if v2 {
		resource.V2 = true
	}

	switch resource.StructName {
	case "SettingGlobalAp":
		resource.FieldProcessor = func(name string, f *FieldInfo) error {
			if strings.HasPrefix(name, "6E") {
				f.FieldName = strings.Replace(f.FieldName, "6E", "SixE", 1)
			}

			return nil
		}
	case "SettingMgmt":
		sshKeyField := NewFieldInfo(resource.StructName+"XSshKeys", "x_ssh_keys", "struct", "", "", false, false, "")
		sshKeyField.Fields = map[string]*FieldInfo{
			"name":        NewFieldInfo("Name", "name", "string", "", "", false, false, ""),
			"keyType":     NewFieldInfo("KeyType", "type", "string", "", "", false, false, ""),
			"key":         NewFieldInfo("Key", "key", "string", "", "", false, false, ""),
			"comment":     NewFieldInfo("Comment", "comment", "string", "", "", false, false, ""),
			"date":        NewFieldInfo("Date", "date", "string", "", "", false, false, ""),
			"fingerprint": NewFieldInfo("Fingerprint", "fingerprint", "string", "", "", false, false, ""),
		}
		resource.Types[sshKeyField.FieldName] = sshKeyField

		resource.FieldProcessor = func(name string, f *FieldInfo) error {
			if name == "XSshKeys" {
				f.FieldType = sshKeyField.FieldName
			}
			return nil
		}
	case "SettingUsg":
		resource.FieldProcessor = func(name string, f *FieldInfo) error {
			if strings.HasSuffix(name, "Timeout") && name != "ArpCacheTimeout" {
				f.FieldType = "int"
				f.CustomUnmarshalType = "emptyStringInt"
			}
			return nil
		}
	}
}
