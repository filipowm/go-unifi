package unifi //nolint: testpackage

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEscapeQuotes is a direct unit test of the Content-Disposition quote escaper:
// backslashes and double quotes must be backslash-escaped so a crafted
// filename/field name cannot break out of the header value.
func TestEscapeQuotes(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		in   string
		want string
	}{
		"plain":              {in: "file", want: "file"},
		"double quote":       {in: `a"b`, want: `a\"b`},
		"backslash":          {in: `a\b`, want: `a\\b`},
		"both":               {in: `a"\b`, want: `a\"\\b`},
		"leading quote":      {in: `"x`, want: `\"x`},
		"empty":              {in: "", want: ""},
		"only backslashes":   {in: `\\`, want: `\\\\`},
		"name with filename": {in: `my "report".pdf`, want: `my \"report\".pdf`},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, escapeQuotes(tc.in))
		})
	}
}

// TestBuildMultipartUploadFieldDefaulting verifies that an empty fieldName defaults
// to "file" while an explicit field name is preserved. It parses the
// produced multipart body back to assert on the actual part name.
func TestBuildMultipartUploadFieldDefaulting(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		fieldName string
		wantField string
	}{
		"empty defaults to file":  {fieldName: "", wantField: "file"},
		"explicit field retained": {fieldName: "image", wantField: "image"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body, contentType, err := buildMultipartUpload(strings.NewReader("hello content"), "upload.txt", tc.fieldName)
			require.NoError(t, err)
			require.NotNil(t, body)

			part := parseSingleMultipartPart(t, body.Bytes(), contentType)
			assert.Equal(t, tc.wantField, part.FormName())
			assert.Equal(t, "upload.txt", part.FileName())
		})
	}
}

// TestBuildMultipartUploadDetectsContentType verifies that the per-part
// Content-Type is the MIME type detected from the content itself, not assumed from
// the filename. PNG magic bytes -> image/png; plain text -> text/plain.
func TestBuildMultipartUploadDetectsContentType(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		content     []byte
		wantContent string
	}{
		"png magic bytes": {
			content:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0},
			wantContent: "image/png",
		},
		"plain text": {
			content:     []byte("just some plain ascii text"),
			wantContent: "text/plain; charset=utf-8",
		},
		"json body": {
			content:     []byte(`{"key":"value"}`),
			wantContent: "application/json",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body, contentType, err := buildMultipartUpload(bytes.NewReader(tc.content), "fixture.bin", "file")
			require.NoError(t, err)

			part := parseSingleMultipartPart(t, body.Bytes(), contentType)
			assert.Equal(t, tc.wantContent, part.Header.Get("Content-Type"))

			// The content must round-trip byte-for-byte through the multipart body.
			got, err := io.ReadAll(part)
			require.NoError(t, err)
			assert.Equal(t, tc.content, got)
		})
	}
}

// TestBuildMultipartUploadReadError verifies that a reader error is wrapped with
// the "unable to read file content into buffer" context.
func TestBuildMultipartUploadReadError(t *testing.T) {
	t.Parallel()

	_, _, err := buildMultipartUpload(&errReader{}, "x", "file")
	require.ErrorContains(t, err, "unable to read file content into buffer")
}

// TestBuildMultipartUploadSizeLimit verifies that a reader exceeding maxUploadSize
// returns an explicit "exceeds maximum size" error rather than silently buffering
// an arbitrarily large payload.
func TestBuildMultipartUploadSizeLimit(t *testing.T) {
	t.Parallel()

	// Synthesize a reader that reports more bytes than the limit without actually
	// allocating maxUploadSize bytes: LimitReader(zeros, limit+2) yields limit+2
	// zero bytes cheaply via io.LimitReader over a zero-byte source.
	overSize := io.LimitReader(zeroReader{}, maxUploadSize+2)
	_, _, err := buildMultipartUpload(overSize, "big.bin", "file")
	require.ErrorContains(t, err, "exceeds maximum size")
}

// errReader always fails on Read, simulating an io.Reader that errors mid-stream.
type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

// zeroReader is an infinite source of zero bytes, used to simulate large uploads cheaply.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// parseSingleMultipartPart parses a multipart body (as produced by
// buildMultipartUpload) and returns its single part, fully buffered so the caller
// can read it after the reader has advanced.
func parseSingleMultipartPart(t *testing.T, body []byte, contentType string) *multipart.Part {
	t.Helper()
	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	boundary, ok := params["boundary"]
	require.True(t, ok, "multipart Content-Type must carry a boundary")

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	part, err := reader.NextPart()
	require.NoError(t, err)
	return part
}

// TestUploadFileFromReaderRoundTrip drives the full UploadFileFromReader path
// against a mock controller and asserts on what actually reaches the wire:
// the part name defaults to "file", the detected Content-Type, the file
// bytes round-trip, and the mandatory X-Requested-With: XMLHttpRequest header is
// present (dropping it makes the controller 404 — a documented UniFi bug).
func TestUploadFileFromReaderRoundTrip(t *testing.T) {
	t.Parallel()

	const fileContent = "uploaded file body contents"

	// The handler only records request metadata (no assertions in the server
	// goroutine — testifylint go-require). The raw multipart body is captured and
	// parsed back in the main goroutine where require is safe.
	var xRequestedWith, contentType string
	var rawBody []byte
	cs := newControllerServer(t, route{
		path: apiV1Path("upload"),
		fn: func(w http.ResponseWriter, r *http.Request) {
			xRequestedWith = r.Header.Get("X-Requested-With")
			contentType = r.Header.Get("Content-Type")
			rawBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
		},
	})
	c := cs.client()

	err := c.UploadFileFromReader(context.Background(), "upload", strings.NewReader(fileContent), "report.txt", "", nil)
	require.NoError(t, err)

	assert.Equal(t, "XMLHttpRequest", xRequestedWith, "X-Requested-With header must be present (UniFi 404 workaround)")

	part := parseSingleMultipartPart(t, rawBody, contentType)
	assert.Equal(t, "file", part.FormName(), "empty field name must default to 'file'")
	assert.Equal(t, "report.txt", part.FileName())
	assert.Equal(t, "text/plain; charset=utf-8", part.Header.Get("Content-Type"), "detected MIME type must be sent")
	gotBody, err := io.ReadAll(part)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(gotBody), "file bytes must round-trip")
}

// TestUploadFileUsesBaseName verifies that UploadFile derives the multipart
// filename from filepath.Base of the supplied path (not the full path), and that
// the file content reaches the controller.
func TestUploadFileUsesBaseName(t *testing.T) {
	t.Parallel()

	const fileContent = "disk file contents"
	dir := t.TempDir()
	fullPath := filepath.Join(dir, "nested", "artifact.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(fileContent), 0o600))

	// The handler only records the raw body + Content-Type; the multipart parse and
	// all assertions happen in the main goroutine (testifylint go-require).
	var contentType string
	var rawBody []byte
	cs := newControllerServer(t, route{
		path: apiV1Path("upload"),
		fn: func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			rawBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
		},
	})
	c := cs.client()

	err := c.UploadFile(context.Background(), "upload", fullPath, "file", nil)
	require.NoError(t, err)

	part := parseSingleMultipartPart(t, rawBody, contentType)
	assert.Equal(t, "artifact.txt", part.FileName(), "filename must be filepath.Base of the path, not the full path")
	gotBody, err := io.ReadAll(part)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(gotBody))
}

// TestUploadFileOpenError verifies that a nonexistent path surfaces the wrapped
// "unable to open file for upload" error.
func TestUploadFileOpenError(t *testing.T) {
	t.Parallel()

	c := newRequestHelperClient()
	missing := filepath.Join(t.TempDir(), "does-not-exist.txt")
	err := c.UploadFile(context.Background(), "upload", missing, "file", nil)
	require.ErrorContains(t, err, "unable to open file for upload")
}
