---
paths:
  - "unifi/**/*.go"
---

# unifi client conventions (hand-written code)

Never edit `*.generated.go`. These rules apply to hand-written `.go` files.

## Wrapping generated code
- Generated CRUD methods are private (`getUser`, `listUser`, `createUser`). Expose them via public wrappers in the hand-written sibling: `GetUser`,
  `ListUser`, etc. Put custom logic (search-by-MAC, field init, custom marshaling) in the wrapper, not the generated file.

## Official API (`unifi/official`)
- One-way dependency: `unifi -> official`. The `official` package MUST import nothing from `unifi`; the transport is injected as the structural `Doer`
  (Get/Post/Put/Delete) and the capability check as a `Gate`. `*unifi.client` satisfies `Doer` via its public request methods.
- Official wrappers pass FULL leading-`/` paths (under `integrationV1Path`); `buildRequestURL` resolves them against the base URL, bypassing `APIPaths.ApiPath`.
- The Internal/Official split lives in `official_surface.go` (`Internal()`/`Official()` + the gate); do not hand-edit the split in `client.generated.go`.

## HTTP & client
- Use `c.Get/Post/Put/Delete` (or `c.Do`) from `requests.go`; don't build raw `http.Request`s.
- Every method takes `context.Context` first.
- Responses are wrapped `{ "meta": {...}, "data": [...] }`. Return the sentinel `ErrNotFound` when a list endpoint yields zero items.

## Errors
- Surface API failures as `ServerError` (status, method, URL, code, validation details). Wrap with `%w` when adding context.

## Logging
- The client embeds a `Logger`; use `c.Debugf(...)`, `c.Trace(...)`, etc. Don't import logrus directly in resource code.

## JSON edge cases
- For fields that may be empty-string-or-int / string-or-number / enabled-disabled, use the helpers in `json.go` (`emptyStringInt`, `numberOrString`,
  `booleanishString`) rather than ad-hoc unmarshaling.

## Validation
- Use `go-playground/validator` struct tags (`validate:"omitempty,ipv4"`). Register custom regex validators via `NewCustomRegexValidator` in `validation.go`.

## Comments
- Keep comments short — max 2 lines — explaining WHY, not what. Only exceed that for genuinely complex logic that can't be made self-evident by naming/structure. Don't narrate obvious code.
