package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCodeFromTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		templateName  string
		template      string
		data          interface{}
		expectedCode  string
		expectedError bool
		errorContains string
	}{
		{
			name:         "valid template",
			templateName: "simple",
			template: `package main

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
