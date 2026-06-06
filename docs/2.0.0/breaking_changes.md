# go-unifi 2.0.0 — API breaking changes

This document is the authoritative changelog of every public-API behavior or signature change introduced
during the 2.0.0 migration (epic [#117](https://github.com/filipowm/go-unifi/issues/117)). It is keyed to
the 13 impact-ordered breaking changes enumerated in the epic, each carrying a verified **DONE/PENDING**
status, a migration note, and a provenance link.

> Status key: **DONE** = change is in the `feat/2.0.0` tree and verified against the actual code.
> **PENDING** = planned; not yet landed.

---

## Breaking changes — 13-row index (epic #117)

| # | Change | Impact | Status |
|---|--------|--------|--------|
| 1 | [API-key authentication only](#1-api-key-authentication-only) | High | **PENDING** |
| 2 | [TLS verify-by-default](#2-tls-verify-by-default) | High | **DONE** |
| 3 | [Go version bump to 1.26](#3-go-version-bump-to-126) | Medium | **DONE** |
| 4 | [OpenAPI-shaped structs](#4-openapi-shaped-structs) | Medium | **PENDING** |
| 5 | [Occasional field renames](#5-occasional-field-renames) | Medium | **PENDING** |
| 6 | [New `integration/v1` API surface](#6-new-integrationv1-api-surface) | Medium | **DONE** |
| 7 | [`Client` gains `SetSetting`](#7-client-gains-setsetting) | Medium | **DONE** |
| 8 | [`Client` gains `*Context` variants](#8-client-gains-context-variants) | Medium | **DONE** |
| 9 | [v1 `Create`/`Update` no longer return `ErrNotFound`](#9-v1-createupdate-no-longer-return-errnotfound) | Medium | **DONE** |
| 10 | [`meta.rc=="error"` on HTTP 200 → `*ServerError`](#10-metarcerror-on-http-200--servererror) | Medium | **DONE** |
| 11 | [Map 404 → `ErrNotFound`](#11-map-404--errnotfound) | Low | **DONE** |
| 12 | [`UseLocking` will be a no-op](#12-uselocking-will-be-a-no-op) | Low | **DONE** |
| 13 | [Remove CSRF handling](#13-remove-csrf-handling) | Low | **PENDING** |

---

### 1. API-key authentication only

**Status: PENDING** — user/password auth is still present; retirement issues are tracked under #125.

**Behavioral change (runtime break).** Username/password authentication (`ClientConfig.User`/`Password`) will
be retired. Every consumer must authenticate with an API key via `ClientConfig.APIKey`.

```go
// before
cfg := &unifi.ClientConfig{User: "admin", Password: "secret"}
// after
cfg := &unifi.ClientConfig{APIKey: "your-api-key"}
```

Every consumer, including the Terraform provider, must migrate to API-key auth before upgrading to 2.0.0.
Once landed, the `User`, `Password`, `RememberMe` fields will be removed from `ClientConfig`; setting them
will be a compile error.

**Provenance:** epic #117, issue #125 (API-key auth retirement).

---

### 2. TLS verify-by-default

**Status: DONE** — landed in Wave 1 (ARCH-06).

**Signature change + behavioral flip (compile break + silent runtime change).** `ClientConfig.VerifySSL bool`
was renamed and inverted to `SkipVerifySSL bool`. The zero value now enables verification (secure by
default).

```go
// before
config := &unifi.ClientConfig{VerifySSL: false} // disabled verification
// after
config := &unifi.ClientConfig{SkipVerifySSL: true} // disable for self-signed cert
```

A caller that left `VerifySSL` unset got verification OFF; the same zero value now gives verification ON.
Connections to self-signed controllers break silently at runtime — except the rename forces a compile error
at every call site that touched the field, surfacing the flip at build time. Disabling verification is
logged at WARN level on every client build.

**Provenance:** ARCH-06, Wave 1.

---

### 3. Go version bump to 1.26

**Status: DONE** — `go 1.26.0` in `go.mod`.

**Build constraint change.** The module requires Go 1.26 or later. Consumers still on Go 1.24/1.25 must
upgrade their toolchain.

**Provenance:** epic #117, `go.mod`.

---

### 4. OpenAPI-shaped structs

**Status: PENDING** — no resources migrated yet; landing per-resource as part of the OpenAPI generator wave.

**Type change (compile break for each migrated resource).** Resources generated from the official OpenAPI
spec (`integration.json`) may adopt different field names, types, or nesting than the legacy reverse-engineered
shapes. Each resource is migrated individually; the breaking surface is bounded to that resource's struct.

Migration: compile against the new module version; fix any field references reported by the compiler.

**Provenance:** epic #117, OpenAPI-generator wave (issues TBD).

---

### 5. Occasional field renames

**Status: PENDING** — no field renames yet; land alongside each OpenAPI-driven resource migration.

**Signature change (compile break per renamed field).** Where the official spec uses a different field name
from the legacy one, the generated struct adopts the spec name. Each rename is documented in this file when
it lands.

Migration: compile-error-driven; rename at each call site.

**Provenance:** epic #117, per-resource OpenAPI migration.

---

### 6. New `integration/v1` API surface

**Status: DONE** — runtime seam landed in PR #119 (`feat/2.0.0`).

**Interface change (compile break for custom `Client` implementations).** The `Client` interface now exposes
two new accessors: `Internal() InternalClient` (the legacy resource API, unchanged call site) and
`Official() official.Client` (the Official UniFi OpenAPI surface). All legacy resource CRUD methods are
moved into the embedded `InternalClient` interface; existing call sites remain source-compatible.

```go
// before: no routing — one flat Client interface
client.GetNetwork(ctx, site, id)

// after: explicit surface selection (both forms work)
client.GetNetwork(ctx, site, id)           // still works — InternalClient is embedded
client.Internal().GetNetwork(ctx, site, id) // explicit form, documents intent
client.Official().Sites(ctx)               // new Official API surface
```

Custom `Client` implementations must add `Internal()` and `Official()` methods. The moq `ClientMock` is
regenerated automatically.

`client.Official()` operations are gated: they return `ErrOfficialAPIDisabled` when
`ClientConfig.DisableOfficialAPI` is set, or `ErrOfficialAPIUnavailable` on old-style controllers, non-API-key
auth, or controller versions below 10.1.68. Use `errors.Is(err, unifi.ErrOfficialAPIUnavailable)`.

**Provenance:** epic #117, PR #119.

---

### 7. `Client` gains `SetSetting`

**Status: DONE** — landed in Wave 1 (ARCH-08).

**Interface addition (compile break for custom `Client` implementations).** The generated `Client` interface
declares:

```go
SetSetting(ctx context.Context, site string, key string, reqBody any) (any, error)
```

The concrete `*client` already implemented it; it was simply missing from the interface. Any third-party
type that implements `unifi.Client` must add this method; the moq `ClientMock` is regenerated automatically.

**Provenance:** ARCH-08, Wave 1.

---

### 8. `Client` gains `*Context` variants

**Status: DONE** — landed in Wave 2 (TEST-15).

**Interface addition (compile break for custom `Client` implementations).** Four context-first lifecycle
methods are now part of the `Client` interface:

```go
LoginContext(ctx context.Context) error
LogoutContext(ctx context.Context) error
VersionContext(ctx context.Context) (string, error)
GetSystemInformationContext(ctx context.Context) (*SysInfo, error)
```

The original no-ctx methods (`Login`/`Logout`/`Version`/`GetSystemInformation`) are unchanged and remain
source-compatible; they delegate to the ctx variants internally. Any third-party type implementing
`unifi.Client` must add these four methods; the moq `ClientMock` is regenerated automatically.

**Provenance:** TEST-15, Wave 2.

---

### 9. v1 `Create`/`Update` no longer return `ErrNotFound`

**Status: DONE** — landed in Wave 2 (ARCH-13).

**Behavioral change (no compile break).** The v1-REST template used to return `ErrNotFound` from a
*successful* create/update when the response data array length was not exactly 1 — semantically wrong and
inconsistent with the v2 template. The ~29 generated v1 `create<X>`/`update<X>` methods now return a
descriptive error instead:

```go
fmt.Errorf("unexpected response: expected 1 <X>, got %d", n)
```

Any caller doing `errors.Is(err, ErrNotFound)` on a create or update path will no longer match — that
branch was always incorrect. Treat any non-nil error from `Create<X>`/`Update<X>` as a failure.
`ErrNotFound` remains the contract for `Get<X>` / single-resource lookups only.

**Provenance:** ARCH-13, Wave 2.

---

### 10. `meta.rc=="error"` on HTTP 200 → `*ServerError`

**Status: DONE** — landed in Wave 2 (ARCH-10).

**Behavioral change (no compile break).** The UniFi v1 API can return HTTP 200 with `meta.rc=="error"` (a
soft / application-level failure). Previously this was caught in exactly one place (`CreateUser`) and
silently swallowed everywhere else. The check is now centralized in `handleResponse`, so **every** decoded
`{meta,data}` 200 with `rc=="error"` surfaces a `*ServerError` carrying the controller's `rc`/`msg`
(enriched with status/method/URL).

```go
var serverErr *unifi.ServerError
if errors.As(err, &serverErr) { /* ... */ }
```

It is **not** `ErrNotFound` (`errors.Is(err, ErrNotFound) == false`), so genuine empty-data 200s and real
404s are unaffected. `CreateUser` retains its nested per-object meta check.

**Provenance:** ARCH-10, Wave 2.

---

### 11. Map 404 → `ErrNotFound`

**Status: DONE** — landed in Wave 1 (ARCH-05).

**Behavioral widening (no compile break).** `(*ServerError).Is` maps a `*ServerError` with
`StatusCode == 404` to the `ErrNotFound` sentinel. Previously only the hand-written list/get wrappers
returned `ErrNotFound`; now a genuine 404 from **any** endpoint also satisfies
`errors.Is(err, ErrNotFound)`. A consumer that distinguished "404 server error" from the `ErrNotFound`
sentinel now sees them as equal.

**Provenance:** ARCH-05, Wave 1.

---

### 12. `UseLocking` will be a no-op

**Status: DONE** — landed in Wave 1 (ARCH-04).

**Behavioral change (no compile break).** `net/http.Client` is goroutine-safe and the client no longer
serializes requests. `ClientConfig.UseLocking` is retained for source compatibility but is marked
`// Deprecated:` and has **no effect** — setting it `true` or `false` changes nothing. Remove it from your
config.

**Provenance:** ARCH-04, Wave 1.

---

### 13. Remove CSRF handling

**Status: PENDING** — CSRF interceptor still active; retirement lands with issue #125 (API-key-only auth).

**Type/behavior removal.** With username/password auth retired (#1), the `CSRFInterceptor` and its
token-management logic become dead code and will be removed. The `CSRFInterceptor` exported type will
disappear; any code referencing it directly will fail to compile.

Migration: remove any direct use of `CSRFInterceptor`; the token is managed transparently for user/pass auth
today and is irrelevant after API-key-only auth lands.

**Provenance:** epic #117, dependent on #1 (API-key-only auth).

---

## Additional changes — provenance index

Changes already documented in earlier waves that complement the 13-row table. Nothing here is new;
entries are relocated from the prior wave-by-wave structure for traceability.

### A. `CSRFInterceptor.CSRFToken`: exported field → accessor method (ARCH-04, Wave 1)

**Signature change (compile break).** The previously-exported, directly-settable `CSRFToken string` field
became an unexported field with a read-only accessor:

```go
// before
type CSRFInterceptor struct { CSRFToken string }
csrf.CSRFToken = "tok"  // settable
tok := csrf.CSRFToken   // field read
// after
type CSRFInterceptor struct { csrfToken string /* guarded by sync.RWMutex */ }
tok := csrf.CSRFToken() // accessor method; no setter
```

The token is managed internally and captured automatically from controller responses (race-safe). Any code
that read or set the field, or used a composite literal with `CSRFToken:`, no longer compiles.

### B. `DpiApp`/`DpiGroup` types removed (ARCH-08, Wave 1)

**Type removal.** The unused `DpiApp` and `DpiGroup` types and their CRUD (`dpi_app.generated.go`,
`dpi_group.generated.go`) were dead code — excluded from the `Client` interface yet still shipped — and are
excluded from generation entirely. No `Client` method ever exposed them, so typical consumers are unaffected;
any code directly referencing `unifi.DpiApp` / `unifi.DpiGroup` struct types must remove it. DPI *settings*
remain available via `SettingDpi`.

### C. `(*client).AddInterceptor` signature: `*ClientInterceptor` → `ClientInterceptor` (ARCH-18, Wave 2)

**Signature change (compile break for direct callers).** `AddInterceptor` now takes the interceptor by
value, matching how the interceptor slice and `ClientConfig.Interceptors` are typed:

```go
// before
c.AddInterceptor(&myInterceptor)
// after
c.AddInterceptor(myInterceptor)
```

Dedup semantics also changed: previously by interface `==` (could panic on non-comparable types); now by
concrete type via `reflect.TypeOf` — only one interceptor of a given concrete type is retained, and
non-comparable interceptors no longer panic. `AddInterceptor` is **not** part of the public `Client`
interface, so consumers using the `Client` interface are unaffected.

### D. `Client` interface split into `InternalClient` + `Internal()`/`Official()` accessors (#119)

See [#6 — New `integration/v1` API surface](#6-new-integrationv1-api-surface) above for the full entry. The
`InternalClient` embedded interface and the `Internal()`/`Official()` accessors are the structural
implementation of that row.

### E. Official API unavailable on classic/old-style controllers (#119)

**New runtime contract (no compile break).** `client.Official()` always returns a non-nil client, but every
operation is gated. They return:

- `ErrOfficialAPIDisabled` when `ClientConfig.DisableOfficialAPI` is set (fails fast, no probe).
- `ErrOfficialAPIUnavailable` on an old-style (classic) controller, non-API-key auth, or a controller
  version below 10.1.68.

Match with `errors.Is(err, unifi.ErrOfficialAPIUnavailable)` / `errors.Is(err, unifi.ErrOfficialAPIDisabled)`.
The Internal API is unaffected.

### F. Internal codegen-tool API changes (not public `unifi` surface)

These affect only forks of the generator:

- `DownloadAndExtract` gained a leading `*http.Client` parameter (TEST-07, Wave 1).
- `DownloadAndExtract`/`downloadJar` gained a leading `context.Context` parameter (ARCH-15, Wave 2).
