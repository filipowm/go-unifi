package main

import (
	_ "embed"
	"fmt"
	"strings"
)

// ClientFunction is the interface for client functions.
type ClientFunction interface {
	Name() string
	IsSetting() bool
}

type FunctionParam struct {
	Name string
	Type string
}

// CustomClientFunction represents a custom client function definition.
type CustomClientFunction struct {
	Name             string
	Parameters       []FunctionParam
	ReturnParameters []string
	Comment          string
}

// Signature returns the signature string for the custom client function.
func (c *CustomClientFunction) Signature() string {
	var b strings.Builder
	if c.Comment != "" {
		b.WriteString(fmt.Sprintf("// %s %s\n", c.Name, c.Comment))
	}
	b.WriteString(c.Name)
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

// ClientInfo represents the client information used for code generation.
type ClientInfo struct {
	Imports         []string
	Functions       []ClientFunction
	CustomFunctions []CustomClientFunction
}

// newClientInfo creates ClientInfo from the provided resources.
func newClientInfo(resources []*Resource) *ClientInfo {
	functions := make([]ClientFunction, 0)
	for _, resource := range resources {
		functions = append(functions, resource)
	}
	return &ClientInfo{Functions: functions}
}

//go:embed client.go.tmpl
var clientGoTemplate string

// GenerateCode generates the code for the client using a template.
func (c *ClientInfo) GenerateCode() (string, error) {
	return generateCodeFromTemplate("client.go.tmpl", clientGoTemplate, c)
}

// Name returns the name of the client.
func (c *ClientInfo) Name() string {
	return "client"
}
