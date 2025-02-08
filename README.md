# UniFi Go SDK
[![GoDoc](https://godoc.org/github.com/filipowm/go-unifi?status.svg)](https://godoc.org/github.com/filipowm/go-unifi)
![GitHub Release](https://img.shields.io/github/v/release/filipowm/go-unifi)
![GitHub branch check runs](https://img.shields.io/github/check-runs/filipowm/go-unifi/main)
![GitHub License](https://img.shields.io/github/license/filipowm/go-unifi)

This SDK provides a Go client for the UniFi Network Controller API. It is used primarily in the [Terraform provider for UniFi](https://github.com/filipowm/terraform-provider-unifi),
but can be used independently for any Go project requiring UniFi Network Controller API integration.

## Features

- Great UniFi Network Controller API coverage through automated code generation and manually added code for undocumented endpoints
- Generated data models from UniFi Controller API specifications
- Daily automated updates to track the latest UniFi Controller versions
- Easy to use client with support for API Key and username/password authentication
- Support for multiple UniFi Controller versions
- Strong typing for all API models with Go structs

## Code Generation

The data models and basic REST methods are generated from JSON specifications found in the UniFi Controller JAR files. Those JSON specs show all fields and the associated regex/validation information.
This ensures accuracy and completeness of the API coverage. However, code generation is not perfect and some endpoints might be missing, or not covered perfectly by the generated code. We hope to rely on official API specifications as soon as they are available.

To regenerate the code for the latest UniFi Controller version:

```bash
go generate unifi/codegen.go
```

**Note:** While the current code generation approach works, we're exploring better ways to extract API specifications. There is no official API specifications available, and the UniFi Controller JAR is obfuscated, making it
challenging to directly use Java classes. Contributions and suggestions for improvements are welcome!

## Usage

Unifi client support both username/password and API Key authentication. It is recommended to use API Key authentication for better security,
as well as dedicated user restricted to local access only.

### Obtaining an API Key
1. Open your Site in UniFi Site Manager
2. Click on `Control Plane -> Admins & Users`.
3. Select your Admin user.
4. Click `Create API Key`.
5. Add a name for your API Key.
6. Copy the key and store it securely, as it will only be displayed once.
7. Click Done to ensure the key is hashed and securely stored.
8. Use the API Key 🎉

### Client Initialization

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
	BaseURL: "https://unifi.localdomain",
	APIKey: "your-api-key",
})
```

Instead of API Key, you can also use username/password for authentication:

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    BaseURL: "https://unifi.localdomain",
    Username: "your-username",
    Password: "your-password",
})
```

If you are using self-signed certificates on your UniFi Controller, you can disable certificate verification:

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    ...
    VerifySSL: false,
})
```

List of available client configuration options is available [here](https://pkg.go.dev/github.com/filipowm/go-unifi/unifi#ClientConfig).

### Customizing HTTP Client

You can customize underlying HTTP client by using `HttpCustomizer` interface:

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    ...
    HttpCustomizer: func(transport *http.Transport) error {
        transport.MaxIdleConns = 10
        return nil
    },
})
```

### Using interceptors

You can use interceptors to modify requests and responses. This gives you more control over the client behavior
and flexibility to add custom logic.

To use interceptor logic, you need to create a struct implementing [ClientInterceptor](https://pkg.go.dev/github.com/filipowm/go-unifi/unifi#ClientInterceptor). interface.
For example, you can use interceptors to log requests and responses:

```go
type LoggingInterceptor struct{}

func (l *LoggingInterceptor) InterceptRequest(req *http.Request) error {
    log.Printf("Request: %s %s", req.Method, req.URL)
    return nil
}

func (l *LoggingInterceptor) InterceptResponse(resp *http.Response) error {
	log.Printf("Response status: %d", resp.StatusCode)
    return nil
}

c, err := unifi.NewClient(&unifi.ClientConfig{
	    ...
    Interceptors: []unifi.ClientInterceptor{&LoggingInterceptor{}},
})
```

### Examples

List all available networks:
```go
network, err := c.ListNetwork(ctx, "site-name")
```

Create user assigned to network:
```go
user, err := c.CreateUser(ctx, "site-name", &unifi.User{
	Name:      "My Network User",
	MAC:       "00:00:00:00:00:00",
	NetworkID: network[0].ID,
	IP:        "10.0.21.37",
})
```

## Plans

- [ ] Increase API coverage, or modify code generation to rely on the official UniFi Controller API specifications
- [ ] Improve error handling (currently only basic error handling is implemented and some of the errors are swallowed)
- [x] Improve client code for better usability
- [x] Support API Key authentication
- [ ] Generate client code for currently generated API structures, for use within or outside the Terraform provider
- [ ] Increase test coverage
- [ ] Implement validation for fields and structures
- [ ] Add more documentation and examples
- [ ] Bugfixing...

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change. I will be happy to find additional maintainers!

## Acknowledgment

This project is a fork of [paultyng/go-unifi](https://github.com/paultyng/go-unifi). Huge thanks to Paul Tyng together with the rest of maintainers for creating and maintaining the original SDK,
which provided an excellent foundation for this fork, and is great piece of engineering work. The fork was created to introduce several improvements including keeping it up to date with the latest UniFi Controller versions, more dev-friendly client usage, enhanced error handling, additional API endpoints support,
improved documentation, better test coverage, and various bug fixes. It's goal is to provide a stable, up to date and reliable SDK for the UniFi Network Controller API.
