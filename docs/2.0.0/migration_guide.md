# go-unifi 1.x → 2.0 Migration Guide

This guide walks you through every breaking change introduced in go-unifi 2.0.0. It is written for
Go developers of all experience levels: a senior can skim the **Fast path** checklist and be done in
minutes; a junior can read each thematic section for the full story of what changed, why, and what
to look for in their own code.

You don't need to keep [breaking_changes.md](breaking_changes.md) open while reading this — all the
rationale and affected symbols are inlined here. That document is a cross-reference for completeness;
this one is your hands-on guide.

## Table of Contents

- [Fast path](#fast-path)
- [Authentication & TLS](#authentication--tls)
  - [API key replaces username/password](#api-key-replaces-usernamepassword)
  - [TLS verification now ON by default](#tls-verification-now-on-by-default)
  - [CSRF handling removed](#csrf-handling-removed)
- [Official API surface (additive, recommended)](#official-api-surface-additive-recommended)
- [Go version](#go-version)
- [Client interface additions](#client-interface-additions)
- [Error handling](#error-handling)
  - [`meta.rc=="error"` on HTTP 200 now surfaces as `*ServerError`](#metarcerror-on-http-200-now-surfaces-as-servererror)
  - [404 responses now satisfy `errors.Is(err, ErrNotFound)`](#404-responses-now-satisfy-errorsiserr-errnotfound)
  - [`Create`/`Update` no longer return `ErrNotFound` on unexpected responses](#createupdate-no-longer-return-errnotfound-on-unexpected-responses)
- [Types and methods](#types-and-methods)
  - [`NewBareClient` replaced by `NewClient` with `SkipSystemInfo: true`](#newbareclient-replaced-by-newclient-with-skipsysteminfo-true)
  - [New `Patch` method](#new-patch-method)
  - [`UseLocking` is a no-op](#uselocking-is-a-no-op)
- [Further reading](#further-reading)

---

## Fast path

Mechanical checklist — tick each off in order:

- [ ] Update the import path to `/v2`: `github.com/filipowm/go-unifi/v2/unifi`
- [ ] Update `go get`: `go get github.com/filipowm/go-unifi/v2`
- [ ] Replace `ClientConfig.User`/`Password`/`RememberMe` with `APIKey:`
- [ ] Remove any calls to `.Login()`, `.LoginContext()`, `.Logout()`, `.LogoutContext()`
- [ ] Rename `VerifySSL: false` → `SkipVerifySSL: true` (and double-check the logic flip — verify is now ON by default)
- [ ] Update your Go toolchain to Go 1.26 or newer
- [ ] Replace `NewBareClient(cfg)` with `NewClient(cfg)` + `SkipSystemInfo: true`
- [ ] Review error checks on `Create`/`Update` paths (no longer returns `ErrNotFound`)
- [ ] If your code implements the `Client` interface, add `SetSetting`, `VersionContext`, and `GetSystemInformationContext`
- [ ] Remove any direct use of `CSRFInterceptor` or `CsrfHeader`

---

## Authentication & TLS

### API key replaces username/password

**What changed.** Username/password authentication is gone. The `ClientConfig` fields
`User`/`Password`/`RememberMe` and the type `UserPassCredentials` no longer exist. The
`Login`/`LoginContext`/`Logout`/`LogoutContext` methods are removed from the `Client` interface.
Old-style (classic) controllers — which only ever supported user/pass — are also unsupported; constructing
a client against one returns `ErrOldStyleUnsupported` immediately.

**Why.** API keys are more secure (no session expiry, no CSRF exposure, scoped credentials) and are
the path Ubiquiti is investing in. Removing the fallback keeps the auth surface small and predictable.

```go
// before
cfg := &unifi.ClientConfig{
    URL:      "https://unifi.localdomain", // the field was always URL:, never BaseURL:
    User:     "admin",
    Password: "secret",
}
// after
cfg := &unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key", // obtain from Control Plane → Admins & Users
}
```

**Check your code.**
- `grep -r 'User:\|Password:\|RememberMe:\|UserPassCredentials' .`
- `grep -r '\.Login(\|\.Logout(' .`

API keys require UniFi Network 9.0.114 or newer on a UniFi OS (new-style) controller.

---

### TLS verification now ON by default

**What changed.** `ClientConfig.VerifySSL bool` was renamed **and inverted** to `SkipVerifySSL bool`.
The zero value now means *verify certificates* (secure by default). In 1.x the zero value `VerifySSL: false`
meant *skip verification* — the exact opposite.

**Why.** The old name was a footgun: leaving it at zero gave you an insecure connection by accident.
The renamed, inverted field means safe code requires no action; unsafe code is explicit and logged at WARN.

```go
// before — zero value silently skipped verification
cfg := &unifi.ClientConfig{VerifySSL: false} // verification OFF

// after — zero value verifies; set true only for self-signed certs
cfg := &unifi.ClientConfig{SkipVerifySSL: true} // disable for self-signed cert (logs a warning)
```

**Check your code.** `grep -r 'VerifySSL' .` — any hit that set `VerifySSL: false` was getting no TLS
verification and will now get full verification (good for most callers). If your controller uses a
self-signed cert you must add `SkipVerifySSL: true`.

---

### CSRF handling removed

**What changed.** With username/password auth gone (above), the `CSRFInterceptor` and its token logic
are removed. The `CsrfHeader` constant is also gone.

**Why.** CSRF tokens were session-cookie artifacts tied to user/pass auth. API-key auth doesn't use
session cookies, so CSRF management is irrelevant.

```go
// before
c.AddInterceptor(&unifi.CSRFInterceptor{}) // no-op now; type is gone
// after — simply remove the line
```

**Check your code.** `grep -r 'CSRFInterceptor\|CsrfHeader' .`

---

## Official API surface (additive, recommended)

**Recommended.** The Official API (`integration/v1`) is the surface Ubiquiti now maintains as a
versioned, stable contract. **Prefer it over the legacy Internal API, and migrate existing Internal
calls to their Official equivalents wherever a covered one exists.** The Internal surface stays
supported for everything the Official API doesn't yet cover, but new code should reach for
`c.Official()` first.

**What changed.** `c.Official()` returns a fluent client for the UniFi official OpenAPI
(`integration/v1`). This is new in 2.0.0 and is purely additive — no existing code changes.

**Why.** Ubiquiti now ships a versioned, stable OpenAPI specification. The Official surface lets you
use it without touching the Internal API.

```go
// Official API — new in 2.0.0; requires controller ≥ 10.1.78 with API-key auth
info, err := c.Official().Info().Get(ctx)
id, err := c.Official().Sites().ResolveID(ctx, "default") // legacy name → UUID

page, err := c.Official().Networks().ListPage(ctx, id, nil)
pol, err := c.Official().Firewall().CreatePolicy(ctx, id, body)
```

Operations on `c.Official()` return `ErrOfficialAPIUnavailable` if the controller is below `10.1.78`,
is old-style, or uses non-API-key auth. Set `DisableOfficialAPI: true` in `ClientConfig` to opt out
entirely (operations then fail fast with `ErrOfficialAPIDisabled`).

**Error handling.** The Official surface uses the same error types as the Internal surface. Errors
returned through the default transport are `*unifi.ServerError`, and HTTP 404 responses satisfy
`errors.Is(err, unifi.ErrNotFound)`:

```go
id, err := c.Official().Sites().ResolveID(ctx, "default")
if errors.Is(err, unifi.ErrNotFound) {
    // site not found
}
var serverErr *unifi.ServerError
if errors.As(err, &serverErr) {
    log.Printf("controller error %d: %s", serverErr.StatusCode, serverErr.Message)
}
```

In 2.0.0 the Internal surface remains the default — calling a resource method directly on `c` is
identical to calling it on `c.Internal()`. The default is expected to flip to Official in 3.0.0.

---

## Go version

**What changed.** The module requires Go 1.26 or later (`go 1.26.0` in `go.mod`).

**Why.** The Official API surface uses range-over-func iterators (`iter.Seq2`), which need Go ≥ 1.23. The
module pins `1.26` as the project's supported toolchain baseline (the `go` directive in `go.mod`) — not
because any single language feature demands it — so contributors and consumers build on one known-good floor.

**Check your code.** Run `go version` — if you're below 1.26 you'll see a build error immediately. Update
your toolchain (`go install golang.org/dl/go1.26.0@latest && go1.26.0 download`).

---

## Client interface additions

**What changed.** The `Client` interface has three new methods. If you maintain a custom type that
implements `Client` (rare — most callers use the concrete `*client` returned by `NewClient`), you
must add these:

```go
// Added — implement all three if you have a custom Client implementation
SetSetting(ctx context.Context, site string, key string, reqBody any) (any, error)
VersionContext(ctx context.Context) (string, error)
GetSystemInformationContext(ctx context.Context) (*SysInfo, error)
```

Note: `LoginContext`/`LogoutContext` were briefly added and then removed (with all of auth row #1 above).

**Why.** `SetSetting` was on the concrete struct but missing from the interface. The `*Context` variants
add proper context propagation for callers who need cancellation/timeouts on lifecycle calls.

**Check your code.** Only affects you if you have a type that satisfies `unifi.Client` by listing the
methods (e.g. a hand-written mock). The generated `ClientMock` (moq) is regenerated automatically.

---

## Error handling

### `meta.rc=="error"` on HTTP 200 now surfaces as `*ServerError`

**What changed.** The UniFi v1 API sometimes returns HTTP 200 with `meta.rc == "error"` — a soft
application-level failure. In 1.x this was caught in exactly one place (`CreateUser`) and silently
swallowed everywhere else. In 2.0.0 it is detected centrally in `handleResponse`, so **every**
decoded `{meta,data}` 200 with `rc=="error"` surfaces as a `*ServerError` carrying the controller's
`rc`/`msg`, enriched with status/method/URL.

**Why.** Silent swallowing of soft errors was a correctness hazard. Surfacing them uniformly lets
callers make explicit decisions.

```go
// before — rc=="error" 200 swallowed silently for most resources
_, err := c.CreateNetwork(ctx, "default", n)
// err could be nil even when the controller reported an application error

// after — same code; rc=="error" now produces *ServerError
var serverErr *unifi.ServerError
if errors.As(err, &serverErr) {
    log.Printf("controller error: %s", serverErr.Message)
}
```

**Check your code.** If you have error checks that assumed success whenever `err == nil` after a resource
call, they are now correct. If you specifically checked for `nil` and then inspected the response body
yourself, you can remove that logic.

---

### 404 responses now satisfy `errors.Is(err, ErrNotFound)`

**What changed.** Previously only hand-written list/get wrappers returned `ErrNotFound`. In 2.0.0,
`(*ServerError).Is` maps any `*ServerError` with `StatusCode == 404` to the `ErrNotFound` sentinel,
so a genuine HTTP 404 from **any** endpoint satisfies `errors.Is(err, unifi.ErrNotFound)`.

**Why.** Consistent error semantics: "not found" means `ErrNotFound`, regardless of which endpoint
produced the 404.

```go
// before/after — same check; now covers all 404s, not just specific resources
if errors.Is(err, unifi.ErrNotFound) {
    // resource doesn't exist
}
```

**Check your code.** If you were checking `err != nil && !errors.Is(err, ErrNotFound)` on create/get
paths, the behavior is now wider — any 404 will match. This is correct and desirable.

---

### `Create`/`Update` no longer return `ErrNotFound` on unexpected responses

**What changed.** The v1-REST template used to return `ErrNotFound` when a successful create/update
response contained a data array with a length other than 1. That was semantically wrong (you just
created something — it can't "not be found"). In 2.0.0 these return a descriptive error instead:
`fmt.Errorf("unexpected response: expected 1 <X>, got %d", n)`.

**Why.** `ErrNotFound` should only mean "the resource doesn't exist at that path." Using it on a
create path confused callers and hid real bugs.

```go
// before — ErrNotFound could (incorrectly) come back from a create
_, err := c.CreateNetwork(ctx, "default", n)
if errors.Is(err, unifi.ErrNotFound) { /* this branch could fire on create */ }

// after — ErrNotFound will NOT come from Create/Update
_, err = c.CreateNetwork(ctx, "default", n)
// non-nil err means failure; errors.Is(err, ErrNotFound) is always false here
```

**Check your code.** `grep -r 'ErrNotFound' .` — any `errors.Is(err, ErrNotFound)` that sits on a
`Create`/`Update` call path was dead code (the controller was returning an application error, not a
404). Remove those branches or replace with a generic `err != nil` check.

---

## Types and methods

### `NewBareClient` replaced by `NewClient` with `SkipSystemInfo: true`

**What changed.** The `NewBareClient` function is removed. Its purpose was to create a client without
the eager `GetSystemInformation()` round-trip. You now do this with the standard `NewClient` and the
`SkipSystemInfo: true` field.

**Why.** A separate constructor created two code paths to maintain. The `SkipSystemInfo` field achieves
the same behaviour without the duplication.

```go
// before
c, err := unifi.NewBareClient(&unifi.ClientConfig{
    URL:    "https://unifi.localdomain", // the field was always URL:, never BaseURL:
    APIKey: "your-api-key",
})
// after
c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:            "https://unifi.localdomain",
    APIKey:         "your-api-key",
    SkipSystemInfo: true, // skip the eager sysinfo check; errors surface on first API call
})
```

**Check your code.** `grep -r 'NewBareClient' .`

---

### New `Patch` method

**What changed.** The `Client` interface (and the concrete `*client`) now exposes a `Patch` method
alongside the existing `Get`/`Post`/`Put`/`Delete`. It sends an HTTP PATCH request for partial updates.

**Why.** PATCH is the standard verb for partial updates, and the escape-hatch surface (`Do`/`Get`/`Post`/
`Put`/`Delete`) was missing it. It is most useful against endpoints that actually accept PATCH — notably
the Official `integration/v1` surface and any new-style endpoint documented to support partial updates.

```go
// new in 2.0.0 — partial update via PATCH against a PATCH-capable endpoint
patch := struct{ Name string `json:"name"` }{Name: "updated"}
err := c.Patch(ctx, "/proxy/network/integration/v1/sites/<id>/...", patch, &resp)
```

`Patch` sends a literal HTTP PATCH — the target endpoint must accept it. Most legacy Internal REST
resources (e.g. `networkconf`) expect `PUT`; use `Put` for those. This is an additive change — no
existing code is broken.

---

### `UseLocking` is a no-op

**What changed.** `ClientConfig.UseLocking` is deprecated and has no effect. `net/http.Client` is
goroutine-safe; requests run concurrently and are not serialized.

**Why.** The old mutex serialisation was a performance hazard with no correctness benefit once the
underlying HTTP client became goroutine-safe.

```go
// before — had an effect (serialised requests)
cfg := &unifi.ClientConfig{UseLocking: true}

// after — field retained for source compat but ignored; remove it
cfg := &unifi.ClientConfig{} // concurrent by default
```

**Check your code.** `grep -r 'UseLocking' .` — you can remove the field; setting it has no effect.

---

## Further reading

- [breaking_changes.md](breaking_changes.md) — authoritative changelog with status, provenance, and
  code snippets for every break
- [compatibility_matrix.md](../compatibility_matrix.md) — go-unifi release ↔ controller version matrix
- [advanced_topics.md](../advanced_topics.md) — raw API calls (`Do`/`Get`/`Post`/`Put`/`Patch`/`Delete`),
  path-resolution rules, interceptors
