package unifi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// marshalRequest marshals the request body to an io.Reader. Returns nil if reqBody is nil.
func marshalRequest(reqBody interface{}) (io.Reader, error) {
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
func (c *client) validateRequestBody(reqBody interface{}) error {
	if reqBody != nil && c.validationMode != DisableValidation {
		if err := c.validator.Validate(reqBody); err != nil {
			err = fmt.Errorf("failed validating request body: %w", err)
			if c.validationMode == HardValidation {
				return err
			} else {
				fmt.Println(err)
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

// Do performs an HTTP request using the given method, apiPath, request body, and decodes the response into respBody.
// It validates the request body, applies interceptors, and decodes the HTTP response into respBody if provided.
// It returns an error if the request or response handling fails.
func (c *client) Do(ctx context.Context, method, apiPath string, reqBody interface{}, respBody interface{}) error {
	if err := c.validateRequestBody(reqBody); err != nil {
		return err
	}
	reqReader, err := marshalRequest(reqBody)
	if err != nil {
		return fmt.Errorf("unable to marshal request: %w", err)
	}

	url, err := c.buildRequestURL(apiPath)
	if err != nil {
		return fmt.Errorf("unable to create request URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url.String(), reqReader)
	if err != nil {
		return fmt.Errorf("unable to create request: %s %s %w", method, apiPath, err)
	}

	if c.useLocking {
		c.lock.Lock()
		defer c.lock.Unlock()
	}
	for _, interceptor := range c.interceptors {
		if err := interceptor.InterceptRequest(req); err != nil {
			return err
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("unable to perform request: %s %s %w", method, apiPath, err)
	}
	defer resp.Body.Close()

	for _, interceptor := range c.interceptors {
		if err := interceptor.InterceptResponse(resp); err != nil {
			return err
		}
	}

	if err := c.errorHandler.HandleError(resp); err != nil {
		return err
	}

	if respBody == nil || resp.ContentLength == 0 {
		return nil
	}

	err = json.NewDecoder(resp.Body).Decode(respBody)
	if err != nil {
		return fmt.Errorf("unable to decode body: %s %s %w", method, apiPath, err)
	}
	return nil
}

// Get sends an HTTP GET request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Get(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(ctx, http.MethodGet, apiPath, reqBody, respBody)
}

// Post sends an HTTP POST request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Post(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(ctx, http.MethodPost, apiPath, reqBody, respBody)
}

// Put sends an HTTP PUT request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Put(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(ctx, http.MethodPut, apiPath, reqBody, respBody)
}

// Delete sends an HTTP DELETE request to the specified API path with the provided request body,
// and decodes the HTTP response into respBody.
// It is a convenience wrapper around Do.
func (c *client) Delete(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error {
	return c.Do(ctx, http.MethodDelete, apiPath, reqBody, respBody)
}
