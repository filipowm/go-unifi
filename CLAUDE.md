# go-unifi

Go client library for the UniFi Network controller API. Most resource types are **code-generated** from the UniFi controller's own API definitions; a thin
hand-written layer wraps them with a usable client.

## CRITICAL: generated code

- Files named `*.generated.go` start with `// Code generated ... DO NOT EDIT.` **Never hand-edit them.** A daily CI workflow regenerates and overwrites them.
- To change generated output: edit `codegen/internal/customizations.yml` (field type/name/tags/unmarshalers) and regenerate, OR add a hand-written sibling `.go` file.
  See `codegen/CLAUDE.md`.
- Generated CRUD methods are private (`getUser`, `listUser`); public wrappers (`GetUser`, `ListUser`) live in hand-written `.go` siblings.
- The generated `Client` interface embeds `InternalClient` (all resource CRUD) and adds transport/lifecycle methods plus `Internal()`/`Official()` accessors. The Official API (`unifi/official`) is a one-way dependency (`unifi -> official`): it imports nothing back, taking the transport as a structural `Doer`.

## Commands

```bash
go build ./...                                              # build
go test -cover -coverprofile=coverage.out -covermode atomic ./...   # full test + coverage
go test -run TestName ./unifi                              # single test
golangci-lint run                                          # lint
go generate unifi/codegen.go                               # regenerate resources (downloads controller)
go generate unifi/device.go                                # regenerate DeviceState stringer
```

A local-only `Makefile` wraps these: `make build|test|test-fast|cover|lint|fmt|check|generate`.
Codegen versions are overridable (both default `latest`): `VERSION` pins the Official OpenAPI spec version (the positional codegen arg) and `LEGACY_VERSION` pins the Internal/legacy controller version — e.g. `make generate-resources VERSION=10.1.85 LEGACY_VERSION=9.5.21`; `make generate` accepts the same vars.

## Layout

```
unifi/                  Single Go package: client + all resource types
  client.go             Client struct, config, auth
  requests.go           Do/Get/Post/Put/Delete, URL building, file upload
  interceptors.go       Request/response interceptors (API key, CSRF)
  api_paths.go          New vs old API style detection + path constants
  unifi_errors.go       ServerError type, ErrNotFound sentinel, error handler
  validation.go         go-playground/validator wrapper + custom regex validators
  json.go               Special unmarshalers (emptyStringInt, numberOrString, ...)
  *.generated.go        GENERATED — do not edit
  <resource>.go         Hand-written wrappers/business logic for <resource>
  official_surface.go   Internal()/Official() accessors + the Official-API capability gate
  features/             Controller feature-flag constants
  official/             Official UniFi OpenAPI (integration/v1) client — imports NOTHING from unifi
codegen/                Code-generation pipeline (see codegen/CLAUDE.md)
docs/                   Documentation
.unifi-version          Internal (legacy) API controller version marker (e.g. 9.5.21)
.unifi-version-official Official OpenAPI (integration/v1) spec version marker (e.g. 10.1.78)
```

## Style

- Go files use **tabs**; max line length 200. Formatting enforced by `gofumpt` + `goimports` + `gci` (run `golangci-lint run`).
- All client methods take `context.Context` as the first argument.
- Wrap returned errors with context using `%w`. Never return a "bare" error from a caller that adds no context — name what failed: `fmt.Errorf("injecting validation tags: %w", err)`. The exception: if the callee already wraps the error with sufficient context, bubbling it up verbatim is fine.
- See `.claude/rules/` for hand-written client conventions and test patterns.
