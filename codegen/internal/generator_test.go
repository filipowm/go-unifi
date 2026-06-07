package internal //nolint:testpackage // tests access unexported symbols

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCodeFromTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		templateName  string
		template      string
		data          any
		expectedCode  string
		expectedError bool
		errorContains string
	}{
		{
			name:         "valid template",
			templateName: "simple",
			template: `package internal //nolint:testpackage // tests access unexported symbols

const greeting = "{{.Greeting}}"`,
			data:         struct{ Greeting string }{Greeting: "hello"},
			expectedCode: "const greeting = \"hello\"",
		},
		{
			name:          "invalid go code output",
			templateName:  "invalid_code",
			template:      `not valid {{ .Value }} go code`,
			data:          struct{ Value string }{Value: "test"},
			expectedError: true,
			errorContains: "failed to format source",
		},
		{
			name:         "no data",
			templateName: "nil_data",
			template:     `package main`,
			data:         nil,
			expectedCode: "package main",
		},
		{
			name:         "complex template",
			templateName: "complex",
			template: `package internal //nolint:testpackage // tests access unexported symbols

type {{.TypeName}} struct {
	{{range .Fields}}
	{{.Name}} {{.Type}}
	{{end}}
}`,
			data: struct {
				TypeName string
				Fields   []struct{ Name, Type string }
			}{
				TypeName: "Person",
				Fields: []struct{ Name, Type string }{
					{Name: "Name", Type: "string"},
					{Name: "Age", Type: "int"},
				},
			},
			expectedCode: "type Person struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			code, err := generateCodeFromTemplate(tt.templateName, tt.template, tt.data)

			if tt.expectedError {
				require.ErrorContains(t, err, tt.errorContains)
			} else {
				require.NoError(t, err)
			}
			a.Contains(code, tt.expectedCode)
		})
	}
}

// TestGenerateCode_InjectedV2BaseDir proves generateCode runs end-to-end against
// INJECTED fieldsDir and v2BaseDir fixtures, with no dependency on the real repo
// layout or findCodegenDir/cwd. Both a v1 and a v2 resource are
// emitted as <name>.generated.go alongside the client.generated.go, and the
// supplied logger receives the per-resource Debug lines.
func TestGenerateCode_InjectedV2BaseDir(t *testing.T) {
	t.Parallel()

	fieldsDir := t.TempDir()
	v2BaseDir := t.TempDir()
	outDir := t.TempDir()

	// A v1 resource and a v2 resource, each a trivial inferable field set.
	require.NoError(t, os.WriteFile(filepath.Join(fieldsDir, "Widget.json"), []byte(`{"name": ".{0,32}"}`), 0o644))  //nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(v2BaseDir, "Gadget.json"), []byte(`{"label": ".{0,32}"}`), 0o644)) //nolint:gosec

	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	err := generateCode("", fieldsDir, v2BaseDir, outDir, CodeCustomizer{}, logger)
	require.NoError(t, err)

	// Both resources and the client interface land in outDir.
	for _, want := range []string{"widget.generated.go", "gadget.generated.go", "client.generated.go"} {
		_, statErr := os.Stat(filepath.Join(outDir, want))
		require.NoErrorf(t, statErr, "expected generated file %s", want)
	}

	// The injected logger (not the package global) captured the pipeline output.
	entries := hook.AllEntries()
	debugMsgs := make([]string, 0, len(entries))
	for _, e := range entries {
		debugMsgs = append(debugMsgs, e.Message)
	}
	assert.NotEmpty(t, debugMsgs, "injected logger must receive pipeline output")
}

func TestWriteGeneratedFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		fileName         string
		content          string
		expectedFileName string
		expectError      bool
	}{
		{
			name:             "valid file",
			fileName:         "TestFile",
			content:          "package main\n\n// Code content",
			expectedFileName: "test_file.generated.go",
			expectError:      false,
		},
		{
			name:             "empty content",
			fileName:         "EmptyFile",
			content:          "",
			expectedFileName: "empty_file.generated.go",
			expectError:      true,
		},
		{
			name:             "file with spaces",
			fileName:         "Test File",
			content:          "package main",
			expectedFileName: "test_file.generated.go",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			tempDir := t.TempDir()

			fileName, err := writeGeneratedFile(tempDir, tt.fileName, tt.content)
			require.NoError(t, err)
			a.Equal(tt.expectedFileName, fileName)

			expectedFile := filepath.Join(tempDir, tt.expectedFileName)
			dataBytes, err := os.ReadFile(expectedFile)
			require.NoError(t, err)
			a.Equal(tt.content, string(dataBytes))
		})
	}
}

func TestWriteGeneratedFile_OverrideExistingFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	tempDir := t.TempDir()
	fileName := "test"

	_, err := writeGeneratedFile(tempDir, fileName, "starting content")
	require.NoError(t, err)

	_, err = writeGeneratedFile(tempDir, fileName, "updated content")
	require.NoError(t, err)

	expectedFile := filepath.Join(tempDir, "test.generated.go")
	dataBytes, err := os.ReadFile(expectedFile)
	require.NoError(t, err)
	a.Equal("updated content", string(dataBytes))
}

func TestWriteGeneratedFile_InvalidPath(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	invalidDir := filepath.Join(tempDir, "nonexistent")

	_, err := writeGeneratedFile(invalidDir, "test", "content")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to write file")
}

func TestGenerateCodeFromFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		fieldsDir      string
		outDir         string
		expectedError  bool
		errorContains  string
		setupMockFiles func(string)
	}{
		{
			name:          "invalid fields directory",
			fieldsDir:     "nonexistent",
			outDir:        t.TempDir(),
			expectedError: true,
			errorContains: "failed to build resources from downloaded fields",
		},
		{
			name:      "valid empty fields directory",
			fieldsDir: t.TempDir(),
			outDir:    t.TempDir(),
			setupMockFiles: func(dir string) {
				// Create empty directory structure
				_ = os.MkdirAll(dir, 0o755)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.setupMockFiles != nil {
				tt.setupMockFiles(tt.fieldsDir)
			}

			// Inject an empty v2 base dir (a fresh temp dir) so generation is
			// decoupled from the real repo layout and never touches
			// findCodegenDir.
			err := generateCode("", tt.fieldsDir, t.TempDir(), tt.outDir, CodeCustomizer{}, nil)

			if tt.expectedError {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
