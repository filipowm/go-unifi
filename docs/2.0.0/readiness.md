# go-unifi 2.0.0 — Issue backlog (epic #117)

Each section below is a ready-to-file GitHub issue, derived from the 2.0.0 readiness audit.
Per-issue meta: **Milestone · Labels · Absorbs (audit item) · #117 task**. Acceptance criteria are checklists;
Evidence cites `file:line` so each issue is independently verifiable.

---

## Milestone: 2.0.0

### 1. `chore(module)!:` bump module path to `/v2`

**Milestone:** 2.0.0 · **Labels:** `chore`, `breaking` · **Absorbs:** M1 · **#117:** T7

**Problem.** `go.mod` declares `module github.com/filipowm/go-unifi` (no `/v2`). Go's semantic-import-versioning
requires a `/vN` path for major version ≥ 2, with no `+incompatible` escape hatch when a `go.mod` is present.
Tagging `v2.0.0` as-is makes `go get github.com/filipowm/go-unifi@v2.0.0` fail — the release is uninstallable.
This must land before any other consumer-facing change and before the tag.

**Acceptance criteria.**
- [ ] `go.mod` module line → `github.com/filipowm/go-unifi/v2`.
- [ ] Rewrite every internal import of the module path to `/v2` (today: `unifi → unifi/official`; also sweep any codegen-template-emitted import paths).
- [ ] `go build ./... && go test ./...` pass.
- [ ] All `go get` / `import` snippets in README + `docs/**` updated to `/v2` (coordinate with issue #2).
- [ ] Verify `go install github.com/filipowm/go-unifi/v2@<pseudo-version>` resolves from the proxy on a clean module cache.

**Evidence.** `go.mod:1` (no `/v2`), `go.mod:3` (`go 1.26.0`); `git tag` highest = `v1.11.0`; `unifi/client.go:14` imports `.../go-unifi/unifi/official`.

---

### 2. `docs:` make all quick-start examples compile + refresh front-door docs

**Milestone:** 2.0.0 · **Labels:** `docs` · **Absorbs:** M2, R4, N1, R9 · **#117:** T7

**Problem.** A new adopter's first copy-paste does not compile. Docs instantiate `ClientConfig` with a `BaseURL:`
field that has never existed (the field is `URL`) and with removed `Username`/`Password`/`RememberMe` fields and a
removed `c.Login()` call. They also carry stale claims (Go 1.16; "Any version after 5.12.35 is supported"; "fully
compatible with paultyng"), a dead godoc.org badge, and the primary `unifi` package has no package-doc, so its
pkg.go.dev landing is bare.

> `BaseURL`/`Username` were **never** valid fields (pre-existing doc bugs — `git log -S 'BaseURL string' --all`
> is empty; the field has always been `URL`, `Username` was always `User`). 2.0.0's auth removal makes the
> remaining `User`/`Password`/`Login()` lines non-compiling too.

**Acceptance criteria.**
- [ ] Replace `BaseURL:` → `URL:` and remove all `User`/`Username`/`Password`/`RememberMe`/`Login()` usage across `README.md`, `docs/getting_started.md`, `docs/configuration.md`, `docs/file_uploads.md`, `docs/advanced_topics.md` (line 86), `docs/migrating_from_upstream.md`.
- [ ] `docs/getting_started.md` Go prereq 1.16 → 1.26; remove the "supports username/password" framing.
- [ ] `README.md` "Any version after 5.12.35…" → accurate floor: API-key auth needs ~9.0.108+; the Official surface needs 10.1.78+. Clarify the two markers (`.unifi-version` 9.5.21 internal, `.unifi-version-official` 10.1.78). Add a 2.0.0 row to `docs/compatibility_matrix.md`.
- [ ] Soften the "fully compatible with `paultyng`" claim (2.0.0 is not source-compatible).
- [ ] Swap the godoc.org badge for the pkg.go.dev badge (`/v2` path).
- [ ] Add `unifi/doc.go` with a `// Package unifi` overview (API-key auth, `NewClient`, `Internal()`/`Official()`).

**Evidence.** `unifi/client.go:62-87` (only `URL`+`APIKey` auth fields), `:130` (`BaseURL()` is a getter method); `README.md:7,19,27,57-60,64,82-95`; `getting_started.md:8,46-100`; `configuration.md`, `file_uploads.md`, `advanced_topics.md:86`, `migrating_from_upstream.md`; `docs/compatibility_matrix.md` stops at 1.11.0.

---

### 3. `docs:` write the 1.x → 2.0 consumer migration guide

**Milestone:** 2.0.0 · **Labels:** `docs` · **Absorbs:** M3, R1 · **#117:** T7

**Problem.** T7 promised a migration guide; none exists. `docs/2.0.0/README.md` self-declares it is a workflow
process spec, "not a user migration guide." `breaking_changes.md` is a solid reference changelog but is unlinked
from README and the docs TOC, and README's only "migration guide" link points at the stale paultyng-upstream doc.
A 1.x consumer has no narrative upgrade path. Also: with rows #4/#5/#6 descoped to 3.0.0 (see #13), the changelog
must be reframed so it doesn't advertise 3 breaks that never land in 2.0.0.

**Acceptance criteria.**
- [ ] Create `docs/2.0.0/migration_guide.md` — task-oriented, with before/after for each landed break: `/v2` import path, API-key auth (#1), TLS verify-by-default (#2, silent), Go 1.26 (#3), error-handling changes (#9/#10/#11), removed/neutered fields (#12 UseLocking, #13 CSRF), and the opt-in Official surface via `Official()`.
- [ ] Relocate `breaking_changes.md` rows #4/#5/#6 into a clearly labelled "Planned for 3.0.0" section; reframe 2.0.0 as "10 breaking changes + an additive Official surface."
- [ ] Document the removal of `NewBareClient` here if issue #9 lands.
- [ ] Link the migration guide + `breaking_changes.md` from `README.md` (top, near install) and `docs/readme.md` (TOC).

**Evidence.** `docs/2.0.0/README.md:4-6`; `README.md:57-60`; `docs/readme.md` TOC (no `docs/2.0.0/*` entry); `breaking_changes.md` table rows #4/#5/#6 = **PENDING**.

---

### 4. `docs:` document the raw-call escape hatch + `Patch`

**Milestone:** 2.0.0 · **Labels:** `docs` · **Absorbs:** R10, R8, N3 · **#117:** T7

**Problem.** The low-level transport methods `Get/Post/Put/Patch/Delete/Do` on the public `Client` interface are
the supported escape hatch for endpoints the SDK doesn't model — the first thing a stranger needs when a resource
isn't generated. They're exposed but poorly documented: `docs/advanced_topics.md` omits `Patch` (added in 2.0.0)
and its example uses a leading-slash path (`/api/...`) that bypasses the `/proxy/network/api` base — wrong on
new-style controllers, the only style supported in 2.0.0. README never links it. Separately, `Patch` is an
interface addition (compile-break for third-party `Client` impls) that is undocumented in the changelog.

**Acceptance criteria.**
- [ ] `advanced_topics.md`: add `Patch` to the method list + a `Patch` example; replace the leading-slash example with the site-relative form (`s/<site>/rest/...`).
- [ ] State the path-resolution rule: no leading slash ⇒ prefixed with `/proxy/network/api`; leading slash / absolute ⇒ used as-is.
- [ ] Add a `Patch` provenance entry to `breaking_changes.md` (sibling to #7 `SetSetting` / #8 `*Context`), not a 14th headline row.
- [ ] Add a short group→resource note to README (Vouchers live under `Hotspot()`; `PatchPolicy` is Firewall-only). Do **not** add a `Vouchers()` accessor (breaks the tag-derived naming).
- [ ] Link `advanced_topics.md` from the README usage section.

**Evidence.** `unifi/client.generated.go:1179-1194` (`Delete/Do/Get/Patch/Post/Put`); `unifi/requests.go:334-387`; `docs/advanced_topics.md:11-16,31,50`; `unifi/api_paths.go:14,104`.

---

### 5. `fix(codegen)!:` `Device.QOSProfile` pointer (2.0.0)

**Milestone:** 2.0.0 · **Labels:** `bug`, `fix`, `breaking` · **Absorbs:** R5 · **#117:** new · **Refs:** PR #108

**Problem.** `Device.QOSProfile` is a value type with `,omitempty`, so it always serialises `"qos_profile":{}`,
which UDM SE rejects (`api.err.NotSupportQosConfig`) — breaking device port-override management. The clean fix in
a major is value → pointer so `omitempty` actually drops it. Companion to issue #12 (the `1.11.1` backport).

> Do **not** cherry-pick PR #108 — it edits `unifi/device.generated.go`, a `DO NOT EDIT` file the daily regen
> overwrites. Fix the codegen source.

**Acceptance criteria.**
- [ ] Add `QOSProfile: { fieldType: "*DeviceQOSProfile" }` under the existing `Device.fields` block in `codegen/internal/customizations.yml`.
- [ ] `go generate unifi/codegen.go`; golden diff shows only the `QOSProfile` type change.
- [ ] Confirm a nil pointer omits the key; update any internal wrapper/test that sets the profile to take its address.
- [ ] Documented in the migration guide (#3) as a field-type change.

**Evidence.** `unifi/device.generated.go:369` (`QOSProfile DeviceQOSProfile json:"qos_profile,omitempty"`), `:528` (`DeviceQOSProfile`); `codegen/internal/customizations.yml` `Device.fields` block `:514-541`.

---

### 6. `chore:` add `SECURITY.md` + fix the disclosure channel

**Milestone:** 2.0.0 · **Labels:** `docs`, `chore` · **Absorbs:** R6 · **#117:** new

**Problem.** 2.0.0's headline is security (TLS verify-by-default, API-key auth) yet there is no `SECURITY.md`, so
GitHub's "Report a vulnerability" UI is empty. The only disclosure instruction (`.github/CONTRIBUTING.md:75-76`)
literally reads "sent by email to `<>`" — an empty placeholder — with PGP commented out. No working private
vulnerability channel.

**Acceptance criteria.**
- [ ] Add `.github/SECURITY.md` (supported versions; private reporting via GitHub Security Advisories + a real email; response SLA).
- [ ] Fix or remove the dead `to <>.` line in `.github/CONTRIBUTING.md:75-76`.

**Evidence.** no `SECURITY.md` in repo; `.github/CONTRIBUTING.md:75-77`.

---

### 7. `ci:` re-enable the daily codegen cron

**Milestone:** 2.0.0 · **Labels:** `chore` · **Absorbs:** R7 · **#117:** new

**Problem.** README and the compatibility matrix advertise "daily automated updates," but the cron in
`.github/workflows/generate.yaml` is commented out (temporarily disabled for the #117 migration). The claim is
currently false.

**Acceptance criteria.**
- [ ] Re-enable `generate.yaml`'s `schedule:`/`cron:` once the codegen pipeline is verified stable post-migration; **or** drop the "daily updates" claim from `README.md:21,28` and `compatibility_matrix.md:18`.
- [ ] Confirm one scheduled run produces a clean (no-diff or intended-diff) regen PR.

**Evidence.** `.github/workflows/generate.yaml:4-5` (commented cron); `README.md:21,28`; `docs/compatibility_matrix.md:18`.

---

### 8. `refactor(codegen):` coverage matrix + completeness drift-guard

**Milestone:** 2.0.0 · **Labels:** `refactor` · **Absorbs:** R3 · **#117:** T2

**Problem.** T2's other parts are done (frozen legacy #124, 9.0.114 floor #126), but the curated OpenAPI-vs-legacy
coverage matrix and its drift-guard test never shipped, and no GitHub issue tracked it. This is the QA artifact
that proves first-batch coverage and flags future surface drift.

**Acceptance criteria.**
- [ ] Author the curated OpenAPI-vs-legacy coverage matrix (hand/LLM-curated).
- [ ] Add a ~30 LOC completeness drift-guard test that fails when the Official surface drifts from the matrix.
- [ ] Do **not** build an auto-matcher (that approach was spiked and rejected as net-negative).

**Evidence.** T2 in #117; #124/#126 closed; no matrix artifact or drift-guard in the tree.

---

### 9. `refactor!:` consolidate the client constructors (`SkipSystemInfo`)

**Milestone:** 2.0.0 · **Labels:** `refactor`, `breaking` · **Absorbs:** D3 · **#117:** new

**Problem.** Two public constructors exist: `NewClient` (validates, detects API style, then makes one extra
authenticated call to fetch system info + pre-warm the version cache) and `NewBareClient` (everything except that
call). Since username/password login was removed in 2.0.0 (#125), that extra call is now a single optional
round-trip whose result is already fetched lazily on first `Version()` use — so the eager call's only remaining
value is fail-fast that the API key works + a log line. A major is the only cheap window to consolidate (removing
a public function later needs a v3).

**Acceptance criteria.**
- [ ] Add `SkipSystemInfo bool` to `ClientConfig`; zero value (`false`) preserves today's eager `NewClient` behavior. Name matches existing opt-out bools (`SkipVerifySSL`, `DisableOfficialAPI`).
- [ ] `NewClient` skips `GetSystemInformation()` when `SkipSystemInfo: true`.
- [ ] Remove public `NewBareClient`; repoint in-package tests to the unexported `newBareClient`.
- [ ] Document the removal in the migration guide (#3).

**Evidence.** `unifi/client.go:351-365` (`NewClient`) vs `:369-371` (`NewBareClient`) vs `:373-401` (`newBareClient`); eager fetch block `:356-363`; lazy version cache `:181-214`; existing opt-out bools `:70,86`; usages `unifi/concurrency_test.go:48`, `unifi/client_test.go:289,341`.

---

### 10. `docs(codegen):` replace placeholder godoc in the Official generator

**Milestone:** 2.0.0 · **Labels:** `docs` · **Absorbs:** N4 · **#117:** new

**Problem.** ~332 Official model types carry oapi-codegen's zero-information `"X defines model for X"` placeholder
doc comments on pkg.go.dev (field-level docs are fine where the spec supplied descriptions). Cosmetic but pervasive
for a release that introduces the dual surface.

**Acceptance criteria.**
- [ ] Add a generator post-pass that rewrites **only** description-less placeholder type docs, leaving spec-supplied docs intact.
- [ ] Must not regress the "exported type needs a comment" lint (so don't simply delete the placeholders).

**Evidence.** N4 audit finding; the Official generator emits one placeholder doc per model in `unifi/official/*.generated.go`.

---

### 11. `chore:` cut & tag `v2.0.0`

**Milestone:** 2.0.0 · **Labels:** `chore` · **Absorbs:** M4, N2 · **#117:** T7

**Problem.** 2.0.0 is untagged and unreleased, #117 is open, and no `feat/2.0.0 → main` PR has been opened.
`.goreleaser.yaml` uses `changelog: use: github` (a raw commit dump), inadequate for a 13-break major.
This is the terminal gate — dependency-gated on all the other 2.0.0 issues + the 3.0.0 descope being recorded.

**Acceptance criteria.**
- [ ] All other 2.0.0 issues merged; `go build ./... && go test ./...` green on `feat/2.0.0`.
- [ ] Open `feat/2.0.0 → main` PR; CI green; merge.
- [ ] Tag `v2.0.0`; verify `go install github.com/filipowm/go-unifi/v2@v2.0.0` resolves from the proxy.
- [ ] Publish a curated GitHub Release body wired to `breaking_changes.md` (goreleaser `release.header`/`footer` or hand-authored).
- [ ] Close #117; bump README badges; notify [terraform-provider-unifi](https://github.com/filipowm/terraform-provider-unifi) of the auth/type break.

**Evidence.** `git tag` highest = `v1.11.0`; `gh pr list --base main --head feat/2.0.0` → `[]`; `.goreleaser.yaml` `changelog: use: github`.

---

## Milestone: 1.11.1 (new — patch on `main`)

### 12. `fix(codegen):` `Device.QOSProfile` pointer (1.11.x backport)

**Milestone:** 1.11.1 · **Labels:** `bug`, `fix` · **Absorbs:** R5 (backport) · **#117:** n/a · **Refs:** PR #108

**Problem.** Same bug as issue #5, but it must also be fixed on the released 1.x line (`main`): the value-type
`Device.QOSProfile` with `,omitempty` always emits `"qos_profile":{}`, rejected by UDM SE
(`api.err.NotSupportQosConfig`). Ship it as `1.11.1` so 1.x users aren't stuck until 2.0.0.

**Acceptance criteria.**
- [ ] On `main`: add the `QOSProfile` `fieldType: "*DeviceQOSProfile"` override to the codegen customizations. **Verify the path on `main`** — it predates the `codegen/internal` extraction (#135), so the file is likely `codegen/customizations.yml`, not `codegen/internal/customizations.yml`.
- [ ] Regenerate; golden diff = only the `QOSProfile` type change. Do **not** cherry-pick PR #108 (edits a generated file).
- [ ] **Semver decision:** value → `*pointer` is a minor source-compat change (callers add `&`). Acceptable as a bugfix in a patch; if strict patch compatibility is required, use a custom `MarshalJSON` that omits the empty profile instead. Pick one and note it in the release.
- [ ] Create the `1.11.1` milestone; tag & release `1.11.1` from `main`.

**Evidence.** Same bug as #5; PR #108 targets `main`. Confirm `main`'s codegen layout before editing.

---

## Milestone: 3.0.0 (descoped from 2.0.0)

### 13. `feat(codegen)!:` retarget internal resources to OpenAPI shapes (rows #4/#5/#6)

**Milestone:** 3.0.0 · **Labels:** `feat`, `breaking` · **Absorbs:** R1 (descoped) · **#117:** T3

**Problem.** Breaking-change rows #4 (OpenAPI-shaped structs), #5 (field renames), and #6 (new `integration/v1`
`APIStyle` for internal resources) are listed in the 2.0.0 changelog but were never implemented — `Official()` is
purely additive and the regex type-inference path is still live for the frozen legacy generator. Decision: descope
to 3.0.0, where the default surface flips to Official.

**Acceptance criteria.**
- [ ] Retarget internal resources to OpenAPI-derived shapes; retire `fieldInfoFromValidation`/`numericFieldInfo`/`normalizeValidation` where replaced.
- [ ] Flip the default client surface from Internal to Official (the 3.0.0 default-flip).
- [ ] Migration notes covering the field renames and shape changes.

**Evidence.** `breaking_changes.md` rows #4/#5/#6 = PENDING; README documents the intended 3.0.0 flip.

---

### 14. `feat!:` dual-version resource shape selection (T4)

**Milestone:** 3.0.0 · **Labels:** `feat`, `breaking` · **Absorbs:** R2 (descoped) · **#117:** T4

**Problem.** T4 — carry both legacy + OpenAPI shapes per divergent resource and runtime-select by detected Network
API version (the "so upgrades just work" promise) — has no implementation; the only version logic is a whole-API
10.1.78 capability gate. The epic's open question (which resources actually diverge between 9.0.114 and 10.1.78+)
is unresolved, so this is correctly deferred until that analysis exists.

**Acceptance criteria.**
- [ ] Determine the divergence set (resources whose shape differs across 9.0.114 → 10.1.78+).
- [ ] Design a runtime shape-selection seam beyond the whole-API capability gate.
- [ ] Implement for DNS policies first (the epic's named example), with version-driven selection.

**Evidence.** `unifi/official_surface.go` (whole-API gate only); DNS exposed as two caller-chosen surfaces (Internal `DNSRecord` / Official `DNSPolicy`), not an auto-selected shape.

---

## Recorded decision (not an issue)

**Codegen input layout stays root-owned.** `codegen/openapi/` and `codegen/v*/` remain root-level siblings of
`codegen/internal/` and `codegen/official/`. The root orchestrator downloads/owns all generator inputs; the
engines are pure, path-injected consumers (one `-version-base-dir` knob). Moving inputs under their engines
(openapi → `codegen/official/openapi`, or `v*` → `codegen/internal`) was considered and **rejected** — it would
make the root write across package/module boundaries and dump the internal download cache inside a Go package, for
no offsetting benefit. Recorded so it isn't re-proposed.
