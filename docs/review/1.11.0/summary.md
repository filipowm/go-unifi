# go-unifi — code review summary (target 1.11.0)

Two-persona deep review of the `unifi` and `codegen` packages, run as a multi-agent workflow
(5 architect + 3 test-lead area reviewers → synthesized into one report per persona). The lead
re-verified every P0 and the security/contract P1s against source before hand-over.

**Documents in this folder**

- [architect-review.md](architect-review.md) — 31 findings (architecture, design, security, performance, concurrency, error-handling).
- [test-review.md](test-review.md) — 20 findings (coverage, test design, testability, test bloat).
- [plan.md](plan.md) — phased, dependency-aware implementation plan with the decisions baked in.

> **Status: review only — no code changed.** Implementation is to be carried out in a separate session using `plan.md`.

---

## Verdict

The architecture is genuinely good: a clean construction/request pipeline, a real interceptor
abstraction, a pluggable error handler and logger, and honest security hardening in the codegen
download/extract path (decompression-bomb caps + zip-slip defense). What drags it down is **three
shipped, runtime-broken contracts** and one **systemic theme — drift between the hand-written layer
and the codegen source of truth, with nothing enforcing their agreement**. On the test side the
foundations are sharp (HTTP machinery, interceptors, error parsing, type-inference all well covered)
but coverage is **absent exactly where the project's value lives**: every hand-written resource
wrapper is at 0%, and the templates that emit *all* generated code are asserted by nothing.

## Findings at a glance

| Persona   | P0    | P1     | P2     | P3     | Total  |
|-----------|-------|--------|--------|--------|--------|
| Architect | 3     | 5      | 14     | 9      | 31     |
| Test lead | 1     | 9      | 6      | 4      | 20     |
| **Total** | **4** | **14** | **20** | **13** | **51** |

### ✅ Verified shipped bugs (re-checked against source)

| ID      | Bug                                                                                                                                | Evidence                                                                                                                                                                     |
|---------|------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ARCH-01 | `Version()` self-deadlocks any `UseLocking:true` client (re-entrant `sync.Mutex`).                                                 | `client.go:166` locks → `…executeRequest` re-locks `client.go`/`requests.go:131`.                                                                                            |
| ARCH-02 | `booleanishString` accepts only `"enabled"`/`"disabled"`, but the controller now sends `true\|false` → whole `Device` fetch fails. | PR #89 (2022) added the override for the then-`enabled`/`disabled` wire format; the field def migrated to `true\|false` (9.3.45–10.0.162) and the decoder was never updated. |
| ARCH-03 | 3 missing `settingFactories` entries → 6 public `Setting*` methods always error.                                                   | 40 factory entries vs 43 generated `Setting*Key`; `mdns`/`roaming_assistant`/`traffic_flow` missing.                                                                         |
| ARCH-08 | `SetSetting` absent from the `Client` interface; `DpiApp`/`DpiGroup` are dead generated code.                                      | confirmed: not in `client.generated.go`; `dpi_*.generated.go` have private CRUD, no wrappers.                                                                                |

## Decisions taken (see plan.md §0 for detail)

| Topic                                                                                                 | Decision                                                                                                                                                                                           |
|-------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ARCH-06 TLS default                                                                                   | **Default to verify-ON + warn** when disabled. Shipped as an inverted plain bool: rename `VerifySSL bool` → `SkipVerifySSL bool` (zero value `false` verifies, `true` disables). |
| ARCH-02 `LtePoe`/`LteExtAnt`                                                                          | **Make `booleanishString` permissive** (bare/quoted bool + enabled/disabled + empty/null); keep public `bool`. *Not* plain `bool` — the override was a real 2022 fix and the wire format migrated. |
| ARCH-03/08 drift                                                                                      | **Generate the settings registry from codegen** + a build-failing drift-guard test.                                                                                                                |
| Open: `SetSetting` expose · `Dpi` delete-vs-wrap · `Get/Delete` body · `UseLocking` · Meta `rc:error` | Recommendations recorded in plan.md §0 (O1–O5) for the implementer to confirm.                                                                                                                     |

---

## Reference table — findings → priority → plan wave

Waves: **W0** P0 hotfixes · **W1** P1 hardening + load-bearing test gaps · **W2** P2 quality/codegen robustness · **W3** P3 polish + test-bloat removal. ✅ = re-verified against source. Full detail per
ID in the linked persona reports.

### Architect ([architect-review.md](architect-review.md))

| ID        | P  | Effort | Wave | Title                                                                                        |
|-----------|----|--------|------|----------------------------------------------------------------------------------------------|
| ARCH-01 ✅ | P0 | M      | W0   | Re-entrant deadlock: Version() with UseLocking=true self-deadlocks the caller                |
| ARCH-02 ✅ | P0 | M      | W0   | booleanishString can't decode its wired fields, failing the whole Device fetch               |
| ARCH-03 ✅ | P0 | S      | W0   | Three Setting getters/updaters broken: settingFactories drifted from codegen                 |
| ARCH-04   | P1 | M      | W1   | sysInfo cache and CSRF token are racy; UseLocking conflates cache guard + serializer         |
| ARCH-05   | P1 | M      | W1   | HandleError discards status/method/URL on empty/non-JSON bodies; ErrNotFound misses 404s     |
| ARCH-06 ✅ | P1 | M      | W1   | TLS verification is OFF by default (VerifySSL zero-value disables checks)                    |
| ARCH-07 ✅ | P1 | S      | W1   | numberOrString decodes JSON null to the literal string "null"                                |
| ARCH-08 ✅ | P1 | M      | W1   | Codegen↔hand-written drift unguarded across the public surface (SYSTEMIC)                    |
| ARCH-09   | P2 | S      | W2   | Constructor mutates caller-owned ClientConfig (URL and UserAgent)                            |
| ARCH-10   | P2 | M      | W2   | Meta.error() soft-error (200 with rc:error) unchecked across generated CRUD                  |
| ARCH-11   | P2 | S      | W2   | ContentLength==0 short-circuit silently skips decoding valid bodies                          |
| ARCH-12   | P2 | L      | W3   | ~22 resource wrapper files are 100% pure-delegation boilerplate codegen could emit           |
| ARCH-13   | P2 | M      | W2   | v1 template returns ErrNotFound from successful Create/Update; untyped SetSetting/GetSetting |
| ARCH-14   | P2 | M      | W2   | Inference failures + CamelCase collisions silently drop fields from generated structs        |
| ARCH-15   | P2 | M      | W2   | No HTTP timeouts/cancellation and no integrity verification on the download pipeline         |
| ARCH-16   | P2 | M      | W2   | DownloadAndExtract accepts a partially-extracted dir from a prior crashed run                |
| ARCH-17   | P2 | M      | W1   | Coverage gaps for security-critical and error paths in the extract pipeline                  |
| ARCH-18   | P2 | S      | W2   | Interceptor dedup via slices.Contains on interface values is fragile and can panic           |
| ARCH-19   | P2 | M      | W2   | resourcePath query-string hack produces malformed V2 get/update/delete URLs                  |
| ARCH-20   | P2 | M      | W2   | ~60 lines of struct+UnmarshalJSON duplicated verbatim between the two templates              |
| ARCH-21   | P2 | M      | W2   | Redundant double application of customizations + dead duplicated IsSetting block             |
| ARCH-22   | P2 | S      | W1   | Custom errors omit Unwrap; ValidationError.Root unreachable; Error() nondeterministic        |
| ARCH-23   | P3 | M      | W3   | Get/Delete send a JSON request body (unconventional, surprising)                             |
| ARCH-24   | P3 | S      | W3   | determineApiStyle builds a throwaway http.Client missing timeout/jar                         |
| ARCH-25   | P3 | M      | W3   | features package constants are untyped strings with no linkage to the API                    |
| ARCH-26   | P3 | S      | W3   | Inconsistent Meta JSON-tag casing; misleading getOldSysInfo decode shape                     |
| ARCH-27   | P3 | S      | W3   | Dead/misleading clutter (emptyStringInt.MarshalJSON, portalfile boilerplate, tombstone)      |
| ARCH-28   | P3 | L      | W3   | Type-inference layer is a fragile hand-rolled regex interpreter                              |
| ARCH-29   | P3 | M      | W3   | Logger interface duplicates 10 methods and leaks logrus through embedding                    |
| ARCH-30   | P3 | M      | W3   | Codegen IO robustness: leaked body, swallowed Close errors, path-safety, cwd coupling        |
| ARCH-31   | P3 | M      | W3   | Generated codegen consistency warts (settings public-in-generated, hardcoded URLs, V2 ID)    |

### Test lead ([test-review.md](test-review.md))

| ID      | P  | Effort | Wave | Title                                                                                   |
|---------|----|--------|------|-----------------------------------------------------------------------------------------|
| TEST-01 | P0 | M      | W0   | Codegen resource templates have ZERO tests — core generated output is unverified        |
| TEST-02 | P1 | S      | W1   | Security guards (bomb limiter, zip-slip check) untested on the failure path             |
| TEST-03 | P1 | L      | W1   | All hand-written unifi/ resource wrappers at 0%; network-probe constructor is the cause |
| TEST-04 | P1 | M      | W1   | Per-resource codegen special-casing (customizeBaseType/Resource) ~18% covered           |
| TEST-05 | P1 | M      | W1   | ErrNotFound public sentinel contract never asserted (zero test references)              |
| TEST-06 | P1 | S      | W1   | Confirmed dead duplicated block in customizeBaseType                                    |
| TEST-07 | P1 | M      | W1   | Download/extract chain bound to http.DefaultClient with no seam                         |
| TEST-08 | P1 | M      | W1   | Codegen suite requires live internet by default (no -short/build-tag guard)             |
| TEST-09 | P1 | M      | W1   | determineApiStyle hard-wires its HTTP client; 302/api-key branches untested             |
| TEST-10 | P1 | M      | W1   | Downstream consumers cannot mock the generated Client interface                         |
| TEST-11 | P2 | M      | W2   | JSON edge-case unmarshalers + Logout/Version/Meta.error() untested                      |
| TEST-12 | P2 | M      | W2   | File-upload multipart machinery 0% covered, entangled with FS + network                 |
| TEST-13 | P2 | M      | W2   | Hidden mutable package globals constrain isolation and parallel-test safety             |
| TEST-14 | P2 | M      | W2   | No shared test-helper/fixture package — scaffolding duplicated 4+ ways                  |
| TEST-15 | P2 | M      | W2   | Context-first contract violated by Version/Logout/Login/GetSystemInformation            |
| TEST-16 | P2 | S      | W2   | utils.go has no dedicated test file; generateCode coupled to real FS layout             |
| TEST-17 | P3 | S      | W3   | Redundant type-inference tests + separately-maintained type literal (bloat)             |
| TEST-18 | P3 | S      | W3   | network_test.go uses reflect.DeepEqual + t.Fatalf; brittle whole-struct compare         |
| TEST-19 | P3 | S      | W3   | TestRequestHeaders parallel subtests share one interceptor (data-race-inviting)         |
| TEST-20 | P3 | S      | W3   | Generated client ordering + YAML client-function wiring asserted only on membership     |

---

*Generated from a multi-agent review workflow (10 agents, ~950k tokens). Findings are advisory; the
plan reflects the maintainer's decisions as of hand-over.*
