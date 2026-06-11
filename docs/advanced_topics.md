# Advanced Topics

This document delves into advanced aspects of using the UniFi Go SDK client, explaining how to customize the HTTP client,
use interceptors effectively, handle errors robustly, and extend validations.

## Making a raw API Call using SDK Methods

For endpoints that are not directly covered by a specialized client method, the UniFi Go SDK provides a set of helper methods for making requests to UniFi API. These methods simplify API interactions
by handling common tasks such as request construction, JSON marshaling of the request body, authentication, applying interceptors, error handling, and decoding the response:

- **Do**: The core method that performs an HTTP request with a given method, API path, request body, and destination for decoding the response. It handles validation, URL construction, interceptors,
  and error processing.
- **Get**: A convenience wrapper around **Do** that executes an HTTP GET request.
- **Post**: A convenience wrapper to perform an HTTP POST request.
- **Put**: Similar to Post, but for HTTP PUT requests.
- **Patch**: A convenience wrapper to perform an HTTP PATCH request (partial update).
- **Delete**: Performs an HTTP DELETE request.

These methods are used internally by higher level functions, such as those in `unifi/device.generated.go` and `unifi/device.go`. For example, when creating a new device, the SDK calls `Post` to send
the device data to the UniFi Controller API, while `Get` is used to retrieve device information.

### Path resolution

The `apiPath` argument follows a simple rule (see `unifi/requests.go` `buildRequestURL`):

- **No leading slash (site-relative):** the path is prefixed with the controller's base API prefix. On
  new-style (UniFi OS) controllers — the **only** style supported in 2.0.0 — that prefix is
  `/proxy/network/api`. Example: `"s/default/rest/networkconf"` → `/proxy/network/api/s/default/rest/networkconf`.
- **Leading slash or absolute URL:** used **as-is**, bypassing the prefix entirely. Use this only when
  you deliberately need a path outside the standard API tree (e.g. `/proxy/network/integration/v1/...`).
  The old `/api/...` form that worked on classic controllers **does not work** on new-style controllers;
  use the site-relative form instead.

**v2 API tree.** Some newer resources (firewall zones, zone policies, AP groups, etc.) live under the
v2 API tree at `/proxy/network/v2/api`. The exported `unifi.NewStyleAPI.ApiV2Path` constant gives you
that base path so you don't need to hard-code it:

```go
// Access a resource under the v2 API tree
var out struct{ /* ... */ }
err = c.Get(ctx, unifi.NewStyleAPI.ApiV2Path+"/site/default/firewall/zone", nil, &out)
if err != nil {
    log.Fatalf("Error: %v", err)
}
```

### Examples

Here is an example of using these methods for a custom API operation:

```go
// Define a custom response structure
var respData struct {
    Meta unifi.Meta  `json:"meta"`
    Data interface{} `json:"data"`
}

// Use the Get method to fetch data from a custom endpoint (site-relative path)
err := c.Get(ctx, "s/default/rest/networkconf", nil, &respData)
if err != nil {
    log.Fatalf("Error performing GET request: %v", err)
}

// For a POST request, define your request payload and response structure:
reqPayload := struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}{
    Field1: "value",
    Field2: 123,
}

var postResp struct {
    Meta unifi.Meta  `json:"meta"`
    Data interface{} `json:"data"`
}

err = c.Post(ctx, "s/default/rest/networkconf", reqPayload, &postResp)
if err != nil {
    log.Fatalf("Error performing POST request: %v", err)
}
// do something with the response
```

For a partial update (PATCH), pass only the fields you want to change. `Patch` sends a literal HTTP PATCH,
so the target endpoint must accept it — the Official `integration/v1` surface does; most legacy Internal
REST resources (e.g. `networkconf`) expect `PUT`, so use `Put` for those:

```go
patch := struct {
    Name string `json:"name"`
}{Name: "updated-name"}

var patchResp struct {
    Meta unifi.Meta  `json:"meta"`
    Data interface{} `json:"data"`
}

// Absolute path (leading slash) ⇒ sent as-is; PATCH-capable endpoint
err = c.Patch(ctx, "/proxy/network/integration/v1/sites/<id>/...", patch, &patchResp)
if err != nil {
    log.Fatalf("Error performing PATCH request: %v", err)
}
```

These helper methods abstract away the boilerplate of manually constructing HTTP requests and processing responses, allowing you to focus on your application's logic while leveraging built-in
validation and error handling provided by the SDK.

## Interceptors and Middleware

Interceptors provide hooks into the request/response cycle and can be used for logging, metrics collection, or modifying
requests before they are sent. They implement the [ClientInterceptor](https://pkg.go.dev/github.com/filipowm/go-unifi/v2/unifi#ClientInterceptor) interface.

### Example: Timing transport via HttpRoundTripperProvider

For timing and tracing, use `ClientConfig.HttpRoundTripperProvider` which wraps the transport and can
properly observe request/response without consuming the response body:

```go
import (
    "log"
    "net/http"
    "time"

    "github.com/filipowm/go-unifi/v2/unifi"
)

// timingTransport wraps an http.RoundTripper and logs request duration.
type timingTransport struct {
    wrapped http.RoundTripper
}

func (t timingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    start := time.Now()
    resp, err := t.wrapped.RoundTrip(req)
    log.Printf("request to %s took %s", req.URL.Path, time.Since(start))
    return resp, err
}

c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key",
    HttpRoundTripperProvider: func() http.RoundTripper {
        return timingTransport{wrapped: http.DefaultTransport}
    },
})
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}
```

### Example: Simple logging interceptor

```go
// LoggingInterceptor logs request method and URL.
type LoggingInterceptor struct{}

func (l *LoggingInterceptor) InterceptRequest(req *http.Request) error {
    log.Printf("[Request] %s %s", req.Method, req.URL)
    return nil
}

func (l *LoggingInterceptor) InterceptResponse(resp *http.Response) error {
    log.Printf("[Response] %d %s", resp.StatusCode, resp.Request.URL)
    return nil
}

c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:          "https://unifi.localdomain",
    APIKey:       "your-api-key",
    Interceptors: []unifi.ClientInterceptor{&LoggingInterceptor{}},
})
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}
```

### Interceptor ordering and response-body hazard

Interceptors run in this order on every request:

1. Built-in API-key auth interceptor.
2. Built-in default-headers interceptor.
3. User-supplied interceptors, in registration order (via `ClientConfig.Interceptors` or `AddInterceptor`).

Response interceptors (`InterceptResponse`) run **before** error handling and response decoding.

**Do not read or consume `resp.Body` inside `InterceptResponse`.** The body is decoded _after_ all
interceptors run; if an interceptor drains the body, the caller receives a zero-valued response struct
with no error — a silent, hard-to-debug failure.

If you need to observe or modify the body (e.g. for logging, tracing, or body rewriting), use
`ClientConfig.HttpRoundTripperProvider` instead. A `http.RoundTripper` wrapper can buffer and restore
the body safely, before the SDK touches it:

```go
type bodyLoggingTransport struct{ wrapped http.RoundTripper }

func (t bodyLoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    resp, err := t.wrapped.RoundTrip(req)
    if err != nil || resp == nil {
        return resp, err
    }
    body, _ := io.ReadAll(resp.Body)
    resp.Body.Close()
    log.Printf("response body: %s", body)
    resp.Body = io.NopCloser(bytes.NewReader(body)) // restore for the SDK to decode
    return resp, nil
}
```

## Debugging and Logging

The SDK provides flexible logging capabilities through the `Logger` interface. You can either use the default logger or implement your own custom logger.

### Using the Default Logger

The SDK includes a default logger based on [logrus](https://github.com/sirupsen/logrus). You can configure it with different logging levels:

```go
// Configure client with default logger at Debug level
config := &unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key",
    Logger: unifi.NewDefaultLogger(unifi.DebugLevel),
}
client, err := unifi.NewClient(config)
```

Available logging levels are:
- `unifi.DisabledLevel` - no logging
- `unifi.TraceLevel` - most verbose level
- `unifi.DebugLevel` - debug information
- `unifi.InfoLevel` - default level, informational messages
- `unifi.WarnLevel` - warning messages
- `unifi.ErrorLevel` - error messages only

The `Logger` interface is embedded directly in `Client`, so logging methods are promoted and callable
on the client itself:

```go
client.Trace("Trace message")
client.Tracef("Trace message with %s", "formatting")
client.Debug("Debug message")
client.Debugf("Debug message with %s", "formatting")
client.Info("Info message")
client.Infof("Info message with %s", "formatting")
client.Warn("Warn message")
client.Warnf("Warn message with %s", "formatting")
client.Error("Error message")
client.Errorf("Error message with %s", "formatting")
```

### Custom Logger Implementation

You can implement your own logger by implementing the `Logger` interface:

```go
type MyCustomLogger struct {
    // your logger fields
}

// Implement all required methods
func (l *MyCustomLogger) Trace(msg string)                              { /* implementation */ }
func (l *MyCustomLogger) Debug(msg string)                              { /* implementation */ }
func (l *MyCustomLogger) Info(msg string)                              { /* implementation */ }
func (l *MyCustomLogger) Error(msg string)                             { /* implementation */ }
func (l *MyCustomLogger) Warn(msg string)                              { /* implementation */ }
func (l *MyCustomLogger) Tracef(format string, args ...interface{})    { /* implementation */ }
func (l *MyCustomLogger) Debugf(format string, args ...interface{})    { /* implementation */ }
func (l *MyCustomLogger) Infof(format string, args ...interface{})     { /* implementation */ }
func (l *MyCustomLogger) Errorf(format string, args ...interface{})    { /* implementation */ }
func (l *MyCustomLogger) Warnf(format string, args ...interface{})     { /* implementation */ }

// Use custom logger in client configuration
config := &unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key",
    Logger: &MyCustomLogger{},
}
client, err := unifi.NewClient(config)
```

If no logger is specified in the configuration, the SDK will use the default logger with `Info` level.

## Advanced Error Handling

The client supports both soft and hard validation modes. When using hard validation, errors returned are of type
`unifi.ValidationError` containing details about which fields failed validation.

Example error handling snippet:

```go
n := &unifi.Network{
    Name:     "my-network",
    Purpose:  "invalid-purpose",
    IPSubnet: "10.0.0.10/24",
}

_, err = c.CreateNetwork(ctx, "default", n)
if err != nil {
    var validationErr *unifi.ValidationError
    if errors.As(err, &validationErr) {
        // Process detailed validation errors
        for field, errMsg := range validationErr.Messages {
            log.Printf("Validation error on %s: %s", field, errMsg)
        }
    } else {
        log.Fatalf("Error creating network: %v", err)
    }
}
```

## Extending Validations

If the default validations do not meet your needs, you can implement custom validation logic. Extend the SDK's validation rules by wrapping or augmenting the existing ones. For example, 
you can create a custom validator function and integrate it into your client initialization. Check [validation.go](../unifi/validation.go) for details.

## Contributing and Extending the SDK

The UniFi Go SDK is designed to be adaptable:

- **Feature Requests:** If the SDK does not support a particular API endpoint, consider contributing by opening an issue or a pull request.
- **Custom Extensions:** You can fork the SDK and add custom methods or enhancements that fit your application needs. But I would greatly appreciate if you could contribute them back to the main repository.
- **Community Support:** Join our community discussions to share improvements and ask for guidance on advanced topics.

For more details on contributing, see the [Contributing Guidelines](https://github.com/filipowm/go-unifi/blob/main/CONTRIBUTING.md).

---

This document is intended for advanced users who need deeper control and customization over the UniFi client.
For most users, the basic configuration and usage examples should suffice.
