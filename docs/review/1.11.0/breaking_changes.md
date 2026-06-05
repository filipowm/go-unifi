# go-unifi 1.11.0 — API breaking changes

This document tracks every public-API behavior or signature change introduced while implementing the
[1.11.0 review](summary.md). Each entry links to the finding ID that motivated it and the migration
guidance for downstream consumers.

> Status: populated wave by wave during implementation. Empty sections mean no breaking change landed
> in that wave (yet).

## Wave 0 — P0 hotfixes

_No breaking changes._ All three P0 fixes (ARCH-01 deadlock, ARCH-02 permissive `booleanishString`
decode, ARCH-03 missing setting factories) are bug fixes that only make previously-broken paths work;
no public signature or documented behavior changes.

## Wave 1 — P1 hardening

Three public breaking changes landed in this wave. Two are TLS-related (ARCH-06) and one is the
concurrency cleanup (ARCH-04). The full migration walk-through lives in the
[migration guide](../../migrating_from_upstream.md) and [client configuration](../../configuration.md);
this section is the authoritative changelog entry.

### 1. `ClientConfig.VerifySSL` type changed: `bool` → `*bool` (ARCH-06)

**Signature change (compile break).** The field type is now `*bool`:

```go
// before
VerifySSL bool
// after
VerifySSL *bool
```

Every caller that set `VerifySSL` by value no longer compiles. Migrate by taking a pointer:

```go
// before
config := &unifi.ClientConfig{VerifySSL: false}
// after
config := &unifi.ClientConfig{VerifySSL: new(false)} // disable verification (self-signed cert)
```

Callers that never set the field are unaffected at compile time (the zero value is now `nil`), but see
the behavioral flip below.

### 2. TLS verification is now SECURE BY DEFAULT (ARCH-06)

**Behavioral flip (silent runtime break).** The default flipped from insecure to secure:

| | old (`bool`) | new (`*bool`) |
| --- | --- | --- |
| field unset / zero value | `false` → `InsecureSkipVerify: true` (verification OFF) | `nil` → verification **ON** |
| explicitly verify | `VerifySSL: true` | `VerifySSL: new(true)` or leave `nil` |
| explicitly skip | `VerifySSL: false` | `VerifySSL: new(false)` |

A caller that previously left `VerifySSL` unset got `InsecureSkipVerify: true` and now gets certificate
verification ON. **This will break connections to controllers using self-signed certificates** (the most
common UniFi deployment) at runtime, with no compile error — the call to `NewClient` succeeds and the
first request fails on TLS handshake instead. To restore the old behavior, set
`VerifySSL: new(false)` explicitly; disabling verification is logged at WARN level on every client build.

### 3. `ClientConfig.UseLocking` is now a deprecated no-op (ARCH-04)

**Behavioral change (no compile break).** `net/http.Client` is goroutine-safe and the client no longer
serializes requests, so the per-request locking the field used to gate has been removed. `UseLocking` is
retained for source compatibility but has **no effect** — setting it `true` or `false` changes nothing.
The field is marked `// Deprecated:` and can be removed from your config. Requests now always run
concurrently and are not serialized.

### 4. `Client` interface gained `SetSetting`; `DpiApp`/`DpiGroup` removed (ARCH-08)

**Interface addition (compile break for custom `Client` implementations).** The generated `Client`
interface now declares:

```go
SetSetting(ctx context.Context, site string, key string, reqBody any) (any, error)
```

The concrete `*client` already implemented it — it was simply unreachable through the interface (its
read counterpart `GetSetting` was exposed; `SetSetting` was not). Any third-party type that implements
`unifi.Client` (e.g. a hand-rolled fake) must add this method. The moq-generated `ClientMock` is
regenerated automatically and needs no manual change.

**Type removal.** The unused `DpiApp` and `DpiGroup` types and their CRUD (`dpi_app.generated.go`,
`dpi_group.generated.go`) were dead code — excluded from the `Client` interface yet still shipped — and
are now excluded from generation entirely. No `Client` method ever exposed them, so typical consumers
are unaffected; any code directly referencing the `unifi.DpiApp` / `unifi.DpiGroup` struct types must
remove it. (DPI *settings* remain available via `SettingDpi`, which is unrelated.)

> Note (internal): `DownloadAndExtract` in the `codegen` tool gained a leading `*http.Client` parameter
> (TEST-07). This is not part of the public `unifi` API surface and affects only forks of the generator.

## Wave 2 — P2 quality & codegen robustness

Four public breaking changes landed in this wave: one generated-code contract change (ARCH-13), one
interceptor API change (ARCH-18), one interface addition (TEST-15), and one error-handling behavioral
change (ARCH-10). All four are documented below.

### 1. Generated `Create<X>`/`Update<X>` no longer return `ErrNotFound` (ARCH-13)

**Behavioral change (no compile break).** The v1-REST template used to return the `ErrNotFound` sentinel
from a *successful* create/update whenever the response data array length was not exactly 1 — semantically
wrong (a successful write reporting "not found") and inconsistent with the v2 template. The ~29 generated
v1 `create<X>`/`update<X>` methods (and their public `Create<X>`/`Update<X>` wrappers) now return a
descriptive error instead:

```go
fmt.Errorf("unexpected response: expected 1 <X>, got %d", n)
```

Any caller doing `errors.Is(err, ErrNotFound)` on a **create or update** path will no longer match — that
branch was always incorrect. Treat any non-nil error from `Create<X>`/`Update<X>` as a failure.
`ErrNotFound` remains the contract for `Get<X>`/single-resource lookups (and list-single) only; this is
now documented in `codegen/CLAUDE.md`. The hand-written `CreateUser` wrapper (nested group/user endpoint)
still surfaces `ErrNotFound` for its inner lookup and is unaffected.

### 2. `(*client).AddInterceptor` signature changed: `*ClientInterceptor` → `ClientInterceptor` (ARCH-18)

**Signature change (compile break for direct callers).** `AddInterceptor` now takes the interceptor by
value, matching how the interceptor slice and `ClientConfig.Interceptors` are typed:

```go
// before
func (c *client) AddInterceptor(interceptor *ClientInterceptor)
// after
func (c *client) AddInterceptor(interceptor ClientInterceptor)
```

```go
// before
c.AddInterceptor(&myInterceptor)
// after
c.AddInterceptor(myInterceptor)
```

Dedup semantics also changed: previously by interface `==` (which panics on non-comparable dynamic types
and only matched identical pointers), now **by concrete type** via `reflect.TypeOf` — only one interceptor
of a given concrete type is retained, and non-comparable interceptors no longer panic. `AddInterceptor` is
**not** part of the public `Client` interface (it is interface-private), so consumers using the `Client`
interface are unaffected; this only touches code holding the concrete `*client`.

### 3. `Client` interface gained four `*Context` methods (TEST-15)

**Interface addition (compile break for custom `Client` implementations).** The generated `Client`
interface now declares context-first variants so cancellation/deadline behavior is testable:

```go
LoginContext(ctx context.Context) error
LogoutContext(ctx context.Context) error
VersionContext(ctx context.Context) (string, error)
GetSystemInformationContext(ctx context.Context) (*SysInfo, error)
```

The original no-ctx methods (`Login`/`Logout`/`Version`/`GetSystemInformation`) are unchanged and remain
source-compatible — they delegate internally to the ctx variants (preserving `c.timeout`, the `sysInfo`
cache + double-checked locking, and `Version()`'s error-swallow-to-`""`). Callers are unaffected. Any
third-party type implementing `unifi.Client` must add these four methods; the moq `ClientMock` is
regenerated automatically.

### 4. HTTP 200 with `meta.rc=="error"` now surfaces `*ServerError` (ARCH-10 / O5)

**Behavioral change (no compile break).** The UniFi v1 API can return HTTP 200 with a top-level
`meta.rc=="error"` (a soft / application-level failure). Previously this was checked in exactly one place
(`CreateUser`) and silently swallowed everywhere else (surfacing as empty data or a generic `ErrNotFound`).
The check is now centralized in the hand-written `handleResponse`, gated to only trigger when a `meta`
block is present, so **every** decoded `{meta,data}` 200 with `rc=="error"` now returns a `*ServerError`
carrying the controller's `rc`/`msg` (enriched with status/method/URL). Use `errors.As(err, &serverErr)`.
It is **not** `ErrNotFound` (`errors.Is(err, ErrNotFound) == false`), so genuine empty-data 200s and real
404s are unaffected. `CreateUser` retains its nested per-object meta check, so its behavior is preserved.

> Note (internal): the `codegen` tool's `DownloadAndExtract`/`downloadJar` gained a leading
> `context.Context` parameter (ARCH-15). This is not part of the public `unifi` API surface and affects
> only forks of the generator.
