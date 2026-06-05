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

**Implemented** — ARCH-22 (`Unwrap` on ValidationError/ServerError, deterministic `ValidationError.Error`, `errors.As` guard), ARCH-05 (`HandleError` keeps status/method/URL on empty/non-JSON bodies via capped read; 404 → `ErrNotFound` through `ServerError.Is`), ARCH-07 (`numberOrString` null → empty), ARCH-04 (CSRF token behind RWMutex; coarse per-request lock dropped — `net/http` is goroutine-safe; `UseLocking` now a deprecated no-op), ARCH-06 (TLS **secure-by-default**: `VerifySSL bool→*bool`, `nil` verifies, WARN on disable), TEST-09 (`apiStyleFromStatus` pure fn + probe through `c.http` + `ClientConfig.APIStyle` offline-construction seam), TEST-03/05 (wrapper coverage suite + `ErrNotFound` contract across `%w`), TEST-10 (moq `ClientMock` + offline constructor), TEST-06 (dead `customizeBaseType` block removed, regen zero-diff), TEST-04 (customize special-casing tests → 100%), **ARCH-08** (generated drift-proof settings registry via per-setting `init()` self-registration + `setting_registry.go`; reflection drift-guard `client_interface_test.go`; `SetSetting` exposed on `Client`; `DpiApp`/`DpiGroup` removed from generation), TEST-07 (`*http.Client` injected into download pipeline), TEST-02/ARCH-17 (bomb-cap + zip-slip + non-200 negative tests), TEST-08 (`go test -short ./codegen` now fully offline).

**Breaking changes** (see [breaking_changes.md](breaking_changes.md)): ARCH-06 `VerifySSL bool→*bool` + secure-by-default flip; ARCH-04 `UseLocking` no-op; ARCH-08 `SetSetting` added to `Client`, `DpiApp`/`DpiGroup` removed; TEST-07 `DownloadAndExtract` signature (package-internal).

**Review follow-ups deferred (minor/nit):** `UnblockUserByMAC`/`KickUserByMAC` still 0% (structurally identical to tested siblings); stamgr error-propagation branch partial; drift sentinel covers 3 keys (self-registration makes new settings auto-register regardless); `HandleError` 1 MiB cap branch untested; TEST-09 duplicate API-key-rejection string. Tracked for a later cleanup pass.

**Verification (final, in-workflow):** `go build ./...` ✓ · `golangci-lint run` 0 issues ✓ · `go test ./unifi/...` ✓ · `go test ./unifi/ -race` ✓ · `go test -short ./codegen/...` offline ✓ · `go vet ./codegen/...` ✓ · regen-reproducible (zero generated diff) ✓
