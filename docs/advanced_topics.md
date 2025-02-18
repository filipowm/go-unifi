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
- **Delete**: Performs an HTTP DELETE request.

These methods are used internally by higher level functions, such as those in `unifi/device.generated.go` and `unifi/device.go`. For example, when creating a new device, the SDK calls `Post` to send
the device data to the UniFi Controller API, while `Get` is used to retrieve device information.

Here is an example of using these methods for a custom API operation:

```go
// Define a custom response structure
var respData struct {
    Meta Meta        `json:"meta"`
    Data interface{} `json:"data"`
}

// Use the Get method to fetch data from a custom endpoint
err := c.Get(ctx, "/api/customEndpoint", nil, &respData)
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
    Meta Meta        `json:"meta"`
    Data interface{} `json:"data"`
}

err = c.Post(ctx, "/api/customPostEndpoint", reqPayload, &postResp)
if err != nil {
    log.Fatalf("Error performing POST request: %v", err)
}
// do something with the response
```

These helper methods abstract away the boilerplate of manually constructing HTTP requests and processing responses, allowing you to focus on your application's logic while leveraging built-in
validation and error handling provided by the SDK.

## Customizing the HTTP Client

While the basic configuration allows simple modifications, you can fully customize the underlying HTTP client for more
control over connection settings, proxy configuration, TLS settings, and connection pooling.

Example:

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    BaseURL: "https://unifi.localdomain",
    APIKey: "your-api-key",
    HttpCustomizer: func (transport *http.Transport) error {
        transport.MaxIdleConns = 20
        transport.IdleConnTimeout = 90 * time.Second
        // Customize TLS settings or add a proxy configuration
        return nil
    },
})
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}
```

## Interceptors and Middleware

Interceptors provide hooks into the request/response cycle and can be used for logging, metrics collection, or modifying
requests before they are sent. They implement the [ClientInterceptor](https://pkg.go.dev/github.com/filipowm/go-unifi/unifi#ClientInterceptor) interface.

### Example: Advanced Logging Interceptor

```go
// AdvancedLoggingInterceptor logs HTTP details and measures request time
type AdvancedLoggingInterceptor struct {}

func (a *AdvancedLoggingInterceptor) InterceptRequest(req *http.Request) error {
    log.Printf("[Request] %s %s", req.Method, req.URL)
    req = req.WithContext(context.WithValue(req.Context(), "start", time.Now()))
    return nil
}

func (a *AdvancedLoggingInterceptor) InterceptResponse(resp *http.Response) error {
    if start, ok := resp.Request.Context().Value("start").(time.Time); ok {
        duration := time.Since(start)
        log.Printf("[Response] %s %s in %v", resp.Request.Method, resp.Request.URL, duration)
    }
    return nil
}

c, err := unifi.NewClient(&unifi.ClientConfig{
    BaseURL: "https://unifi.localdomain",
    APIKey: "your-api-key",
    Interceptors: []unifi.ClientInterceptor{&AdvancedLoggingInterceptor{}},
})
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}
```

## Debugging and Logging

For troubleshooting, it might be useful to enable verbose logging. You can implement an interceptor to log additional
details like headers, body content, and timings. This can be enabled conditionally in your application's debug mode.

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
        for field, errMsg := range validationErr.Root {
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