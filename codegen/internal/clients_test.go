package internal //nolint:testpackage // tests access unexported symbols

import (
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
				FunctionName: "Foo",
			},
			wantFunc: "Foo()",
		},
		{
			name: "with comment, no params, no returns",
			fn: CustomClientFunction{
				FunctionName: "Bar",
			},
			wantFunc: "Bar()",
		},
		{
			name: "with one param and one return",
			fn: CustomClientFunction{
				FunctionName:     "Baz",
				Parameters:       []FunctionParam{{"a", "int"}},
				ReturnParameters: []string{"error"},
			},
			wantFunc: "Baz(a int) error",
		},
		{
			name: "with multiple returns",
			fn: CustomClientFunction{
				FunctionName:     "Qux",
				Parameters:       []FunctionParam{{"x", "string"}},
				ReturnParameters: []string{"int", "error"},
			},
			wantFunc: "Qux(x string) (int, error)",
		},
		{
			name: "with multiple params",
			fn: CustomClientFunction{
				FunctionName:     "MultiParams",
				Parameters:       []FunctionParam{{"x", "string"}, {"y", "int"}},
				ReturnParameters: []string{},
			},
			wantFunc: "MultiParams(x string, y int)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			a.Equal(tt.wantFunc, tt.fn.Signature())
		})
	}
}

// clientFunctionNames returns the generated function names (skipping marker
// comments, whose Name() is empty) from a built ClientInfo.
func clientFunctionNames(ci *ClientInfo) []string {
	var names []string
	for _, f := range ci.Functions {
		if n := f.Name(); n != "" {
			names = append(names, n)
		}
	}
	return names
}

// clientMarkerComments returns the section marker comments (Name() empty) from a
// built ClientInfo.
func clientMarkerComments(ci *ClientInfo) []string {
	var comments []string
	for _, f := range ci.Functions {
		if f.Name() == "" {
			comments = append(comments, f.Comment())
		}
	}
	return comments
}

func TestAddResourceExcludeFunctions(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		structName   string
		exclude      []string
		wantFuncs    []string
		wantMarkers  bool
		wantExcluded []string
	}{
		"normal resource, no exclusions": {
			structName:  "Network",
			exclude:     nil,
			wantFuncs:   []string{"GetNetwork", "ListNetwork", "CreateNetwork", "UpdateNetwork", "DeleteNetwork"},
			wantMarkers: true,
		},
		"normal resource, exclude Update and Delete": {
			structName:   "Network",
			exclude:      []string{"Update", "Delete"},
			wantFuncs:    []string{"GetNetwork", "ListNetwork", "CreateNetwork"},
			wantExcluded: []string{"UpdateNetwork", "DeleteNetwork"},
			wantMarkers:  true,
		},
		"settings resource, no exclusions, has header and footer": {
			structName:  "SettingMgmt",
			exclude:     nil,
			wantFuncs:   []string{"GetSettingMgmt", "UpdateSettingMgmt"},
			wantMarkers: true,
		},
		"settings resource, exclude Update": {
			structName:   "SettingMgmt",
			exclude:      []string{"Update"},
			wantFuncs:    []string{"GetSettingMgmt"},
			wantExcluded: []string{"UpdateSettingMgmt"},
			wantMarkers:  true,
		},
		"exclude all actions emits nothing": {
			structName:   "Network",
			exclude:      []string{"Get", "List", "Create", "Update", "Delete"},
			wantFuncs:    nil,
			wantExcluded: []string{"GetNetwork", "ListNetwork", "CreateNetwork", "UpdateNetwork", "DeleteNetwork"},
			wantMarkers:  false,
		},
		"exclusion is case-sensitive": {
			structName:  "Network",
			exclude:     []string{"update", "delete"},
			wantFuncs:   []string{"GetNetwork", "ListNetwork", "CreateNetwork", "UpdateNetwork", "DeleteNetwork"},
			wantMarkers: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			ci := NewClientInfoBuilder().
				AddResource(&Resource{StructName: tt.structName}, tt.exclude).
				Build()
			names := clientFunctionNames(ci)

			for _, want := range tt.wantFuncs {
				a.Contains(names, want)
			}
			for _, notWant := range tt.wantExcluded {
				a.NotContains(names, notWant)
			}

			markers := clientMarkerComments(ci)
			if tt.wantMarkers {
				a.Len(markers, 2, "expected both header and footer markers")
			} else {
				a.Empty(markers, "expected no marker comments when fully excluded")
				a.Empty(names, "expected no functions when fully excluded")
			}
		})
	}
}

func TestGenerateCode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	b := NewClientInfoBuilder()
	b.AddImport("fmt")
	b.AddFunction(&CustomClientFunction{
		FunctionName:     "TestFunc",
		Parameters:       []FunctionParam{{"x", "int"}},
		ReturnParameters: []string{"error"},
		FunctionComment:  "This is a test function",
	})
	ci := b.Build()
	code, err := ci.GenerateCode()
	require.NoError(t, err)
	a.NotEmpty(code, "GenerateCode() returned empty code")
	a.Contains(code, "TestFunc")
	// Ensure the hardcoded official-API import survives template relocation.
	a.Contains(code, "github.com/filipowm/go-unifi/v2/unifi/official")
}
