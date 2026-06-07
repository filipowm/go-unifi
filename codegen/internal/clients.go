package internal

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
)

// ClientFunction is the interface for client functions.
type ClientFunction interface {
	Name() string
	ResourceName() string
	Comment() string
	Signature() string
}

type FunctionParam struct {
	Name string
	Type string
}

type Comment struct {
	comment      string
	resourceName string
}

func (c *Comment) Name() string {
	return ""
}

func (c *Comment) Comment() string {
	return c.comment
}

func (c *Comment) Signature() string {
	return ""
}

func (c *Comment) ResourceName() string {
	return c.resourceName
}

// CustomClientFunction represents a custom client function definition.
type CustomClientFunction struct {
	Resource         string          `yaml:"resourceName"`
	FunctionName     string          `yaml:"name"`
	Parameters       []FunctionParam `yaml:"params"`
	ReturnParameters []string        `yaml:"returns"`
	FunctionComment  string          `yaml:"comment"`
}

func (c *CustomClientFunction) Name() string {
	return c.FunctionName
}

func (c *CustomClientFunction) ResourceName() string {
	return c.Resource
}

// Signature returns the signature string for the custom client function.
func (c *CustomClientFunction) Signature() string {
	if c.Name() == "" {
		return ""
	}
	var b strings.Builder
	// if c.comment != "" {
	//	 b.WriteString(fmt.Sprintf("// %s %s\n", c.Name, c.Comment))
	// }
	b.WriteString(c.Name())
	b.WriteString("(")

	// Build parameters without trailing comma
	params := make([]string, 0, len(c.Parameters))
	for _, v := range c.Parameters {
		params = append(params, fmt.Sprintf("%s %s", v.Name, v.Type))
	}
	b.WriteString(strings.Join(params, ", "))
	b.WriteString(")")

	if len(c.ReturnParameters) > 1 {
		b.WriteString(" (")
		b.WriteString(strings.Join(c.ReturnParameters, ", "))
		b.WriteString(")")
	} else if len(c.ReturnParameters) == 1 {
		b.WriteString(" " + c.ReturnParameters[0])
	}
	return b.String()
}

func (c *CustomClientFunction) Comment() string {
	return c.FunctionComment
}

// newClientInfo creates ClientInfo from the provided resources.
func newClientInfo(imports []string, functions []ClientFunction) *ClientInfo {
	return &ClientInfo{imports, functions}
}

// ClientInfo represents the client information used for code generation.
type ClientInfo struct {
	Imports   []string
	Functions []ClientFunction
}

// Name returns the name of the client.
func (c *ClientInfo) Name() string {
	return "Client"
}

// ResourceFunctions returns the resource-bound functions (standard CRUD plus
// custom resource methods), which make up the embedded InternalClient interface.
// They are distinguished from transport/lifecycle functions by carrying a
// resource name.
func (c *ClientInfo) ResourceFunctions() []ClientFunction {
	return filterFunctions(c.Functions, func(f ClientFunction) bool { return f.ResourceName() != "" })
}

// TransportFunctions returns the transport/lifecycle functions (Do/Get/Post/Put/
// Delete, Login/Logout, Version, BaseURL, ...), which sit on the top-level Client
// interface alongside the Internal()/Official() accessors. They are the functions
// with no resource name.
func (c *ClientInfo) TransportFunctions() []ClientFunction {
	return filterFunctions(c.Functions, func(f ClientFunction) bool { return f.ResourceName() == "" })
}

// filterFunctions returns the subset of fns satisfying keep, preserving order.
func filterFunctions(fns []ClientFunction, keep func(ClientFunction) bool) []ClientFunction {
	out := make([]ClientFunction, 0, len(fns))
	for _, f := range fns {
		if keep(f) {
			out = append(out, f)
		}
	}
	return out
}

//go:embed client.go.tmpl
var clientGoTemplate string

// GenerateCode generates the code for the client using a template.
func (c *ClientInfo) GenerateCode() (string, error) {
	return generateCodeFromTemplate("client.go.tmpl", clientGoTemplate, c)
}

type ClientInfoBuilder struct {
	imports   []string
	functions []ClientFunction
}

func NewClientInfoBuilder() *ClientInfoBuilder {
	return &ClientInfoBuilder{}
}

func (c *ClientInfoBuilder) AddFunction(f ClientFunction) *ClientInfoBuilder {
	c.functions = append(c.functions, f)
	return c
}

func (c *ClientInfoBuilder) AddFunctions(f []CustomClientFunction) *ClientInfoBuilder {
	for _, v := range f {
		c.functions = append(c.functions, &v)
	}
	return c
}

// resourceAction describes a single standard client CRUD action.
type resourceAction struct {
	name    string
	comment string
	params  []FunctionParam
	returns []string
}

// standardActions returns the standard client CRUD actions for r, in generation
// order. Settings expose only Get (singleton getter) + Update; other resources
// expose the full Get/List/Create/Update/Delete set. This is the single source of
// truth for the action catalog — both code generation (AddResource) and
// customization validation (standardActionNames) derive from it.
func standardActions(r *Resource) []resourceAction {
	if r.IsSetting() {
		return []resourceAction{
			{"Get", "retrieves the settings for a resource", nil, singlePointerReturn(r.Name())},
			{"Update", "updates a resource", singlePointerParam(r.Name()), singlePointerReturn(r.Name())},
		}
	}
	return []resourceAction{
		{"Get", "retrieves a resource", []FunctionParam{{"id", "string"}}, singlePointerReturn(r.Name())},
		{"List", "lists the resources", nil, []string{"[]" + r.Name()}},
		{"Create", "creates a resource", singlePointerParam(r.Name()), singlePointerReturn(r.Name())},
		{"Update", "updates a resource", singlePointerParam(r.Name()), singlePointerReturn(r.Name())},
		{"Delete", "deletes a resource", []FunctionParam{{"id", "string"}}, nil},
	}
}

// standardActionNames is the set of valid action names for r, derived from the
// same table as standardActions so the catalog never drifts.
func standardActionNames(r *Resource) map[string]bool {
	actions := standardActions(r)
	names := make(map[string]bool, len(actions))
	for _, a := range actions {
		names[a.name] = true
	}
	return names
}

// AddResource adds the standard client CRUD methods for r, omitting any whose
// action name appears in excludeFunctions. When every action is excluded, nothing
// is emitted (not even the section marker comments).
func (c *ClientInfoBuilder) AddResource(r *Resource, excludeFunctions []string) *ClientInfoBuilder {
	excluded := make(map[string]bool, len(excludeFunctions))
	for _, a := range excludeFunctions {
		excluded[a] = true
	}
	included := make([]resourceAction, 0, 5)
	for _, a := range standardActions(r) {
		if !excluded[a.name] {
			included = append(included, a)
		}
	}
	if len(included) == 0 {
		return c
	}
	c.AddFunction(&Comment{comment: fmt.Sprintf("==== client methods for %s resource ====", r.Name()), resourceName: r.Name()})
	for _, a := range included {
		c.addResourceFunction(a.name, r.Name(), a.comment, a.params, a.returns)
	}
	c.AddFunction(&Comment{comment: fmt.Sprintf("==== end of client methods for %s resource ====", r.Name()), resourceName: r.Name() + "_end"})
	return c
}

func (c *ClientInfoBuilder) AddImport(i string) *ClientInfoBuilder {
	c.imports = append(c.imports, i)
	return c
}

func (c *ClientInfoBuilder) AddImports(i []string) *ClientInfoBuilder {
	c.imports = append(c.imports, i...)
	return c
}

func (c *ClientInfoBuilder) Build() *ClientInfo {
	// Sort the functions by resource name and then by name.
	sort.Slice(c.functions, func(i, j int) bool {
		if c.functions[i].ResourceName() == c.functions[j].ResourceName() {
			return c.functions[i].Signature() < c.functions[j].Signature()
		}
		return c.functions[i].ResourceName() < c.functions[j].ResourceName()
	})

	return newClientInfo(c.imports, c.functions)
}

func (c *ClientInfoBuilder) addResourceFunction(actionName, resourceName, comment string, additionalParams []FunctionParam, additionalReturns []string) {
	fName := fmt.Sprintf("%s%s", actionName, resourceName)

	params := make([]FunctionParam, 0, 2+len(additionalParams))
	params = append(params, FunctionParam{"ctx", "context.Context"}, FunctionParam{"site", "string"})
	params = append(params, additionalParams...)
	returns := additionalReturns
	returns = append(returns, "error")
	f := CustomClientFunction{
		FunctionName:     fName,
		Resource:         resourceName,
		Parameters:       params,
		ReturnParameters: returns,
		FunctionComment:  fmt.Sprintf("%s %s", fName, comment),
	}
	c.AddFunction(&f)
}

func singlePointerReturn(name string) []string {
	return []string{"*" + name}
}

func singlePointerParam(name string) []FunctionParam {
	return []FunctionParam{{strings.ToLower(name[0:1]), "*" + name}}
}
