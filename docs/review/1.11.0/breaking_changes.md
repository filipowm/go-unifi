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

_To be populated._
