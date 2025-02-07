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
- Support for multiple UniFi Controller versions
- Strong typing for all API models with Go structs

## Code Generation

The data models and basic REST methodsare generated from JSON specifications found in the UniFi Controller JAR files. Those JSON specs show all fields and the associated regex/validation information.
This ensures accuracy and completeness of the API coverage. However, code generation is not perfect and some endpoints might be missing, or not covered perfectly by the generated code. We hope to rely on official API specifications as soon as they are available.

To regenerate the code for the latest UniFi Controller version:

```bash
go generate unifi/codegen.go
```

**Note:** While the current code generation approach works, we're exploring better ways to extract API specifications. There is no official API specifications available, and the UniFi Controller JAR is obfuscated, making it
challenging to directly use Java classes. Contributions and suggestions for improvements are welcome!

## Usage

TBD

## Plans

- [ ] Increase API coverage, or modify code generation to rely on the official UniFi Controller API specifications
- [ ] Improve error handling (currently only basic error handling is implemented and some of errors are swallowed)
- [ ] Improve client creation
- [ ] Support authentication via Control Plane API Key
- [ ] Generate client code for currently generated API structures, for use within or outside of the Terraform provider
- [ ] Increase test coverage
- [ ] Implement validation for all fields
- [ ] Add more documentation and examples
- [ ] Bugfixing...

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change. I will be happy to find additional maintainers!

## Acknowledgment

This project is a fork of [paultyng/go-unifi](https://github.com/paultyng/go-unifi). Huge thanks to Paul Tyng together with the rest of maintainers for creating and maintaining the original SDK,
which provided an excellent foundation for this fork, and is great piece of engineering work. The fork was created to introduce several improvements including keeping it up to date with the latest UniFi Controller versions, more dev-friendly client usage, enhanced error handling, additional API endpoints support,
improved documentation, better test coverage, and various bug fixes. It's goal is to provide a stable, up to date and reliable SDK for the UniFi Network Controller API.
