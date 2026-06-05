# go-unifi 1.11.0 — implementation plan (draft, hand-over ready)

This plan turns the [architect](architect-review.md) and [test-lead](test-review.md) reviews into an
ordered, dependency-aware implementation backlog. It is meant to be picked up in a **separate
implementation session**. No code has been changed yet.

See [summary.md](summary.md) for the finding→wave reference table and the verdict.

---

## 0. Decisions already made (do not re-litigate)

| Topic                  | Finding           | Decision                                                                                    | Implementation note                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
|------------------------|-------------------|---------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| TLS default            | ARCH-06           | **Keep the field, default to verify-ON, warn when disabled.** No rename, no hard API break. | A plain `bool` zero-value is `false`, so "default true unless explicitly opted out" is **not expressible with a plain `bool`**. Implement by changing `ClientConfig.VerifySSL` to **`*bool`** (`nil` ⇒ verify), or add an explicit `InsecureSkipVerify bool` opt-out and treat `VerifySSL` as deprecated. Whichever is chosen, emit a `Warn` log on every client build where verification ends up disabled. Document prominently.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `LtePoe`/`LteExtAnt`   | ARCH-02           | **Keep `booleanishString`; make its decoder permissive** (do NOT collapse to plain `bool`). | History matters here: PR #89 (Aug 2022) added `booleanishString` deliberately — the controller then sent `"enabled"`/`"disabled"`. Its field def has since migrated to `true\|false` (all of 9.3.45–10.0.162), so the read decoder is stale and now fails on the bare `true`/`false` the controller sends — *that* is the bug. The PUBLIC `Device.LtePoe`/`LteExtAnt` are already `bool`; `booleanishString` is only the internal read-path helper. **Fix:** make `booleanishString.UnmarshalJSON` permissive — accept bare `true`/`false`, quoted `"true"`/`"false"`, `"enabled"`/`"disabled"`, and `""`/`null`→false; never hard-error (a single bad field must not poison the whole `Device` decode). No `MarshalJSON` needed — writes already emit bare `true`/`false` via the `bool` field, matching the current regex. **Do NOT drop to plain `bool`:** that gambles on the unverified wire form (bare vs quoted) and abandons older-controller / `enabled`-`disabled` compat. Add a round-trip table test over every form. The one residual unknown (bare vs quoted on the current wire) is rendered moot by the permissive decoder. |
| Settings/codegen drift | ARCH-03 / ARCH-08 | **Generate the key→factory registry from codegen.**                                         | Quick-patch the 3 missing entries first to stop the bleeding, then have codegen emit the `settingFactories` registry (it already emits every `Setting*Key` + typed getter, so it owns the full set). Replace the hand-mirrored `expectedSettingTypes` test with one derived from the generated source so drift fails the build.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |

### Decisions confirmed with the maintainer (O1–O5)

All five were confirmed as the recommended option. They are settled — treat them as binding, not advisory.

| #  | Question                                                                                                       | Recommendation                                                                                                                                                                                                                                                                   |
|----|----------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| O1 | `SetSetting` (ARCH-08): expose in the `Client` interface (mirror `GetSetting`) or unexport to `setSetting`?    | **Expose** via `customizations.yml` so the public interface is symmetric with `GetSetting`.                                                                                                                                                                                      |
| O2 | `DpiApp`/`DpiGroup` (ARCH-08): delete generation, or add hand-written wrappers?                                | **Exclude `DpiApp`/`DpiGroup` from generation** (generation-level `excludeResources`, not just the interface) so no `dpi_*.generated.go` / dead private CRUD ships. Re-add wrappers (like `FirewallZoneMatrix`) only if a consumer needs DPI.                                    |
| O3 | `Get`/`Delete` JSON body (ARCH-23): drop the `reqBody` param (breaking) or keep + document?                    | **Defer to a major bump**; for 1.11.0 document the foot-gun. Revisit alongside real query-param support (ARCH-19).                                                                                                                                                               |
| O4 | `UseLocking` (ARCH-04): drop the coarse whole-request lock entirely, or keep as an explicit opt-in serializer? | **Drop it from the request path** (`net/http.Client` is already goroutine-safe) and protect only the actual shared state (`sysInfo`, CSRF token). If a per-controller serializer is genuinely wanted, make it a separate, clearly-named option never held across the round-trip. |
| O5 | Meta `rc:error` on HTTP 200 (ARCH-10): centralize in `handleResponse` or generate into templates?              | **Centralize** in the hand-written `handleResponse`, gated to only trigger when a `meta` block is present; then resolve the `user.go` TODO.                                                                                                                                      |

---

## 1. Constraints for the implementation session

- **Never hand-edit `*.generated.go`.** Change generated output via `codegen/customizations.yml`, the version JSON, or the `*.tmpl` templates, then run `go generate unifi/codegen.go`. Behavior changes
  go in hand-written siblings. (See `codegen/CLAUDE.md`.)
- **Go toolchain gotcha:** an old asdf Go (1.20.x) may shadow Homebrew Go on this machine and fail to parse `go 1.25.0` in `go.mod`. Prepend `/opt/homebrew/opt/go/bin` to `PATH` before
  building/testing/linting.
- **Conventions:** tabs, `gofumpt`/`goimports`/`gci`, max line 200, `context.Context` first, wrap errors with `%w`, table-driven `t.Parallel` tests with `testify`, `httptest` for round-trips. See
  `.claude/rules/`.
- **Coverage baseline:** capture `go test -cover ./unifi` and `./codegen` before starting so each wave can show movement.
- **Each finding ID is traceable** back to [architect-review.md](architect-review.md) / [test-review.md](test-review.md) for full problem statements and proposals.

---

## 2. Cross-cutting workstreams

Some findings are best executed as a single coherent change rather than one-by-one. Three workstreams cut across the waves:

- **WS-A — Kill codegen↔hand-written drift** (ARCH-03, ARCH-08, ARCH-12, ARCH-13, ARCH-31, TEST-04, TEST-06, TEST-20): generate the settings registry, add a reflection guard that every exported
  `*client` method is in the `Client` interface, decide Dpi/SetSetting, and (optionally, larger) generate the boilerplate wrappers.
- **WS-B — Offline-testable client + harness** (TEST-03, TEST-09, TEST-10, TEST-14, ARCH-24): add a seam to skip/inject `determineApiStyle`, extract its pure decision, build one shared
  `newTestClient`/`newControllerServer` helper, and generate a mock. This single seam unblocks the entire 0%-coverage wrapper layer.
- **WS-C — Security & robustness** (ARCH-06, ARCH-15, ARCH-16, ARCH-19, TEST-02): TLS default, download timeouts/integrity, atomic extraction, query-param support, and pinning the extract-path guards.

---

## 3. Waves

Effort: **S** < ~2h · **M** < ~1 day · **L** > 1 day. Risk is the chance the change regresses behavior.

### Wave 0 — P0 hotfixes (shipped bugs) — *do first, ship as a patch if needed*

| ID      | Action                                                                                                                                                                                            | Effort | Risk | Depends on |
|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|------|------------|
| ARCH-03 | Add the 3 missing `settingFactories` entries (`mdns`, `roaming_assistant`, `traffic_flow`) + matching test pins. Unbreaks 6 public methods.                                                       | S      | Low  | —          |
| ARCH-01 | Stop holding `c.lock` across the HTTP call in `Version()`; double-checked cache read/fetch/store. Add a `UseLocking:true` deadlock regression test.                                               | M      | Med  | —          |
| ARCH-02 | Make `booleanishString.UnmarshalJSON` permissive (bare/quoted bool, enabled/disabled, empty/null→false); keep public `bool`; round-trip table test. **Do not** collapse to plain `bool` (see §0). | M      | Med  | —          |
| TEST-01 | Golden/snapshot test for `api.go.tmpl`/`apiv2.go.tmpl` (incl. one V2 resource). Guards every later codegen change in this plan.                                                                   | M      | Low  | —          |

**Acceptance:** the 3 bugs have failing-before/passing-after tests; `go test -race ./unifi` clean for `Version()`; golden test green and wired into CI.

### Wave 1 — P1 hardening + the load-bearing test gaps

| ID      | Action                                                                                                                                                                                                                                   | Effort | Risk | Depends on    |
|---------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|------|---------------|
| ARCH-04 | Fine-grained sync: `sysInfo` behind `atomic.Pointer`/dedicated mutex (incl. the `NewClient` write); CSRF token behind a mutex in `CSRFInterceptor`. Drop coarse request lock (see O4). Document the concurrency contract. `-race` tests. | M      | Med  | ARCH-01       |
| ARCH-05 | `HandleError`: read body once into a capped buffer; always return a populated `*ServerError{Status,Method,URL}` even on empty/non-JSON bodies. Map 404 → `ErrNotFound` via `Is`/`Unwrap`.                                                | M      | Med  | ARCH-22       |
| ARCH-06 | TLS verify-by-default + warn (see decision table for the `*bool`/opt-out mechanism). Update README/config docs.                                                                                                                          | M      | Med  | —             |
| ARCH-07 | `numberOrString`: treat `null` as empty; normalize/reject other scalars. Add table test.                                                                                                                                                 | S      | Low  | —             |
| ARCH-08 | Generate the settings registry from codegen; reflection guard that exported `*client` methods ∈ `Client`; resolve O1 (`SetSetting`) and O2 (`Dpi`).                                                                                      | M      | Med  | WS-A, TEST-01 |
| ARCH-22 | Add `Unwrap()` to `ValidationError`/`ServerError`; sort `ValidationError.Error()` keys; check the `errors.As` bool in `Validate`.                                                                                                        | S      | Low  | —             |
| TEST-03 | Wrapper test suite on the WS-B seam: `CreateUser`, `OverrideUserFingerprint` (DELETE vs PUT), `GetSite`/`GetDevice`, `SetSetting`/`GetSetting`, command bodies, list-envelope unwrap, `Put`/`Delete`.                                    | L      | Low  | WS-B, TEST-09 |
| TEST-05 | Assert the `ErrNotFound` contract (incl. identity across `%w`) inside the TEST-03 suite.                                                                                                                                                 | M      | Low  | TEST-03       |
| TEST-04 | Table tests for `customizeBaseType`/`customizeResource` per-resource special cases.                                                                                                                                                      | M      | Low  | TEST-06       |
| TEST-06 | Delete the confirmed dead duplicated `IsSetting()` block in `customizeBaseType` (keep the switch case).                                                                                                                                  | S      | Low  | —             |
| TEST-02 | Negative tests for `sanitizeExtractedPath`, `copyWithLimit` bomb cap, oversized zip entry.                                                                                                                                               | S      | Low  | —             |
| TEST-08 | Gate live-network/`go run` codegen tests behind `testing.Short()` / `//go:build integration`; route firmware API through the existing `UnifiVersionProvider` seam where possible.                                                        | M      | Low  | —             |
| TEST-09 | Extract `apiStyleFromStatus(status, isAPIKey)` pure fn + unit-test all branches; route the probe through an injectable seam; optional `ClientConfig.APIStyle` override for offline construction.                                         | M      | Med  | —             |
| TEST-10 | Generate a `Client` mock (moq/mockgen in a hand-written sibling so daily regen keeps it synced) + exported offline test constructor.                                                                                                     | M      | Low  | TEST-09       |

**Acceptance:** `-race` clean under concurrent `Version()`/`Get`/`Post`; `errors.Is(err, ErrNotFound)` holds for both 404 and empty-data; wrapper layer coverage off 0%; `go test -short ./codegen` runs
offline; secure-by-default verified by test.

### Wave 2 — P2 quality & codegen robustness

| ID      | Action                                                                                                                                            | Effort | Risk |
|---------|---------------------------------------------------------------------------------------------------------------------------------------------------|--------|------|
| ARCH-10 | Centralize Meta `rc:error` (200-with-error) handling in `handleResponse` (see O5); resolve `user.go` TODO; test.                                  | M      | Med  |
| ARCH-11 | Decode based on body (treat `io.EOF` as empty) instead of the `ContentLength==0` short-circuit.                                                   | S      | Low  |
| ARCH-13 | v1 template: stop returning `ErrNotFound` from successful create/update; align with v2. Document `ErrNotFound` is get/list-single only.           | M      | Med  |
| ARCH-14 | Codegen: warn on dropped/colliding fields, skip only failing nested child, deterministic key ordering, strict mode for CI; golden type-diff test. | M      | Med  |
| ARCH-15 | Download pipeline: shared `http.Client` with timeout + cancellable context; checksum/host validation; re-validate firmware channel/product.       | M      | Med  |
| ARCH-16 | Atomic extraction (temp dir + rename, or `.complete` sentinel) so partial dirs aren't accepted.                                                   | M      | Low  |
| ARCH-18 | Define interceptor dedup semantics (by concrete type or drop it); change `AddInterceptor` to take `ClientInterceptor` not `*ClientInterceptor`.   | S      | Low  |
| ARCH-19 | First-class query-param support in customizations/templates; interim guard rejecting `?` in id-suffixed `resourcePath`.                           | M      | Med  |
| ARCH-20 | Factor shared template header/defines/struct+unmarshal block into a common partial parsed into both templates.                                    | M      | Low  |
| ARCH-21 | Apply customizations exactly once; split resource-level overrides; remove dead double-apply.                                                      | M      | Med  |
| ARCH-09 | Stop mutating caller-owned `ClientConfig` (URL, UserAgent); use locals/normalized copy.                                                           | S      | Low  |
| TEST-11 | Tests for `numberOrString`/`emptyStringInt` branches, `Logout`, `Version`, `Meta.error()`.                                                        | M      | Low  |
| TEST-12 | Extract pure `buildMultipartUpload`; unit-test field defaulting/MIME/escapeQuotes; upload round-trip incl. `X-Requested-With`.                    | M      | Low  |
| TEST-13 | Thread codegen logger through `generate`/options; let `newValidator` accept extra validators; make API-path sets value-returning/immutable.       | M      | Med  |
| TEST-14 | Single shared `unifi/testhelpers_test.go` (`newTestClient`, `newControllerServer`); migrate existing tests; kill the swallowed-error pattern.     | M      | Low  |
| TEST-15 | Add ctx-accepting variants for `Version`/`Login`/`Logout`/`GetSystemInformation` (or thread ctx into private helpers).                            | M      | Med  |
| TEST-16 | `utils_test.go` for `ensurePath`/`findProjectRoot`/`findCodegenDir`; inject codegen v2 base dir into `generateCode`.                              | S      | Low  |

### Wave 3 — P3 polish, ergonomics & test-bloat removal

| ID      | Action                                                                                                                                                                         | Effort | Risk |
|---------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|------|
| ARCH-12 | (Larger) codegen emits public CRUD wrappers for no-custom-logic resources → delete ~22 boilerplate files.                                                                      | L      | Med  |
| ARCH-23 | Drop/deprecate `reqBody` on `Get`/`Delete` (see O3) or add explicit params; doc.                                                                                               | M      | Med  |
| ARCH-24 | Reuse `c.http` (with per-request redirect policy) in `determineApiStyle` instead of a throwaway client.                                                                        | S      | Low  |
| ARCH-25 | `type Feature string`; type the constants + `GetFeature`/`IsFeatureEnabled`; expose the known set; add a test.                                                                 | M      | Low  |
| ARCH-26 | Standardize `meta` JSON-tag casing; fix the misleading `getOldSysInfo` decode shape.                                                                                           | S      | Low  |
| ARCH-27 | Delete dead/misleading code: `emptyStringInt.MarshalJSON` (if unused), `portalfile.go` import-fix block + no-op unmarshal, `unifi.go` tombstone; doc partial-CRUD intent.      | S      | Low  |
| ARCH-28 | (Larger) replace the regex "type interpreter" with a table-driven classifier / `regexp/syntax` AST inspection.                                                                 | L      | Med  |
| ARCH-29 | Collapse `Logger` non-`f` methods onto `f` variants; stop embedding `*logrus.Logger`; add `logging_test.go`.                                                                   | M      | Low  |
| ARCH-30 | Codegen IO fixes: move `defer Body.Close()` before status check; explicit `Close` checks; harden path checks; delete leftover `ace.jar`; `filepath.IsAbs`; `flag.ExitOnError`. | M      | Low  |
| ARCH-31 | Settings follow the wrapper convention; URL style → data; gate apiv2 ID code; explicit returns slice; validate YAML override combos.                                           | M      | Low  |
| TEST-17 | Delete redundant `TestFieldInfoFromValidation`; inline one-case `t.Run`; collapse the 3 setting registry tests; drop the duplicated 39-entry type literal.                     | S      | Low  |
| TEST-18 | Replace `reflect.DeepEqual`/`t.Fatalf` in `network_test.go` with narrowed `assert.Equal`.                                                                                      | S      | Low  |
| TEST-19 | Fix `TestRequestHeaders` shared-interceptor parallel reads (own client per subtest).                                                                                           | S      | Low  |
| TEST-20 | Assert stable generated client ordering + end-to-end YAML client-function wiring.                                                                                              | S      | Low  |

---

## 4. Suggested sequencing

1. **Wave 0** (independent P0 fixes; TEST-01 first so all later codegen edits are guarded).
2. **WS-B seam** (TEST-09 → TEST-10/TEST-14) early in Wave 1 — it unblocks TEST-03/05 and the whole wrapper-coverage story.
3. **WS-A** (ARCH-08 + TEST-04/06) once TEST-01 guards template output.
4. Remaining Wave 1, then Wave 2, then Wave 3. The two **L**-effort refactors (ARCH-12, ARCH-28) are optional and can be deferred to a later release without blocking anything else.

## 5. Definition of done (per wave)

- New/changed behavior covered by a failing-before/passing-after test.
- `go build ./...`, `go test ./...`, `golangci-lint run` green (with the PATH fix).
- Generated files regenerated via `go generate`, never hand-edited; generated diff reviewed.
- Coverage moved in the right direction vs the captured baseline.
- Each addressed finding ID referenced in the commit/PR for traceability back to this plan.
