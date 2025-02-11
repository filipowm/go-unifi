package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomClientFunctionSignature(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		fn          CustomClientFunction
		wantComment string // expected comment in the signature
		wantFunc    string // expected function signature
	}{
		{
			name: "no comment, no params, no returns",
			fn: CustomClientFunction{
				Name: "Foo",
			},
			wantFunc: "Foo()",
		},
		{
			name: "with comment, no params, no returns",
			fn: CustomClientFunction{
				Name:    "Bar",
				Comment: "does something",
			},
			wantComment: "// Bar does something",
			wantFunc:    "Bar()",
		},
		{
			name: "with one param and one return",
			fn: CustomClientFunction{
				Name:             "Baz",
				Parameters:       []FunctionParam{{"a", "int"}},
				ReturnParameters: []string{"error"},
			},
			wantFunc: "Baz(a int) error",
		},
		{
			name: "with multiple returns",
			fn: CustomClientFunction{
				Name:             "Qux",
				Parameters:       []FunctionParam{{"x", "string"}},
				ReturnParameters: []string{"int", "error"},
			},
			wantFunc: "Qux(x string) (int, error)",
		},
		{
			name: "with multiple params",
			fn: CustomClientFunction{
				Name:             "MultiParams",
				Parameters:       []FunctionParam{{"x", "string"}, {"y", "int"}},
				ReturnParameters: []string{},
				Comment:          "function with multiple parameters",
			},
			wantComment: "// MultiParams function with multiple parameters",
			wantFunc:    "MultiParams(x string, y int)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			got := tt.fn.Signature()

			parts := strings.Split(got, "\n")
			var comment, funcSig string
			if len(tt.wantComment) > 0 {
				comment = parts[0]
				funcSig = parts[1]
			} else {
				funcSig = parts[0]
			}
			a.Equal(tt.wantComment, comment)
			a.Equal(tt.wantFunc, funcSig)
		})
	}
}

func TestGenerateCode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	ci := &ClientInfo{
		Imports: []string{"fmt"},
		CustomFunctions: []CustomClientFunction{
			{
				Name:             "TestFunc",
				Parameters:       []FunctionParam{{"x", "int"}},
				ReturnParameters: []string{"error"},
				Comment:          "This is a test function",
			},
		},
	}
	code, err := ci.GenerateCode()
	require.NoError(t, err)
	a.NotEmpty(code, "GenerateCode() returned empty code")
	a.Contains(code, "TestFunc")
}
