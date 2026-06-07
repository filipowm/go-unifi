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
| 1 | [API-key authentication only](#1-api-key-authentication-only) | High | **DONE** |
| 2 | [TLS verify-by-default](#2-tls-verify-by-default) | High | **DONE** |
| 3 | [Go version bump to 1.26](#3-go-version-bump-to-126) | Medium | **DONE** |
| 4 | [OpenAPI-shaped structs](#4-openapi-shaped-structs) | Medium | **PENDING** |
| 5 | [Occasional field renames](#5-occasional-field-renames) | Medium | **PENDING** |
| 6 | [New `integration/v1` `APIStyle`](#6-new-integrationv1-apistyle) | Medium | **PENDING** |
| 7 | [`Client` gains `SetSetting`](#7-client-gains-setsetting) | Medium | **DONE** |
| 8 | [`Client` gains `*Context` variants](#8-client-gains-context-variants) | Medium | **DONE** |
| 9 | [v1 `Create`/`Update` no longer return `ErrNotFound`](#9-v1-createupdate-no-longer-return-errnotfound) | Medium | **DONE** |
| 10 | [`meta.rc=="error"` on HTTP 200 → `*ServerError`](#10-metarcerror-on-http-200--servererror) | Medium | **DONE** |
| 11 | [Map 404 → `ErrNotFound`](#11-map-404--errnotfound) | Low | **DONE** |
| 12 | [`UseLocking` is a no-op](#12-uselocking-is-a-no-op) | Low | **DONE** |
| 13 | [Remove CSRF handling](#13-remove-csrf-handling) | Low | **DONE** |

---

### 1. API-key authentication only

**Status: DONE** — landed in issue #125 (API-key-only auth).

**Behavioral change + signature change (compile break).** Username/password authentication
(`ClientConfig.User`/`Password`/`RememberMe`) is retired. Every consumer must authenticate with an API key
via `ClientConfig.APIKey`. The `Login`/`LoginContext`/`Logout`/`LogoutContext` methods are removed from
the `Client` interface. Old-style (classic) controllers, which only supported user/pass auth, are
no longer reachable and construction fails immediately with an explicit error.

```go
// before
cfg := &unifi.ClientConfig{User: "admin", Password: "secret"}
// after
cfg := &unifi.ClientConfig{APIKey: "your-api-key"}
```

Compile breaks: `ClientConfig.User`, `.Password`, `.RememberMe` removed; `UserPassCredentials` type removed;
`Login`/`LoginContext`/`Logout`/`LogoutContext` methods removed from `Client` interface;
`CsrfHeader` constant removed; `APIPaths.LoginPath`/`.LogoutPath` fields removed.

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

### 6. New `integration/v1` `APIStyle`

**Status: PENDING** — routing generated resources through the official `integration/v1` API is part of
the OpenAPI generator wave (same milestone as rows 4/5); not yet landed.

**Interface change (compile break for custom `Client` implementations).** A new `APIStyle` constant will
route code-generated resources through the UniFi official `integration/v1` API path instead of the legacy
`/api/s/{site}/...` endpoints. The generated CRUD methods and their structs will adopt OpenAPI-derived
shapes (see rows 4/5).

The `integration/v1` path is a capability layered on the new-style API — it is not a fourth independent
style alongside `V1`, `V2`, and `new-style`; see `unifi/api_paths.go` for the documented constraint.

Note: the `Official()` accessor and the `integration/v1` info/sites vertical that **did** land in PR #119
are documented in entries [D](#d-client-interface-split-into-internalclient--internalofficial-accessors-119)
and [E](#e-official-api-unavailable-on-classicold-style-controllers-119) of the provenance index below.

**Provenance:** epic #117, OpenAPI generator wave (issues TBD, same milestone as rows 4/5).

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

**Status: DONE** — landed in Wave 2 (TEST-15); `Login`/`Logout` ctx variants superseded by #125.

**Interface addition (compile break for custom `Client` implementations).** Two context-first lifecycle
methods are part of the `Client` interface:

```go
VersionContext(ctx context.Context) (string, error)
GetSystemInformationContext(ctx context.Context) (*SysInfo, error)
```

`LoginContext`/`LogoutContext` were added in Wave 2 but removed in issue #125 (API-key-only auth, row #1).
The no-ctx `Version`/`GetSystemInformation` remain source-compatible. Any third-party type implementing
`unifi.Client` must add these two methods; the moq `ClientMock` is regenerated automatically.

**Provenance:** TEST-15, Wave 2; #125 for Login/Logout removal.

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

### 12. `UseLocking` is a no-op

**Status: DONE** — landed in Wave 1 (ARCH-04).

**Behavioral change (no compile break).** `net/http.Client` is goroutine-safe and the client no longer
serializes requests. `ClientConfig.UseLocking` is retained for source compatibility but is marked
`// Deprecated:` and has **no effect** — setting it `true` or `false` changes nothing. Remove it from your
config.

**Provenance:** ARCH-04, Wave 1.

---

### 13. Remove CSRF handling

**Status: DONE** — landed in issue #125 (API-key-only auth).

**Type removal (compile break for direct users).** With username/password auth retired (#1), the
`CSRFInterceptor` and its token-management logic are removed. The `CsrfHeader` constant is also removed.

Migration: remove any direct use of `CSRFInterceptor` or `CsrfHeader`; neither is relevant with API-key
authentication.

**Provenance:** epic #117, issue #125 (API-key-only auth).

---

## Additional changes — provenance index

Changes already documented in earlier waves that complement the 13-row table. Nothing here is new;
entries are relocated from the prior wave-by-wave structure for traceability.

### A. `CSRFInterceptor.CSRFToken`: exported field → accessor method (ARCH-04, Wave 1) — superseded by #125

**Superseded by row [#13](#13-remove-csrf-handling).** `CSRFInterceptor` is now entirely removed. This Wave 1
intermediate step (field → accessor) is moot; the type is gone. Any code referencing `CSRFInterceptor` in
any form fails to compile.

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

See [#6 — New `integration/v1` `APIStyle`](#6-new-integrationv1-apistyle) above for the full entry. The
`InternalClient` embedded interface and the `Internal()`/`Official()` accessors are the structural
implementation of that row.

This is the **2.0.0-canonical-Internal** step: in 2.0.0 the embedded Internal surface stays the default, so
existing code is untouched; 3.0.0 is expected to flip the default to the Official client (the **3.0.0-flip**).

### E. Official API unavailable on classic/old-style controllers (#119)

**New runtime contract (no compile break).** `client.Official()` always returns a non-nil client, but every
operation is gated. They return:

- `ErrOfficialAPIDisabled` when `ClientConfig.DisableOfficialAPI` is set (fails fast, no probe).
- `ErrOfficialAPIUnavailable` on an old-style (classic) controller, non-API-key auth, or a controller
  version below 10.1.78.

Match with `errors.Is(err, unifi.ErrOfficialAPIUnavailable)` / `errors.Is(err, unifi.ErrOfficialAPIDisabled)`.
The Internal API is unaffected.

### F. Internal codegen-tool API changes (not public `unifi` surface)

These affect only forks of the generator:

- `DownloadAndExtract` gained a leading `*http.Client` parameter (TEST-07, Wave 1).
- `DownloadAndExtract`/`downloadJar` gained a leading `context.Context` parameter (ARCH-15, Wave 2).
- `internal.Generate` / `generateCode` gained a leading `floorFieldsDir` parameter; root `codegen` added a
  `-floor-version` flag and a frozen `codegen/v9.0.114/` floor snapshot (#126, below).

### G. Supported controller floor 9.0.114; pre-floor resources retired (#126)

**Status: DONE** — landed in issue #126 (apply the 9.0.114 floor).

**Support-contract change (no realized compile break in this change).** 2.0.0 sets **9.0.114** as the
supported controller floor: the Internal-API codegen now generates the resource set as a two-snapshot
**merge** — the `9.0.114` floor bounds the surface below while the pinned `9.5.21` snapshot supplies the
newest field shapes (newest wins). Any resource **retired before 9.0.114** is therefore not generated and
not shipped; the floor is enforced structurally so the daily auto-regen can never reintroduce a pre-floor
resource.

Realized drop set: **none at the resource level; 17 fields intentionally retired at the field level.**
The merge is newest-wins at struct-name granularity (whole-resource replacement, not field-level union):
when a resource exists in both snapshots, the 9.5.21 struct wins in full and any field present only in the
9.0.114 snapshot is silently dropped. The resource-level subset claim is correct: 9.0.114 has 74 resources
vs 9.5.21's 77, with only `SettingMdns`, `SettingRoamingAssistant`, and `SettingTrafficFlow` added after
the floor (and correctly kept). But at **field level**, 17 fields across 4 shared resources exist in
9.0.114 but NOT in 9.5.21 and are therefore **intentionally dropped** by the newest-wins merge:

| Resource | Dropped fields (present only in 9.0.114) |
|---|---|
| `ChannelPlan` | `ap_blacklisted_channels`, `conf_source`, `coupling`, `fitness`, `note`, `radio`, `satisfaction`, `satisfaction_table`, `site_blacklisted_channels` |
| `SettingIps` | `ad_blocking_configurations`, `ad_blocking_enabled`, `dns_filtering`, `dns_filters` |
| `SettingSuperMgmt` | `autobackup_s3_access_key`, `autobackup_s3_secret`, `autobackup_s3_bucket` |
| `NetworkConf` | `igmp_forward_unknown_multicast` |

**Decision: floor bounds resource membership, not fields.** These 17 fields are intentionally unsupported
in 2.0.0. The floor guarantees that no resource _type_ retired before 9.0.114 is generated; it does not
guarantee that every field present on a 9.0.114 controller is representable. Fields dropped by 9.5.21 are
treated as retired — callers on supported controllers (≥ 9.0.114) that relied on these fields must work
around their absence or request they be re-added via a hand-written sibling file.

The golden type-diff is empty because no _public type_ was added or removed versus the prior surface
(the 17 fields were already absent from the 9.5.21-frozen output). The three settings added after the floor
were correctly kept.

Migration: none required for the generated surface. Operationally, controllers older than **9.0.114** are
out of support for 2.0.0; the 17 fields listed above are not available regardless of controller version.

**Provenance:** epic #117 task 2 (retirement half), issue #126; frozen snapshot `codegen/v9.0.114/`,
pin in `unifi/codegen.go` (`-floor-version=9.0.114`).
