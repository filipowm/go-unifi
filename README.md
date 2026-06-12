# UniFi Go SDK

![GitHub Release](https://img.shields.io/github/v/release/filipowm/go-unifi)
![Supported Internal API Version](https://img.shields.io/badge/dynamic/regex?url=https%3A%2F%2Fraw.githubusercontent.com%2Ffilipowm%2Fgo-unifi%2Frefs%2Fheads%2Fmain%2F.unifi-version&search=(.*)%3F&logo=ubiquiti&label=Supported%20Internal%20API%20Version&color=yellow)
![Supported Official API Version](https://img.shields.io/badge/dynamic/regex?url=https%3A%2F%2Fraw.githubusercontent.com%2Ffilipowm%2Fgo-unifi%2Frefs%2Fheads%2Fmain%2F.unifi-version-official&search=(.*)%3F&logo=ubiquiti&label=Supported%20Official%20API%20Version&color=blue)
[![Docs](https://img.shields.io/badge/docs-reference-blue)](https://github.com/filipowm/go-unifi/blob/main/docs/readme.md)
[![Go Reference](https://pkg.go.dev/badge/github.com/filipowm/go-unifi/v2/unifi.svg)](https://pkg.go.dev/github.com/filipowm/go-unifi/v2/unifi)
![GitHub branch check runs](https://img.shields.io/github/check-runs/filipowm/go-unifi/main)
![GitHub License](https://img.shields.io/github/license/filipowm/go-unifi)

This SDK provides a Go client for the UniFi Network Controller API. It is used primarily in the [Terraform provider for UniFi](https://github.com/filipowm/terraform-provider-unifi),
but can be used independently for any Go project requiring UniFi Network Controller API integration.

Check out the detailed [documentation](docs/readme.md) for more information, including the
[1.x → 2.0 migration guide](docs/2.0.0/migration_guide.md) and the
[breaking-changes log](docs/2.0.0/breaking_changes.md).

## Features

- Great UniFi Network Controller API coverage through automated code generation and manually added code for undocumented endpoints
- Easy to use client with support for API Key authentication (username/password removed in 2.0.0)
- Generated data models from UniFi Controller API specifications
- Daily automated updates to track the latest UniFi Controller versions
- Support for multiple UniFi Controller versions
- Strong typing for all API models with Go structs

## Supported UniFi Controller Versions

API-key authentication (the only supported auth in 2.0.0) requires a new-style UniFi OS controller,
version **9.0.114** or newer. Old-style (classic) controllers are unsupported and construction fails
immediately with `ErrOldStyleUnsupported`.

The Internal API is tested against controller **9.5.21** (`.unifi-version`). The Official OpenAPI
surface (`c.Official()`) requires controller **10.1.78** or newer (`.unifi-version-official`).

The SDK is updated daily to track the latest UniFi Controller versions.
If you encounter any issues with the latest UniFi Controller version, please open an issue.

See the [compatibility matrix](docs/compatibility_matrix.md) for the mapping between `go-unifi` releases and supported UniFi Controller versions.
It also includes a changelog of breaking compatibility changes for each release.

## Code Generation

The data models and basic REST methods are generated from JSON specifications found in the UniFi Controller JAR files. Those JSON specs show all fields and the associated regex/validation information.
This ensures accuracy and completeness of the API coverage. However, code generation is not perfect and some endpoints might be missing, or not covered perfectly by the generated code. We hope to rely
on official API specifications as soon as they are available.

To regenerate the code for the latest UniFi Controller version:

```bash
make generate-resources
```

To regenerate for a specific controller version, override `VERSION`:

```bash
make generate-resources VERSION=9.3.45
```

`make generate` regenerates everything (resource types + the `DeviceState` stringer) and accepts the same `VERSION` override.

**Note:** While the current code generation approach works, we're exploring better ways to extract API specifications. There is no official API specifications available,
and the UniFi Controller JAR is obfuscated, making it challenging to directly use Java classes. Contributions and suggestions for improvements are welcome!

## Migrating from `paultyng/go-unifi`

If you already use `paultyng/go-unifi`, you can migrate to this SDK — it is a fork and the core client
methods remain the same.
Check out the [migration guide](docs/migrating_from_upstream.md) for information on how to migrate from the upstream `paultyng/go-unifi` SDK.

## Upgrading from go-unifi 1.x

See the [1.x → 2.0 migration guide](docs/2.0.0/migration_guide.md) for a step-by-step walkthrough of every
breaking change. A quick reference of all 10 breaks is in
[breaking_changes.md](docs/2.0.0/breaking_changes.md).

## Usage

The UniFi client requires API Key authentication (available from UniFi Controller 9.0.114+).

### Obtaining an API Key

1. Open your Site in UniFi Site Manager
2. Click on `Control Plane -> Admins & Users`.
3. Select your Admin user.
4. Click `Create API Key`.
5. Add a name for your API Key.
6. Copy the key and store it securely, as it will only be displayed once.
7. Click Done to ensure the key is hashed and securely stored.
8. Use the API Key

### Client Initialization

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key",
})
```

TLS certificate verification is **on by default** (secure by default). If you are using self-signed
certificates on your UniFi Controller, you can disable certificate verification by setting `SkipVerifySSL`
to `true` (the zero value `false` verifies). Disabling verification logs a
warning and makes the connection vulnerable to man-in-the-middle attacks, so prefer adding the
controller's CA instead.

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:           "https://unifi.localdomain",
    APIKey:        "your-api-key",
    SkipVerifySSL: true, // disable TLS verification (self-signed cert)
})
```

See the [full list of client configuration options](https://pkg.go.dev/github.com/filipowm/go-unifi/v2/unifi#ClientConfig) on pkg.go.dev.

### Internal vs Official API

The client exposes two API surfaces:

- **Internal** — the legacy UniFi Network API the SDK has always wrapped. Every resource method
  (`GetNetwork`, `ListUser`, …) lives here and is reachable directly on the client *or* via `c.Internal()`.
- **Official** — the official UniFi OpenAPI (`integration/v1`), reached via `c.Official()`. It requires a
  new-style UniFi OS controller (version `10.1.78`+) with **API-key** authentication. The surface is
  **fluent**: one accessor per resource group (`Firewall()`, `Networks()`, `Devices()`, …, derived from the
  OpenAPI tags), each returning an independently mockable per-group interface. Methods are **generated** from
  the committed OpenAPI snapshot in a uniform shape per resource. **List endpoints expose two methods** so
  draining is explicit and never accidental — `List…Page(ctx, …, *official.ListOptions)` returns a single
  **bounded** `official.Page[T]` (nil opts ⇒ the first page at the default size; `Limit` is clamped to 200),
  and `List…All(ctx, …, filter)` returns a lazy, abortable `iter.Seq2[T, error]` that pages on demand (range it and
  `break` to stop, or `official.Collect` it into a slice). Alongside: `Get…` (the single-item `…Details`) and
  `Create/Update/Patch…` (taking the `…CreateOrUpdate` body), plus the hand-written `Info().Get`,
  `Sites().ListPage`/`Sites().ListAll` and `Sites().ResolveID`.

```go
sites, err := c.Internal().ListSites(ctx)                     // legacy API (same as c.ListSites(ctx))
info, err := c.Official().Info().Get(ctx)                     // official OpenAPI
id, err := c.Official().Sites().ResolveID(ctx, "default")     // map a legacy site name to its official UUID (returns uuid.UUID)

page, err := c.Official().Networks().ListPage(ctx, id, nil)   // ONE bounded page (the safe default)
page, err = c.Official().Networks().ListPage(ctx, id, &official.ListOptions{Limit: 50, Filter: "name.eq('lan')"}) // bounded + filtered

for net, err := range c.Official().Networks().ListAll(ctx, id, "") { // lazy drain — break stops further fetches
	if err != nil { /* handle */ break }
	_ = net
}
all, err := official.Collect(c.Official().Networks().ListAll(ctx, id, "")) // explicit materialization into a slice

pol, err := c.Official().Firewall().CreatePolicy(ctx, id, body) // fluent, per-group accessor
```

> **Resource groups:** `Hotspot()` exposes Vouchers; `Firewall()` exposes `PatchPolicy` and other
> firewall resources. The Official surface has no top-level `Vouchers()` accessor — use `c.Official().Hotspot()`.

In **2.0.0 the Internal surface stays the canonical default**, so existing code is untouched — calling a
resource method on the client is identical to calling it on `c.Internal()`. **3.0.0 is expected to flip the
default to the Official client.** The Official client is gated: operations return
`unifi.ErrOfficialAPIUnavailable` on a classic/old-style controller, non-API-key auth, or a controller
below `10.1.78`, and `unifi.ErrOfficialAPIDisabled` when `ClientConfig.DisableOfficialAPI` is set. Match
either with `errors.Is`. Site identifiers differ between the surfaces — the Internal API uses the site
**name** while the Official API uses a **`uuid.UUID`** (from `github.com/google/uuid`) — so
`Official().Sites().ResolveID` maps the familiar name to the UUID for you. If you already have a
UUID string, use `uuid.Parse("…")` to convert it.

### Low-level API calls

For endpoints not covered by a generated method, the client exposes `Do`, `Get`, `Post`, `Put`, `Patch`,
and `Delete`. See [Advanced Topics](docs/advanced_topics.md) for the path-resolution rules and examples.

### Customizing HTTP Client

You can customize underlying HTTP client by using `HttpTransportCustomizer` interface:

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key",
    HttpTransportCustomizer: func (transport *http.Transport) (*http.Transport, error) {
        transport.MaxIdleConns = 10
        return transport, nil
    },
})
```

### Using interceptors

You can use interceptors to modify requests and responses. This gives you more control over the client behavior
and flexibility to add custom logic.

To use interceptor logic, you need to create a struct implementing [ClientInterceptor](https://pkg.go.dev/github.com/filipowm/go-unifi/v2/unifi#ClientInterceptor) interface.
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
    URL:          "https://unifi.localdomain",
    APIKey:       "your-api-key",
    Interceptors: []unifi.ClientInterceptor{&LoggingInterceptor{}},
})
```

### Client-side validation

The SDK provides basic validation for the API models. It is recommended to use it to ensure that the data you are sending
to the UniFi Controller is correct. The validation is based on the regex and validation rules provided in
the UniFi Controller API specs extracted from the JAR files.

Client supports 3 modes of validation:

- `unifi.SoftValidation` (_default_) - will log a warning if any of the fields are invalid before sending the request, but will not stop the request
- `unifi.HardValidation` - will return an error if any of the fields are invalid before sending the request
- `unifi.DisableValidation` - will disable validation completely

To change the validation mode, you can use the `ValidationMode` field in the client configuration:

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:            "https://unifi.localdomain",
    APIKey:         "your-api-key",
    ValidationMode: unifi.HardValidation,
})
```

If you use hard validation, you can get access to `unifi.ValidationError` struct, which contains information about the validation errors:

```go
n := &unifi.Network{
    Name:     "my-network",
    Purpose:  "invalid-purpose",
    IPSubnet: "10.0.0.10/24",
}
_, err = c.CreateNetwork(ctx, "default", n)

if err != nil {
    validationError := &unifi.ValidationError{}
    errors.As(err, &validationError)
    fmt.Printf("Error: %v\n", validationError)
    fmt.Printf("Root: %v\n", validationError.Root)
}
```

`Root` error is `validator.ValidationErrors` struct from [go-playground/validator](https://pkg.go.dev/github.com/go-playground/validator/v10#ValidationErrors),
which contains detailed information about the validation errors.

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

- [ ] Support Unifi Controller API V2
    - [x] AP Groups
    - [x] DNS Records
    - [x] Zone-based firewalls
    - [ ] Traffic management
    - [ ] other...?
- [x] Increase API coverage, or modify code generation to rely on the official UniFi Controller API specifications
- [x] Improve error handling (currently only basic error handling is implemented and error details are not propagated)
- [x] Improve client code for better usability
- [x] Support API Key authentication
- [x] Generate client code for currently generated API structures, for use within or outside the Terraform provider
- [ ] Increase test coverage
- [x] Implement validation for fields and structures
- [ ] Extend validators for more complex cases
- [x] Add more documentation and examples
- [ ] Bugfixing...

## Development

A `Makefile` provides common local tasks. Run `make help` for the full list. Most useful targets:

| Target           | Description                                                                  |
|------------------|------------------------------------------------------------------------------|
| `make build`     | Compile all packages                                                         |
| `make test`      | Run tests with coverage (generated files excluded from the report)           |
| `make test-fast` | Run tests without coverage; supports `RUN=TestName`                          |
| `make cover`     | Run tests and open the HTML coverage report                                  |
| `make lint`      | Run `golangci-lint`                                                          |
| `make fmt`       | Format code (gofumpt/goimports/gci via golangci-lint)                        |
| `make check`     | Build, lint and test — the pre-push gate                                     |
| `make generate`  | Regenerate resource types and the `DeviceState` stringer (accepts `VERSION`) |

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change. I will be happy to find additional
maintainers!

## Acknowledgment

This project is a fork of [paultyng/go-unifi](https://github.com/paultyng/go-unifi). Huge thanks to Paul Tyng together with the rest of maintainers for creating and maintaining the original SDK,
which provided an excellent foundation for this fork, and is great piece of engineering work. The fork was created to introduce several improvements including keeping it up to date with the latest
UniFi Controller versions, more dev-friendly client usage, enhanced error handling, additional API endpoints support,
improved documentation, better test coverage, and various bug fixes. It's goal is to provide a stable, up to date and reliable SDK for the UniFi Network Controller API.
