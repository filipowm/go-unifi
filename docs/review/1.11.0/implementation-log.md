# go-unifi 1.11.0 — implementation log

Tracks the workflow-driven implementation of [plan.md](plan.md). Scope: all P0/P1/P2 findings; **P3
discarded**; **ARCH-12 skipped** (deferred per plan). Branch: `chore/review-1.11.0`. Controller pin
for codegen: **9.5.21** (offline cache; regen via `cd unifi && go run ../codegen -version-base-dir=../codegen 9.5.21`).

## Baseline (pre-change)

- `go build ./...`: clean
- `golangci-lint run`: 0 issues
- `go test ./unifi/`: pass, coverage **10.0%**
- codegen: live-network tests present (not yet gated — TEST-08)

## Wave 0 — P0 hotfixes (ARCH-01, ARCH-02, ARCH-03, TEST-01)

- Status: **complete** ✅ (verify all-green; reviewed by architect + test-lead; 2 major review findings remediated)

**Implemented**
- **ARCH-01** (`unifi/client.go`, `client_test.go`): added dedicated `sysInfoMu sync.RWMutex` (separate from `c.lock`); `Version()` rewritten double-checked (RLock read → fetch holding no lock → Lock+recheck+store); `NewClient` sysInfo write guarded. Tests: `TestVersionWithLockingNoDeadlock` (goroutine+2s select timeout; empirically fails on old re-entrant code) + `TestVersionConcurrentCachedFetch` (50 goroutines under `-race`, torn-read + cache-hit invariant). The coarse `executeRequest` `useLocking` lock left for Wave 1 / ARCH-04.
- **ARCH-02** (`unifi/json.go`, `json_test.go`): `booleanishString.UnmarshalJSON` made permissive (true/"true"/enabled/"1"→true; everything else incl. null/garbage→false; never hard-errors). Underlying `bool`; no MarshalJSON. Table test over all forms.
- **ARCH-03** (`unifi/setting.go`, `setting_test.go`): registered the 3 missing keys (mdns / roaming_assistant / traffic_flow) in `settingFactories` + the `expectedSettingTypes` mirror — unbreaks 6 exported interface methods. Registry now 43/43. (Systemic drift guard deferred to W1 / ARCH-08.)
- **TEST-01** (`codegen/resources_golden_test.go`, `codegen/testdata/*.golden`): golden + shape + endpoint-path tests for `api.go.tmpl`/`apiv2.go.tmpl` via `NewResource`+`processJSON`+`GenerateCode`. Pins V1+V2 output, both custom-unmarshal branches (`bool(aux.X)` type-cast and `emptyBoolToTrue(aux.X)` func), Setting/Device/APGroup path logic. `-update-golden` flag.

**Review outcome:** no blockers, no regressions. Test-lead's 2 major findings (no concurrent `-race` Version test; golden fixture used a non-existent `boolFromBooleanish`) were remediated. Minor/nit findings (extra ARCH-02 edge cases, subtest-name slashes, pre-existing setting_test non-testify style) deferred — low value, some pre-existing.

**Breaking changes:** none.

**Verification (final, in-workflow):** `go build ./...` ✓ · `golangci-lint run` 0 issues ✓ · `go test ./unifi/...` ✓ · `go test ./unifi/ -run TestVersion -race` ✓ · `go vet ./codegen/...` ✓ · `go test ./codegen/ -run TestResourceGenerateCode` ✓

## Wave 1 — P1 hardening + load-bearing test gaps

- Status: **complete** ✅ (verify all-green incl. `-race` + offline `-short` codegen + regen-reproducible; architect + test-lead review found **no regressions**; 1 major review finding remediated)

**Implemented** — ARCH-22 (`Unwrap` on ValidationError/ServerError, deterministic `ValidationError.Error`, `errors.As` guard), ARCH-05 (`HandleError` keeps status/method/URL on empty/non-JSON bodies via capped read; 404 → `ErrNotFound` through `ServerError.Is`), ARCH-07 (`numberOrString` null → empty), ARCH-04 (CSRF token behind RWMutex; coarse per-request lock dropped — `net/http` is goroutine-safe; `UseLocking` now a deprecated no-op), ARCH-06 (TLS **secure-by-default**: `VerifySSL bool` → `SkipVerifySSL bool`, renamed + inverted so the zero value verifies, WARN on disable), TEST-09 (`apiStyleFromStatus` pure fn + probe through `c.http` + `ClientConfig.APIStyle` offline-construction seam), TEST-03/05 (wrapper coverage suite + `ErrNotFound` contract across `%w`), TEST-10 (moq `ClientMock` + offline constructor), TEST-06 (dead `customizeBaseType` block removed, regen zero-diff), TEST-04 (customize special-casing tests → 100%), **ARCH-08** (generated drift-proof settings registry via per-setting `init()` self-registration + `setting_registry.go`; reflection drift-guard `client_interface_test.go`; `SetSetting` exposed on `Client`; `DpiApp`/`DpiGroup` removed from generation), TEST-07 (`*http.Client` injected into download pipeline), TEST-02/ARCH-17 (bomb-cap + zip-slip + non-200 negative tests), TEST-08 (`go test -short ./codegen` now fully offline).

**Breaking changes** (see [breaking_changes.md](breaking_changes.md)): ARCH-06 `VerifySSL bool` → `SkipVerifySSL bool` (renamed + inverted) + secure-by-default flip; ARCH-04 `UseLocking` no-op; ARCH-08 `SetSetting` added to `Client`, `DpiApp`/`DpiGroup` removed; TEST-07 `DownloadAndExtract` signature (package-internal).

**Review follow-ups deferred (minor/nit):** `UnblockUserByMAC`/`KickUserByMAC` still 0% (structurally identical to tested siblings); stamgr error-propagation branch partial; drift sentinel covers 3 keys (self-registration makes new settings auto-register regardless); `HandleError` 1 MiB cap branch untested; TEST-09 duplicate API-key-rejection string. Tracked for a later cleanup pass.

**Verification (final, in-workflow):** `go build ./...` ✓ · `golangci-lint run` 0 issues ✓ · `go test ./unifi/...` ✓ · `go test ./unifi/ -race` ✓ · `go test -short ./codegen/...` offline ✓ · `go vet ./codegen/...` ✓ · regen-reproducible (zero generated diff) ✓

## Wave 2 — P2 quality & codegen robustness

- Status: **complete** ✅ (verify 8/8 all-green; architect + test-lead review found **no regressions**, rated the work "high-quality" / "among the strongest test work in this codebase"; 1 major review finding remediated in-workflow + 5 high-value minor/regression findings remediated in a follow-up workflow). Coverage: unifi **9.1% → 10.6%**, codegen **83.5% → 84.4%**.
- Commit: `d8f5f25` (single atomic code commit — see below).

**Orchestration.** Two workflows: the main Wave-2 workflow (Phase A codegen-regen serialized → Phase B `unifi` ‖ `codegen-runtime` disjoint streams → Phase C verify→review→remediate) and a follow-up minor-hardening workflow for the review minors. moq was pre-installed as a binary (`/Users/filipowm/go/bin/moq` v0.7.1) so the mock regenerates **offline** when the interface changes — the key de-risk for TEST-15. A false `verify-failed` (the regen/mock gate was worded "diff vs HEAD must be empty," but Wave 2 intentionally rewrites generated output and it was not yet committed) was diagnosed, the gate corrected to an **idempotency** test (regenerate the committed tree → no further change), and the workflow resumed so the in-workflow architect+test-lead review actually ran.

**Implemented**
- **Phase A — codegen templates + regenerate (serialized):** ARCH-20 (shared `common.tmpl` partial parsed into both templates — zero generated diff), ARCH-21 (drop dead double-apply in `collectResourceGenerators`; split resource- vs field-level overrides; remove dead `IsSetting` block), ARCH-14 (warn on dropped/colliding fields + deterministic JSON-key sort + opt-in `UNIFI_CODEGEN_STRICT` hard-fail + golden type-diff snapshot), **ARCH-13** (v1 create/update return `fmt.Errorf("unexpected response: expected 1 X, got N")` instead of `ErrNotFound`; `codegen/CLAUDE.md` documents ErrNotFound = get/list-single only; behavioral test `TestUserGroupCreateUpdateNotFoundContract` drives a real generated path), ARCH-19 (first-class `queryParams` map rendered after the `/%s` id segment + guard rejecting `?` in id-suffixed `resourcePath`; `DescribedFeature` migrated off the `?`-hack), **TEST-15** (Client interface gains `LoginContext`/`LogoutContext`/`VersionContext`/`GetSystemInformationContext`; no-ctx methods delegate; mock regenerated offline).
- **Phase B — `unifi` stream:** ARCH-11 (decode-on-body, `io.EOF`=empty, not `ContentLength==0`; single 64 MiB capped read with explicit overflow error), **ARCH-10/O5** (centralized Meta `rc:error` in `handleResponse` gated on a meta block present; `*ServerError` enriched with status/method/URL; `CreateUser` nested per-object check restored; `user.go` TODO resolved), ARCH-09 (stop mutating caller `ClientConfig` URL/UserAgent — local copy), ARCH-18 (`AddInterceptor(ClientInterceptor)` value sig + dedup by concrete type, panic-safe), TEST-11 (unmarshaler branches + `Logout`/`Version`/`Meta.error`), TEST-12 (pure `buildMultipartUpload` extraction + upload round-trip incl. `X-Requested-With`), TEST-13-unifi (`newValidator(...extra)` seam; value-returning `oldStyleAPI()`/`newStyleAPI()` copies), TEST-14 (consolidate onto shared `testhelpers_test.go`; race-safe request recording via mutex).
- **Phase B — `codegen-runtime` stream:** ARCH-15 (thread `context` + default timeout through `DownloadAndExtract`/`downloadJar`; https + Ubiquiti-host validation; firmware channel/product re-validation), ARCH-16 (atomic extraction: temp dir + rename, `.extract-complete` sentinel — partial/crashed extracts never accepted), TEST-13-codegen (thread a `Logger` through `generate`/options instead of the package global → removes the test race), TEST-16 (`utils_test.go` for `ensurePath`/`findProjectRoot`/`findCodegenDir`; inject the v2 base dir into `generateCode`).

**Review remediation.** Major (test-lead): ARCH-13 was protected only at golden-text level → added a behavioral test invoking a generated create/update path. Minor/regression follow-ups (separate workflow): ARCH-10 `*ServerError` now carries response status/method/URL (was `Server error (0) for  :`); **restored `CreateUser`'s nested per-object meta check** — its removal in the first pass was a regression + unnecessary breaking change, now **eliminated** (the previously-noted "ARCH-10-user" break is struck); ARCH-11 explicit "response body exceeded N bytes" overflow error + test; `testhelpers` request-recording race fixed with a mutex; TEST-15 `VersionContext` slow-path fetch-error subtest.

**Breaking changes** (see [breaking_changes.md](breaking_changes.md)): ARCH-13 (generated create/update no longer return `ErrNotFound`), ARCH-18 (`AddInterceptor` value signature + concrete-type dedup), TEST-15 (Client interface +4 `*Context` methods), ARCH-10 (HTTP 200 `meta.rc=="error"` → `*ServerError`). Internal: ARCH-15 (`DownloadAndExtract`/`downloadJar` gain a leading `context.Context` — codegen-only).

**Review follow-ups deferred (nit, documented):** ARCH-14 collision detection doesn't cross-check the whitespace-prefixed base-struct field keys (no real collision in the 9.5.21 catalog — `UNIFI_CODEGEN_STRICT` regen is clean); ARCH-16 publish window (RemoveAll→Rename) is crash-safe via the sentinel (re-extracts); ARCH-16 test couples to the `.tmp-` substring. Low value; tracked for a later cleanup pass.

**Verification (final, in-workflow + independently re-run by the orchestrator):** `go build ./...` ✓ · `golangci-lint run` 0 issues ✓ · `go test ./unifi/...` ✓ · `go test ./unifi/ -race` ✓ · `go test -short ./codegen/...` offline (`GOPROXY=off`) ✓ · `go test -short ./codegen/ -race` ✓ · `go vet ./codegen/...` ✓ · regen idempotent (H1==H2) ✓ · mock idempotent (M1==M2) ✓

> **Commit note:** Wave 2 landed as a **single atomic code commit** (`d8f5f25`), unlike Wave 1's lane-split. TEST-14's shared-helper consolidation made the `unifi` test files mutually compile-dependent, and the regenerated Client interface (TEST-15) couples the generated tree to the hand-written impl — so no finer split keeps every commit build+test green without de-consolidating TEST-14. Per-finding traceability lives in the commit body + this log. `.unifi-version` (user-owned, uncommitted) was kept out of the commit per the standing rule.

## Final whole-codebase review (process contract §2.8)

- Status: **complete** ✅ — verdict **ship-with-followups**. 33 agents: in-workflow baseline (9/9 green) → 8 parallel adversarial dimension reviewers (client/config, requests pipeline, error model, settings/wrappers, codegen templates, codegen resources, codegen download-security, tests+doc-accuracy) → per-finding adversarial verification (real? in-scope? severity?) → synthesis → gated remediation.
- **Zero confirmed in-scope blocker/major findings; zero must-fixes.** Of 23 verified findings, 9 were skipped (not real / out-of-scope / binding-decision outcomes), 14 confirmed real+in-scope but all **minor/nit, recommendation=document**. Final verify all-green; regen + mock idempotent.

**Acted on (high-value, low-risk):**
- **breaking_changes.md gaps closed** (found by the review): (4) `CSRFInterceptor.CSRFToken` exported field → accessor method (ARCH-04, Wave 1) — a real compile break that was undocumented; (5) the 404 → `ErrNotFound` widening via `ServerError.Is` (ARCH-05, Wave 1) — an undocumented behavioral widening.
- **Capital-`M` `Meta` regression test** (`TestHandleResponseMetaRcErrorCapitalMeta`, requests_test.go): pins the centralized soft-error probe's reliance on `encoding/json` case-insensitive key matching (real v1 envelopes emit capital `Meta`; the probe tags lowercase `meta`). Guards against a future exact-case refactor silently re-breaking what ARCH-10/O5 fixed — the one test the review actively recommended scheduling.

**Post-review fixes applied (the two latent-correctness items, TDD failing-then-green):**
- `FR-codegen-templates-1` (codegen/resources.go) ✅: `QuerySuffix()` now doubles `%` (`strings.ReplaceAll(r.QueryString, "%", "%%")`) so a future `queryParams` value that url-encodes to contain `%` (e.g. `&`→`%26`) stays a literal in the `fmt.Sprintf` format string instead of a malformed verb. Zero generated diff (today's only param is `%`-free). Test: `codegen TestQuerySuffix` `%`-case.
- `FR-error-model-3` (unifi/validation.go) ✅: `ValidationError.Error()` now surfaces `Root.Error()` when `Messages` is empty (the non-struct fallback), instead of rendering an empty `"validation failed: \n"`. Test: `TestValidateNonStructFallback` asserts the root cause is in the message.

**Deferred follow-ups (remaining; all nit/cosmetic doc/comment/defense-in-depth — optional later pass):**
- `FR-error-model-1` (req): note that the O5 soft-error check only runs for non-nil `respBody` (bodyless DELETE/Post soft-200 rc:errors still pass — strictly better than pre-1.11.0, not a regression).
- `FR-requests-pipeline-2` (req): comment that `decodeResponseBody` parses the buffered body twice (meta probe + decode); the *network* read is single. Accepted O5 tradeoff.
- `FR-requests-pipeline-4` (unifi_errors): optionally truncate the raw-body fallback in `errorBodyFallbackMessage` to keep error strings log-friendly (body already capped at 1 MiB).
- `FR-client-config-1` (client): document `ClientConfig.Interceptors` dedup-by-concrete-type + built-in precedence on the public field godoc; optional `Debugf` on skip.
- `FR-client-config-4` (api_paths): an out-of-range `APIStyle` enum silently pins new-style + skips probing — optional erroring default / validate tag; at least document.
- `FR-codegen-resources-1` (resources): the ARCH-14 collision guard doesn't cover whitespace-keyed injected base fields — iterate map values (skip nil spacers) or add a comment. No real collision in the 9.5.21 catalog.
- `FR-settings-wrappers-1` (setting): dead/unused per-call `Meta` fields in Get/SetSetting response structs after O5 centralization — optionally drop or lowercase the tag.
- `FR-codegen-download-security-1` (download): ARCH-15 host pinning validates hop 0 only; redirects are followed unconstrained — document in codegen/CLAUDE.md trust-model; optional per-hop `CheckRedirect` re-validation.
- `FR-tests-and-docs-3` (testhelpers): two stale comments after the mutex fix (a non-existent `requestsSnapshot`; an outdated "unsynchronized slice" note).

**Verification (final review, in-workflow):** all 9 baseline checks green (build, lint 0, `unifi` + `-race`, `codegen -short` offline `GOPROXY=off`, `vet`, `codegen -race`, regen-idempotent, mock-idempotent). Coverage unifi **10.6%** / codegen **84.4%** (low unifi aggregate is the un-unit-exercised generated-CRUD surface; the hand-written core + codegen pipeline carry real coverage).

**1.11.0 effort: COMPLETE.** P0/P1/P2 all delivered; P3 + ARCH-12 out of scope by decision. Tree green, `-race`-clean, codegen idempotent. No code surgery warranted before shipping.
