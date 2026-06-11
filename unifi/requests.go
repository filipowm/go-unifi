package unifi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

// marshalRequest marshals the request body to an io.Reader. Returns nil if reqBody is nil.
func marshalRequest(reqBody any) (io.Reader, error) {
	if reqBody == nil {
		return nil, nil //nolint: nilnil
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(reqBytes), nil
}

// buildRequestURL constructs the full URL for a given apiPath using the client's baseURL and apiPaths.
func (c *client) buildRequestURL(apiPath string) (*url.URL, error) {
	reqURL, err := url.Parse(apiPath)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(apiPath, "/") && !reqURL.IsAbs() {
		reqURL.Path = path.Join(c.apiPaths.ApiPath, reqURL.Path)
	}
	return c.baseURL.ResolveReference(reqURL), nil
}

// validateRequestBody validates the request body if validation is enabled.
func (c *client) validateRequestBody(reqBody any) error {
	if reqBody != nil && c.validationMode != DisableValidation {
		c.Trace("Validating request body")
		if err := c.validator.Validate(reqBody); err != nil {
			if c.validationMode == HardValidation {
				return fmt.Errorf("failed validating request body: %w", err)
			} else {
				c.Warnf("failed validating request body: %s", err)
			}
		}
	}
	return nil
}

// newRequestContext creates a new context for the request with a timeout if specified.
func (c *client) newRequestContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if c.timeout != 0 {
		return context.WithTimeout(ctx, c.timeout)
	}
	return ctx, func() {}
}

// applyRequestInterceptors runs the request interceptors against the given request,
// returning the first error encountered.
func (c *client) applyRequestInterceptors(req *http.Request) error {
	c.Trace("Executing request interceptors")
	for _, interceptor := range c.interceptors {
		if err := interceptor.InterceptRequest(req); err != nil {
			return err
		}
	}
	return nil
}

// overrideHeaders sets the provided headers, replacing any already set (e.g. by interceptors).
func overrideHeaders(req *http.Request, headers http.Header) {
	for key, values := range headers {
		req.Header.Del(key) // no-op if absent; replaces interceptor-set values
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

// handleResponse runs the response interceptors, checks for errors, and decodes the response
// body into respBody if one is expected. Returns an error if any step fails.
func (c *client) handleResponse(resp *http.Response, respBody any, method, apiPath string) error {
	c.Trace("Executing response interceptors")
	for _, interceptor := range c.interceptors {
		if err := interceptor.InterceptResponse(resp); err != nil {
			return err
		}
	}
	c.Trace("Checking for errors in response")
	if err := c.errorHandler.HandleError(resp); err != nil {
		return err
	}
	// If no response body is expected, this is the only unconditional skip.
	// Do NOT key the decode decision off resp.ContentLength — a server,
	// proxy, or HTTP/2 path can deliver a non-empty JSON body while reporting
	// ContentLength==0 (or -1 for chunked), which would silently leave respBody
	// zero-valued. Decide on the body itself instead.
	if respBody == nil {
		c.Trace("No response body to decode")
		return nil
	}
	return c.decodeResponseBody(resp, respBody, method, apiPath)
}

// decodeResponseBody buffers the response body once and decodes it into respBody.
// It also performs the centralized v1 meta rc:error check and the
// decode-on-body / empty-body handling. respBody is assumed non-nil.
func (c *client) decodeResponseBody(resp *http.Response, respBody any, method, apiPath string) error {
	// Buffer the body ONCE into a capped []byte so it can be both probed for a v1
	// meta envelope and decoded into respBody without re-reading the
	// network stream. The cap bounds memory against a runaway/hostile body while
	// staying generous enough for large legitimate list responses.
	//
	// Read ONE byte past the cap so an over-cap body can be detected
	// rather than silently truncated (a truncated body would otherwise surface as
	// an opaque "unable to decode body" JSON error that hides the real cause).
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxResponseBodySize)+1))
	if err != nil {
		return fmt.Errorf("unable to read body: %s %s %w", method, apiPath, err)
	}
	if len(body) > maxResponseBodySize {
		return fmt.Errorf("response body exceeded %d bytes: %s %s", maxResponseBodySize, method, apiPath)
	}

	if metaErr := metaEnvelopeError(resp, body); metaErr != nil {
		c.Trace("Response carries a meta rc:error envelope")
		return metaErr
	}

	c.Trace("Decoding response body")
	// Decode from the buffered bytes. A genuinely empty body yields io.EOF, which
	// we treat as "no content": leave respBody untouched and return nil.
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(respBody); err != nil {
		if errors.Is(err, io.EOF) {
			c.Trace("Empty response body, nothing to decode")
			return nil
		}
		return fmt.Errorf("unable to decode body: %s %s %w", method, apiPath, err)
	}
	return nil
}

// metaEnvelopeError detects a v1 API soft failure (HTTP 200 with meta.rc=="error").
// It is gated strictly on a meta block actually being present in the
// body (probe.Meta != nil): a v2-style bare body carries no meta envelope and must
// never fabricate an error. The probe uses the canonical lowercase "meta" wire tag
// regardless of respBody's own struct tags. A body that is not valid JSON yields no
// meta error here — the subsequent decode surfaces the real decode failure.
//
// A soft rc:error is surfaced as a *ServerError that, on its own, carries
// only the rc/msg from Meta.error(). The HTTP context (status code, request method,
// request URL) is stamped onto that *ServerError here so ServerError.Error() renders
// the same rich message the non-2xx HandleError path produces — instead of the lossy
// "Server error (0) for  : <msg>". A soft rc:error is NOT a 404, so the stamped
// StatusCode (typically 200) keeps errors.Is(err, ErrNotFound)==false.
func metaEnvelopeError(resp *http.Response, body []byte) error {
	var probe struct {
		Meta *Meta `json:"meta"`
	}
	// A body that is not valid JSON has no meta envelope to inspect; the probe
	// error is intentionally ignored because the caller's decode of respBody
	// surfaces the real decode failure. On any failure probe.Meta stays nil.
	_ = json.Unmarshal(body, &probe)
	if probe.Meta == nil {
		return nil
	}
	err := probe.Meta.error()
	if err == nil {
		return nil
	}
	// Enrich the soft-error *ServerError with the response context so it does not
	// render with a zero status/empty method+URL. resp.Request can be nil on a
	// hand-built *http.Response (e.g. unit tests), so guard it.
	var serverErr *ServerError
	if errors.As(err, &serverErr) {
		serverErr.StatusCode = resp.StatusCode
		if resp.Request != nil {
			serverErr.RequestMethod = resp.Request.Method
			if resp.Request.URL != nil {
				serverErr.RequestURL = resp.Request.URL.String()
			}
		}
	}
	return err
}

// executeRequest executes an HTTP request with the given context, method, URL, body, and headers.
// It applies interceptors, handles errors, and decodes the response body if provided.
// Returns an error if the request or response handling fails.
func (c *client) executeRequest(ctx context.Context, method, apiPath string, body io.Reader, headers http.Header, respBody any) error {
	url, err := c.buildRequestURL(apiPath)
	if err != nil {
		return fmt.Errorf("unable to create request URL: %w", err)
	}
	c.Debugf("Executing request: %s %s", method, url.String())

	req, err := http.NewRequestWithContext(ctx, method, url.String(), body)
	if err != nil {
		return fmt.Errorf("unable to create request: %s %s %w", method, apiPath, err)
	}

	// NOTE: requests are intentionally NOT serialized here. net/http.Client is
	// goroutine-safe. The former coarse per-request lock (gated on
	// ClientConfig.UseLocking) was removed in 1.11.0: it killed HTTP concurrency
	// and enabled the Version() re-entrant deadlock. ClientConfig.UseLocking is
	// now a no-op.
	if err := c.applyRequestInterceptors(req); err != nil {
		return err
	}

	// Set headers if provided overriding any coming from interceptors
	overrideHeaders(req, headers)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("unable to perform request: %s %s %w", method, apiPath, err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, respBody, method, apiPath)
}

// UploadFile uploads a file to the UniFi controller.
// It takes a context, API path, file path, field name, and additional form fields.
// The file is uploaded as multipart/form-data.
// It returns the response body and an error if the operation fails.
func (c *client) UploadFile(ctx context.Context, apiPath, filePath, fieldName string, respBody any) error {
	c.Tracef("Uploading file: %s to %s", filePath, apiPath)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open file for upload: %w", err)
	}
	defer file.Close()
	return c.UploadFileFromReader(ctx, apiPath, file, filepath.Base(filePath), fieldName, respBody)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// NOTE! This is a copy of the function from the mime/multipart package, but allows to set custom mimetype.
func createFormFile(w *multipart.Writer, mimeType, fieldname, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(fieldname), escapeQuotes(filename)))
	h.Set("Content-Type", mimeType)
	return w.CreatePart(h)
}

// maxUploadSize caps how much content buildMultipartUpload buffers from the
// caller's reader, preventing OOM on unexpectedly large uploads (512 MiB).
const maxUploadSize = 512 * 1024 * 1024 // 512 MiB

// buildMultipartUpload assembles a multipart/form-data body for a file upload from
// reader. It is the pure (no filesystem, no network) heart of UploadFileFromReader,
// extracted so the field-name defaulting, MIME detection and Content-Disposition
// quoting can be unit-tested in isolation.
//
// fieldName defaults to "file" when empty. The MIME type is detected from the
// content with mimetype.DetectReader; because DetectReader consumes the reader,
// the content is buffered once up front and the buffer is read TWICE — once to
// detect, once to copy into the form part (the documented buffer-twice
// workaround). It returns the assembled body buffer and the matching multipart
// Content-Type (writer.FormDataContentType()).
func buildMultipartUpload(reader io.Reader, filename, fieldName string) (*bytes.Buffer, string, error) {
	// Buffer content with a size cap to avoid OOM on runaway uploads.
	// Read one byte past the limit so an over-size source is detected rather than silently truncated.
	var buf bytes.Buffer
	limited := io.LimitReader(reader, maxUploadSize+1)
	n, err := io.Copy(&buf, limited)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read file content into buffer: %w", err)
	}
	if n > maxUploadSize {
		return nil, "", fmt.Errorf("upload exceeds maximum size of %d bytes", maxUploadSize)
	}
	contentReader := bytes.NewReader(buf.Bytes())

	if fieldName == "" {
		fieldName = "file"
	}

	// Detect MIME type from the first reader
	mt, err := mimetype.DetectReader(contentReader)
	if err != nil {
		return nil, "", fmt.Errorf("unable to detect file mimetype: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := createFormFile(writer, mt.String(), fieldName, filename)
	if err != nil {
		return nil, "", fmt.Errorf("unable to create form file: %w", err)
	}
	// reinit reader
	contentReader = bytes.NewReader(buf.Bytes())
	// Copy the file content to the form field from the second reader
	if _, err = io.Copy(part, contentReader); err != nil {
		return nil, "", fmt.Errorf("unable to copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("unable to close multipart writer: %w", err)
	}

	return body, writer.FormDataContentType(), nil
}

// UploadFileFromReader uploads a file to the UniFi controller from an io.Reader.
// It takes a context, API path, reader, filename, field name, and additional form fields.
// The file is uploaded as multipart/form-data.
func (c *client) UploadFileFromReader(ctx context.Context, apiPath string, reader io.Reader, filename, fieldName string, respBody any) error {
	c.Tracef("Uploading file: %s to %s", filename, apiPath)

	body, contentType, err := buildMultipartUpload(reader, filename, fieldName)
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Set("Content-Type", contentType)
	headers.Set("X-Requested-With", "XMLHttpRequest") // TODO if not provided, the response will be 404. UniFi bug?

	return c.executeRequest(ctx, http.MethodPost, apiPath, body, headers, respBody)
}

// Do performs an HTTP request using the given method, apiPath, request body, and decodes the response into respBody.
// It validates the request body, applies interceptors, and decodes the HTTP response into respBody if provided.
// It returns an error if the request or response handling fails.
func (c *client) Do(ctx context.Context, method, apiPath string, reqBody any, respBody any) error {
	c.Tracef("Performing request: %s %s", method, apiPath)

	if err := c.validateRequestBody(reqBody); err != nil {
		return err
	}

	body, err := marshalRequest(reqBody)
	if err != nil {
		return fmt.Errorf("unable to marshal request: %w", err)
	}

	headers := http.Header{}
	if reqBody != nil {
		headers.Set("Content-Type", "application/json")
	}

	return c.executeRequest(ctx, method, apiPath, body, headers, respBody)
}

// Get sends an HTTP GET request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Get(ctx context.Context, apiPath string, reqBody any, respBody any) error {
	return c.Do(ctx, http.MethodGet, apiPath, reqBody, respBody)
}

// Post sends an HTTP POST request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Post(ctx context.Context, apiPath string, reqBody any, respBody any) error {
	return c.Do(ctx, http.MethodPost, apiPath, reqBody, respBody)
}

// Put sends an HTTP PUT request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Put(ctx context.Context, apiPath string, reqBody any, respBody any) error {
	return c.Do(ctx, http.MethodPut, apiPath, reqBody, respBody)
}

// Patch sends an HTTP PATCH request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Patch(ctx context.Context, apiPath string, reqBody any, respBody any) error {
	return c.Do(ctx, http.MethodPatch, apiPath, reqBody, respBody)
}

// Delete sends an HTTP DELETE request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Delete(ctx context.Context, apiPath string, reqBody any, respBody any) error {
	return c.Do(ctx, http.MethodDelete, apiPath, reqBody, respBody)
}
