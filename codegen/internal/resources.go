package internal

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/filipowm/go-unifi/codegen/shared"
	"github.com/iancoleman/strcase"
)

// strictEnvVar, when set to a truthy value, promotes every dropped field
// (failed type inference) and every CamelCase field-name collision from a WARN
// into a hard generation error. It is OFF by default so the daily auto-regen
// keeps its current best-effort behavior; CI can opt in by exporting the var.
const strictEnvVar = "UNIFI_CODEGEN_STRICT"

// strictMode reports whether strict generation is enabled via the environment.
// Accepted truthy values: "1", "true", "yes", "on" (case-insensitive).
func strictMode() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(strictEnvVar))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// strictViolationError wraps a dropped-field or field-collision error produced under
// strict mode. It lets the file-processing loop distinguish a strict policy
// failure (which must abort the whole generation) from an ordinary per-file
// error like malformed JSON (which is still skipped with a warning). Detected
// with errors.As.
type strictViolationError struct {
	err error
}

func (e *strictViolationError) Error() string { return e.err.Error() }
func (e *strictViolationError) Unwrap() error { return e.err }

// sortedKeys returns the keys of a string-keyed map in deterministic (sorted)
// order. Ranging over this instead of the raw map makes field processing —
// including collision resolution and the resulting generated output —
// reproducible run-to-run regardless of Go's randomized map iteration order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

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
	StructName   string
	ResourcePath string
	// QueryString is the URL-encoded query parameters (without a leading "?")
	// declared via customizations.yml queryParams, e.g. "includeSystemFeatures=true".
	// It is appended AFTER the id segment on id-suffixed URLs so the id never lands
	// behind the query string. Empty when the resource declares no query params.
	QueryString    string
	Types          map[string]*FieldInfo
	FieldProcessor FieldProcessor
	V2             bool
	// logger receives this resource's generation diagnostics (dropped-field and
	// collision warnings). It is injected by buildResourcesFromDownloadedFields;
	// when nil (e.g. a Resource built directly in a test), log() falls back to
	// the package-global logger.
	logger shared.Logger
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

// QuerySuffix returns the query string ready to splice into a URL format string:
// "?a=1&b=2" when query params are declared, or "" otherwise. Templates append
// this AFTER the id segment so an id never lands behind the query string.
func (r *Resource) QuerySuffix() string {
	if r.QueryString == "" {
		return ""
	}
	// Templates splice this into a fmt.Sprintf FORMAT-string literal, so any '%'
	// produced by url.Values.Encode (e.g. '&' -> '%26') must be doubled to stay a
	// literal percent rather than a (malformed) verb. Today's only param is
	// %-free, so this is zero-diff hardening against future queryParams.
	return "?" + strings.ReplaceAll(r.QueryString, "%", "%%")
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

// log returns the resource's injected logger, or the package-global fallback
// when none was set. Keeping access behind this accessor lets directly-built
// Resource values (tests) log without panicking on a nil field.
func (r *Resource) log() shared.Logger {
	return orDefaultLogger(r.logger)
}

// validateResourcePath guards against the footgun: a query string smuggled
// into ResourcePath ("described-features?includeSystemFeatures=true") makes the
// id-suffixed get/update/delete templates emit ".../described-features?q=1/%s",
// where the id lands AFTER the query string — a never-correct URL. The first-class
// fix is the queryParams customization (rendered via QuerySuffix); this guard
// ensures nobody re-introduces the raw "?" form. In strict mode it is a hard
// error; otherwise it warns. It is intentionally unconditional: even list-only
// resources should migrate to queryParams rather than rely on the query happening
// to terminate the URL cleanly.
func (r *Resource) validateResourcePath() error {
	if !strings.Contains(r.ResourcePath, "?") {
		return nil
	}
	msg := fmt.Sprintf("resource %s: resourcePath %q contains a raw query string; use the queryParams customization instead so id-suffixed URLs stay well-formed",
		r.StructName, r.ResourcePath)
	if strictMode() {
		return &strictViolationError{errors.New(msg)}
	}
	r.log().Warnf("%s", msg)
	return nil
}

func (r *Resource) processFields(fields map[string]any) error {
	t := r.Types[r.StructName]
	// Process JSON keys in deterministic (sorted) order so collision resolution
	// and the resulting field map are reproducible run-to-run (Go randomizes map
	// iteration).
	for _, name := range sortedKeys(fields) {
		validation := fields[name]
		fieldInfo, err := r.fieldInfoFromValidation(name, validation, false)
		if err != nil {
			if dropErr := r.reportDroppedField(name, validation, err); dropErr != nil {
				return dropErr
			}
			continue
		}

		if collErr := r.reportFieldNameCollision(t.Fields, fieldInfo); collErr != nil {
			return collErr
		}

		t.Fields[fieldInfo.FieldName] = fieldInfo
	}
	return nil
}

// reportDroppedField records that the field named jsonKey could not be inferred
// and is being dropped from the generated struct. In strict mode it returns a
// hard error (failing generation); otherwise it logs a WARN and returns nil so
// the caller skips only this field. The raw validation is included so a human
// (or CI) can see exactly what shape was unrecognized.
func (r *Resource) reportDroppedField(jsonKey string, validation any, err error) error {
	if strictMode() {
		return &strictViolationError{fmt.Errorf("resource %s: dropping field %q (validation %#v): %w", r.StructName, jsonKey, validation, err)}
	}
	r.log().Warnf("resource %s: dropping field %q (validation %#v): %s", r.StructName, jsonKey, validation, err)
	return nil
}

// reportFieldNameCollision detects when fieldInfo's derived Go FieldName already
// exists in fields under a DIFFERENT JSONName — i.e. two distinct controller
// JSON keys normalized (strcase.ToCamel + cleanName acronym substitution) to the
// same exported identifier, so assigning the second would silently overwrite the
// first and lose a field + its json tag. In strict mode this is a hard error;
// otherwise it logs a WARN. The deterministic key sort in the callers guarantees
// which JSON key is processed first, so the surviving (overwritten) winner is
// stable across runs. Returns nil when there is no collision.
func (r *Resource) reportFieldNameCollision(fields map[string]*FieldInfo, fieldInfo *FieldInfo) error {
	existing, ok := fields[fieldInfo.FieldName]
	if !ok || existing == nil || existing.JSONName == fieldInfo.JSONName {
		return nil
	}
	if strictMode() {
		return &strictViolationError{fmt.Errorf("resource %s: CamelCase collision on Go field %q between JSON keys %q and %q",
			r.StructName, fieldInfo.FieldName, existing.JSONName, fieldInfo.JSONName)}
	}
	r.log().Warnf("resource %s: CamelCase collision on Go field %q between JSON keys %q and %q (the latter overwrites the former)",
		r.StructName, fieldInfo.FieldName, existing.JSONName, fieldInfo.JSONName)
	return nil
}

func (r *Resource) fieldInfoFromValidation(name string, validation any, isArray bool) (*FieldInfo, error) {
	fieldName := strcase.ToCamel(name)
	fieldName = cleanName(fieldName, fieldReps)

	switch validation := validation.(type) {
	case []any:
		return r.fieldInfoFromArray(fieldName, name, validation)
	case map[string]any:
		return r.fieldInfoFromMap(fieldName, name, validation)
	case string:
		return r.fieldInfoFromString(fieldName, name, validation, isArray)
	}

	return &FieldInfo{}, fmt.Errorf("unable to determine type from validation %q", validation)
}

func (r *Resource) fieldInfoFromArray(fieldName, name string, validation []any) (*FieldInfo, error) {
	empty := &FieldInfo{}

	if len(validation) == 0 {
		fieldInfo := NewFieldInfo(fieldName, name, "string", "", "", false, true, "")
		err := r.FieldProcessor(fieldName, fieldInfo)
		return fieldInfo, err
	}
	if len(validation) > 1 {
		return empty, fmt.Errorf("unknown validation %#v", validation)
	}

	fieldInfo, err := r.fieldInfoFromValidation(name, validation[0], true)
	if err != nil {
		return empty, err
	}

	fieldInfo.OmitEmpty = true
	fieldInfo.IsArray = true

	err = r.FieldProcessor(fieldName, fieldInfo)
	return fieldInfo, err
}

func (r *Resource) fieldInfoFromMap(fieldName, name string, validation map[string]any) (*FieldInfo, error) {
	empty := &FieldInfo{}

	typeName := r.StructName + fieldName

	result := NewFieldInfo(fieldName, name, typeName, "", "", true, false, "")
	result.Fields = make(map[string]*FieldInfo)

	// Process nested keys in deterministic (sorted) order, mirroring
	// processFields, so nested-struct field ordering and collision resolution are
	// reproducible.
	for _, childName := range sortedKeys(validation) {
		fv := validation[childName]
		child, err := r.fieldInfoFromValidation(childName, fv, false)
		if err != nil {
			// Skip ONLY the failing nested child rather than discarding the whole
			// nested struct and its siblings (the old behavior). In strict mode the
			// drop is promoted to a hard error.
			if dropErr := r.reportDroppedField(typeName+"."+childName, fv, err); dropErr != nil {
				return empty, dropErr
			}
			continue
		}

		if collErr := r.reportFieldNameCollision(result.Fields, child); collErr != nil {
			return empty, collErr
		}

		result.Fields[child.FieldName] = child
	}

	err := r.FieldProcessor(fieldName, result)
	r.Types[typeName] = result
	return result, err
}

func (r *Resource) fieldInfoFromString(fieldName, name, validation string, isArray bool) (*FieldInfo, error) {
	fieldValidationComment := validation
	normalized := normalizeValidation(validation)

	if normalized == "falsetrue" || normalized == "truefalse" {
		fieldInfo := NewFieldInfo(fieldName, name, "bool", "", "", false, false, "")
		return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
	}

	if fieldInfo, handled, err := r.numericFieldInfo(fieldName, name, validation, normalized, isArray); handled {
		return fieldInfo, err
	}

	if validation != "" && normalized != "" {
		r.log().Tracef("normalize %q to %q", validation, normalized)
	}

	fieldValidation := defineFieldValidation(fieldValidationComment, isArray)
	omitEmpty := !strings.Contains(validation, "^$") && !strings.HasSuffix(fieldName, "ID")
	fieldInfo := NewFieldInfo(fieldName, name, "string", fieldValidation, fieldValidationComment, omitEmpty, false, "")
	return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
}

// numericFieldInfo handles validations that normalize to a numeric form (int or
// float64). The returned bool reports whether the numeric branch handled the
// field; when it is false the caller must fall through to string handling. This
// preserves the original `break` semantics for the IP-octet pattern (`\.){3}`),
// which builds a string field from the original validation comment.
func (r *Resource) numericFieldInfo(fieldName, name, validation, normalized string, isArray bool) (*FieldInfo, bool, error) {
	if _, err := strconv.ParseFloat(normalized, 64); err != nil {
		// Not numeric: caller falls through to string handling.
		return nil, false, nil //nolint:nilerr // err only signals "not a number", nothing to propagate
	}

	fieldValidationComment := validation
	if normalized == "09" || normalized == "09.09" {
		fieldValidationComment = ""
	}

	if strings.Contains(normalized, ".") {
		if strings.Contains(validation, "\\.){3}") {
			return nil, false, nil
		}

		fieldInfo := NewFieldInfo(fieldName, name, "float64", "", fieldValidationComment, true, false, "")
		return fieldInfo, true, r.FieldProcessor(fieldName, fieldInfo)
	}

	fieldValidation := defineFieldValidation(fieldValidationComment, isArray)
	fieldInfo := NewFieldInfo(fieldName, name, "int", fieldValidation, fieldValidationComment, true, false, "")
	fieldInfo.CustomUnmarshalType = "emptyStringInt"
	return fieldInfo, true, r.FieldProcessor(fieldName, fieldInfo)
}

func (r *Resource) processJSON(b []byte) error {
	var fields map[string]any
	err := json.Unmarshal(b, &fields)
	if err != nil {
		return err
	}

	return r.processFields(fields)
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

func buildResourcesFromDownloadedFields(fieldsDir string, customizer CodeCustomizer, v2 bool, logger shared.Logger) ([]*Resource, error) {
	logger = orDefaultLogger(logger)
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
		logger.Debugf("Processing %s...", fieldsFile.Name())
		name = name[:len(name)-len(ext)]

		urlPath := strings.ToLower(name)
		structName := cleanName(name, fileReps)

		fieldsFilePath := filepath.Join(fieldsDir, fieldsFile.Name())
		b, err := os.ReadFile(fieldsFilePath)
		if err != nil {
			logger.Warnf("skipping file %s: %s", fieldsFile.Name(), err)
			continue
		}

		resource := NewResource(structName, urlPath)
		resource.logger = logger
		customizeResource(resource, v2)
		customizer.ApplyToResource(resource)

		// Guard against a raw query string smuggled into resourcePath, which would
		// emit malformed id-suffixed URLs (id after the query).
		if err = resource.validateResourcePath(); err != nil {
			var sv *strictViolationError
			if errors.As(err, &sv) {
				return nil, fmt.Errorf("strict mode: %s: %w", fieldsFile.Name(), err)
			}
			logger.Warnf("skipping file %s: %s", fieldsFile.Name(), err)
			continue
		}

		err = resource.processJSON(b)
		if err != nil {
			// A strict-mode field-drop/collision must abort the whole generation,
			// not be quietly skipped like a malformed-JSON file.
			var sv *strictViolationError
			if errors.As(err, &sv) {
				return nil, fmt.Errorf("strict mode: %s: %w", fieldsFile.Name(), err)
			}
			logger.Warnf("skipping file %s: %s", fieldsFile.Name(), err)
			continue
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func buildCustomResources(dir string, customizer CodeCustomizer, v2 bool, logger shared.Logger) ([]*Resource, error) {
	return buildResourcesFromDownloadedFields(dir, customizer, v2, logger)
}

// buildMergedResources builds the Internal-API resource set from two committed
// field snapshots: floorFieldsDir (the supported-version floor) and fieldsDir
// (the newest field shapes). It returns their union by struct name with the
// newest snapshot winning — so newest field shapes apply on top of the floor,
// resources retired before the floor (absent from BOTH snapshots) never appear,
// and resources added after the floor are kept. An empty floorFieldsDir disables
// the merge and yields the newest snapshot alone (the single-snapshot path used
// by unit tests).
func buildMergedResources(floorFieldsDir, fieldsDir string, customizer CodeCustomizer, logger shared.Logger) ([]*Resource, error) {
	newest, err := buildResourcesFromDownloadedFields(fieldsDir, customizer, false, logger)
	if err != nil {
		return nil, err
	}
	if floorFieldsDir == "" {
		return newest, nil
	}
	floor, err := buildResourcesFromDownloadedFields(floorFieldsDir, customizer, false, logger)
	if err != nil {
		return nil, err
	}
	return mergeResourceSets(floor, newest, orDefaultLogger(logger)), nil
}

// mergeResourceSets unions the floor and newest resource sets keyed by struct
// name. The newest snapshot is emitted first, preserving its order and field
// shapes (newest wins); floor-only resources — present at the floor but retired
// by the newest snapshot, yet still within the supported range — are appended.
// Keeping newest's exact order makes the merge a no-op for the common case where
// the floor is a subset of the newest snapshot.
func mergeResourceSets(floor, newest []*Resource, logger shared.Logger) []*Resource {
	have := make(map[string]bool, len(newest))
	merged := make([]*Resource, 0, len(newest)+len(floor))
	for _, r := range newest {
		have[r.StructName] = true
		merged = append(merged, r)
	}
	for _, r := range floor {
		if !have[r.StructName] {
			logger.Debugf("merge: keeping floor-only resource %s (absent in newest snapshot)", r.StructName)
			merged = append(merged, r)
		}
	}
	return merged
}

func customizeBaseType(resource *Resource) {
	baseType := resource.BaseType()
	switch {
	case resource.IsSetting():
		baseType.Fields[" Key"] = NewFieldInfo("Key", "key", "string", "", "", false, false, "")

		if resource.StructName == "SettingUsg" {
			// Removed in v7, retaining for backwards compatibility
			baseType.Fields["MdnsEnabled"] = NewFieldInfo("MdnsEnabled", "mdns_enabled", "bool", "", "", false, false, "")
		}
	case resource.StructName == "Device":
		baseType.Fields[" MAC"] = NewFieldInfo("MAC", "mac", "string", createValidations(false, validation{v: mac}), "", true, false, "")
		baseType.Fields["Adopted"] = NewFieldInfo("Adopted", "adopted", "bool", "", "", false, false, "")
		baseType.Fields["Model"] = NewFieldInfo("Model", "model", "string", "", "", true, false, "")
		baseType.Fields["State"] = NewFieldInfo("State", "state", "DeviceState", "", "", false, false, "")
		baseType.Fields["Type"] = NewFieldInfo("Type", "type", "string", "", "", true, false, "")
	case resource.StructName == "User":
		baseType.Fields[" IP"] = NewFieldInfo("IP", "ip", "string", createValidations(false, validation{v: ip}), "non-generated field", true, false, "")
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
