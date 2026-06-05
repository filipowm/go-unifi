# go-unifi 1.11.0 implementation — STATUS / resume point

**Purpose:** hand-off doc so a new session can continue the workflow-driven implementation of the
[1.11.0 review](summary.md). Read this first, then [plan.md](plan.md), [architect-review.md](architect-review.md),
[test-review.md](test-review.md), [breaking_changes.md](breaking_changes.md), [implementation-log.md](implementation-log.md).

**Last updated:** after Wave 1 completed and was committed. **Next action: Wave 2.**

---

## 1. Mission & scope

Implement the review findings as a phased, workflow-orchestrated effort.

- **In scope:** all **P0, P1, P2** findings.
- **Discarded:** all **P3** findings (ARCH-23..31, TEST-17..20). Do NOT implement them.
- **Skipped (user decision):** **ARCH-12** (the L-effort codegen-emit-wrappers refactor). There is **no Wave 3**.
- Net: Wave 0 (P0) ✅ done · Wave 1 (P1) ✅ done · **Wave 2 (P2) ← do this next** · then a final whole-codebase review.

## 2. Process contract (user's binding requirements — follow exactly)

1. **Everything runs through the `Workflow` tool** (ultracode is on). Use multiple phases + subagents.
2. **Subagents within a wave must not overlap** (see §5 constraints — partition by Go package / file ownership).
3. **Build/test/lint checks AND the architect+test-lead review must happen INSIDE the workflow**, not in the main loop. Pattern per wave: `Implement → Verify (fix loop) → Review (architect ‖ test-lead) → Remediate (gated on blocker/major) → re-Verify`.
4. After every wave **all tests + lint must pass**.
5. **API breaking changes** go in [breaking_changes.md](breaking_changes.md).
6. **Pause after each wave** and summarize; wait for the user's go-ahead before the next wave.
7. **Commits:** feature branch `chore/review-1.11.0`; **separate commit per cohesive, test-passing change** (per finding/lane), conventional-commit style with the finding IDs, `!` + `BREAKING CHANGE:` footer for breaks. Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
8. After ALL waves: a **final thorough whole-codebase review by software architect + test lead**.

## 3. Environment & commands (CRITICAL)

- **Branch:** `chore/review-1.11.0` (off `main`). Work here.
- **Go PATH gotcha:** an old asdf Go 1.20 shadows Homebrew Go 1.26. **Prepend to every go/gofmt/golangci-lint command:**
  `export PATH="/opt/homebrew/opt/go/bin:$PATH"` (gives go 1.26.4). Without it, `go.mod`'s `go 1.26` fails to parse.
- **`.unifi-version`:** shows `10.3.58` and is a **pre-existing uncommitted user change — NEVER stage/commit/restore it**. The codegen pin is **9.5.21** (HEAD's value; the complete cached field defs live in `codegen/v9.5.21/`).
- **Offline regeneration (zero-network, reproducible, zero-diff when no source changed):**
  `cd unifi && go run ../codegen -version-base-dir=../codegen 9.5.21`
  The generator writes into `unifi/` from CWD=`unifi`, uses the cached `codegen/v9.5.21`, and rewrites `.unifi-version` to `9.5.21` as a side effect — **restore it afterward** (`cp /tmp/uv.bak .unifi-version`, or just leave it for the orchestrator; never commit it). NEVER use the `go generate`/`latest` form — it downloads the newest controller and pollutes the tree.
- **Authoritative verify set** (what the in-workflow Verify agent should run; ignore `.unifi-version` diffs):
  `go build ./...` · `golangci-lint run` (expect 0 issues) · `go test ./unifi/...` · `go test ./unifi/ -race` · `go test -short ./codegen/...` (offline since TEST-08) · `go vet ./codegen/...` · regen-reproducibility (regen then `git diff --stat -- 'unifi/*.generated.go'` must be EMPTY).
- **Never hand-edit `*.generated.go`.** Change generated output via `codegen/customizations.yml`, `codegen/*.tmpl`, or `codegen/*.go`, then regen. Behavior changes go in hand-written `unifi/<resource>.go` siblings.
- **Conventions:** tabs; `gofmt`/gofumpt/goimports/gci (golangci-lint enforces); lines <200; `context.Context` first; wrap errors `%w`; testify table tests `map[string]struct{}` with `t.Parallel()` on outer+subtests; `httptest` round-trips; internal tests `package unifi`/`package main`, public-API `package unifi_test`. See `.claude/rules/`.

## 4. ⚠️ Lessons learned (apply to Wave 2)

- **Forbid agents from running ANY git command** (`git add/commit/rm/reset/stash`). In Wave 1 a Phase-A agent committed part of the wave *incoherently* (generated files without their source), which had to be undone and re-committed. **The orchestrator (main loop) owns ALL git.** Put this rule in every implement/remediate agent prompt.
- **`new(false)` is valid Go 1.26** (`new(expr)` landed in 1.26; the lib requires go 1.26). The `VerifySSL *bool` docs using `new(false)` are correct — don't "fix" them.
- Reviewers can miss things: verify breaking-change docs cover **every** lane's breaks (the W1 remediation initially missed ARCH-08's `SetSetting`/`Dpi` breaks — caught and added in the main loop).
- An agent may leave `.unifi-version` rewritten to `9.5.21` after regen — restore the user's `10.3.58` before committing and never include it in a commit.

## 5. Orchestration model & hard constraints

- **Codegen regeneration writes into the `unifi` package.** Therefore a regen-agent and any unifi-agent **cannot run concurrently** (both touch/compile/test the unifi package). And two agents cannot run the **same Go package's** tests in parallel.
- Practical schedule per wave:
  - **Phase A (serialized, alone):** all codegen-source changes that **regenerate** (one agent, or a short sequence). Finalize the generated tree + interface here.
  - **Phase B (parallel, disjoint packages):** `unifi` stream (sequential agents within) ‖ `codegen` non-regen stream (edits `codegen/*.go`/`*_test.go`, NO regen). These are different Go packages → safe in parallel. Scope each agent's build/test to **its own package** (`./unifi/...` or `./codegen/...`), never `./...`, while the other stream is live.
  - **Phase C:** Verify (full `./...`, with bounded fix loop) → Review (architect ‖ test-lead, read-only) → Remediate (gated on blocker/major) → re-Verify.
- Worktrees were intentionally NOT used (merge complexity); single shared tree + serialization instead.
- Structured-output schemas (REPORT / VERIFY / REVIEW / REMEDIATE) — reuse the shapes from the Wave 1 script (see `.claude/.../workflows/scripts/wave1-p1-hardening-*.js` in the session dir if resuming same session; otherwise re-derive from this doc).

## 6. Progress

### Wave 0 — P0 hotfixes ✅ (commits `44fa888`, `1c99505`, `d2c4bef`, `da6a959`, `402f30b`)
ARCH-01 (Version deadlock), ARCH-02 (permissive `booleanishString`), ARCH-03 (3 missing setting factories), TEST-01 (codegen template golden tests). No breaking changes.

### Wave 1 — P1 hardening ✅ (commits `ab890eb`, `392de62`, `0dee64a`, `ba185b8`, `1329e49`, `a335ea6`, `fe8ae4a`, `be9ebc0`; HEAD=`be9ebc0`)
ARCH-04/05/06/07/08/22 + TEST-02/03/04/05/06/07/08/09/10 + ARCH-17. See [implementation-log.md](implementation-log.md) for detail.
- **Breaking (documented):** `VerifySSL bool→*bool` + secure-by-default; `UseLocking` no-op; `SetSetting` added to `Client`; `DpiApp`/`DpiGroup` removed; (internal) `DownloadAndExtract` signature.
- Verify all-green; architect + test-lead reviewed (no regressions); 1 major remediated. Deferred minors logged in implementation-log.md.
- New seams a Wave-2 author should reuse: **`ClientConfig.APIStyle` offline-construction override**, the wrapper **`testhelpers_test.go`** + `newTestClient`/RoundTripper pattern, the moq **`ClientMock`** (`unifi/client_mock.generated.go`, regen via `//go:generate` in `unifi/mock.go`), the injectable `*http.Client` in `codegen` download pipeline, and `go test -short ./codegen` offline gating.

## 7. Wave 2 — TODO (P2 quality & codegen robustness, ~17 findings)

Findings (full text in [architect-review.md](architect-review.md)/[test-review.md](test-review.md)):
**ARCH-09** constructor mutates caller `ClientConfig` (URL/UserAgent) · **ARCH-10** Meta `rc:error` (200-with-error) unchecked → centralize in `handleResponse` per **O5** + resolve `user.go` TODO · **ARCH-11** `ContentLength==0` short-circuit (decode on body, treat `io.EOF` as empty) · **ARCH-13** v1 template returns `ErrNotFound` from successful create/update → distinct error; doc `ErrNotFound` is get/list-only · **ARCH-14** codegen field-drop/CamelCase-collision silently lost → warn + strict mode + deterministic ordering + golden type-diff · **ARCH-15** download pipeline timeouts/cancellation + integrity/host validation · **ARCH-16** atomic extraction (temp dir+rename or `.complete` sentinel) · **ARCH-18** interceptor dedup by concrete type; `AddInterceptor(ClientInterceptor)` not `*ClientInterceptor` · **ARCH-19** real query-param support / reject `?` in id-suffixed `resourcePath` (DescribedFeature) · **ARCH-20** factor shared template header/defines/struct+unmarshal into a common partial · **ARCH-21** apply customizations exactly once (drop double-apply); split resource-level overrides · **TEST-11** unmarshaler branches + `Logout`/`Version`/`Meta.error()` tests · **TEST-12** extract pure `buildMultipartUpload` + upload tests · **TEST-13** thread codegen logger / `newValidator` extra validators / immutable API-path sets · **TEST-14** single shared `unifi/testhelpers_test.go` (consolidate; partly seeded in W1) · **TEST-15** ctx-first variants for `Version`/`Login`/`Logout`/`GetSystemInformation` · **TEST-16** `utils_test.go` for `ensurePath`/`findProjectRoot`/`findCodegenDir` + inject codegen v2 base dir.

**Decisions already binding:** O5 (centralize Meta `rc:error` in `handleResponse`, gated on a meta block present). ARCH-13: per O3, the `Get/Delete` reqBody foot-gun is documented not removed (that was P3/ARCH-23, discarded) — but ARCH-13's create/update `ErrNotFound` fix IS in scope.

**Likely Wave 2 breaking changes** (→ breaking_changes.md): **ARCH-18** (`AddInterceptor` signature `*ClientInterceptor`→`ClientInterceptor`), **ARCH-13** (create/update no longer return `ErrNotFound`), possibly **ARCH-09**/**ARCH-11** behavior, **TEST-15** (new `*Context` method variants / interface additions).

**Suggested Wave-2 lane partition** (regen-touching findings → Phase A serialized; rest → Phase B parallel disjoint packages):
- **Phase A (codegen regen):** ARCH-13 (api.go.tmpl create/update), ARCH-19 (query-param / resourcePath, customizations.yml + apiv2 template), ARCH-20 + ARCH-21 (template de-dup + apply-once in `generator.go`/`resources.go`/`customize.go`), ARCH-14 (codegen field-drop warnings + golden type-diff in `resources.go`). All change templates/codegen + regen → one serialized stream; expect intended generated diffs (regen-reproducible must still hold).
- **Phase B unifi stream:** ARCH-09 (client.go non-mutation), ARCH-10 (requests.go `handleResponse` Meta check + user.go TODO), ARCH-11 (requests.go decode-on-body), ARCH-18 (client.go/interceptors.go dedup + `AddInterceptor` sig), TEST-15 (ctx-first variants — may need customizations.yml regen if interface methods change → coordinate with Phase A), TEST-11 (json/Logout/Version/Meta tests), TEST-12 (`buildMultipartUpload` extract + tests), TEST-14 (consolidate test helpers).
- **Phase B codegen stream (no regen):** TEST-13 (logger/validator/api-path-set testability), TEST-16 (`utils_test.go` + inject codegen v2 base dir), and ARCH-15/ARCH-16 (download.go timeouts + atomic extraction — these are `codegen` runtime behavior, edit `download.go`/`utils.go` + tests).
- ⚠️ TEST-15 changing the `Client` interface (adding `*Context` methods) requires customizations.yml + regen → put the interface/codegen part in **Phase A**, the impl/tests in Phase B. Sequence carefully (Phase A finalizes interface before Phase B unifi compiles against it).

## 8. Still pending after Wave 2

- **Final whole-codebase review** by software architect + test lead (per process contract §2.8), then a closing summary. Document any final breaking-change deltas.
- Optionally revisit the deferred Wave 1 minor follow-ups (see implementation-log.md) if the user wants them.

## 9. Key file references

- Review source of truth: `docs/review/1.11.0/{summary,plan,architect-review,test-review}.md`
- Living docs to keep updating: `docs/review/1.11.0/{breaking_changes,implementation-log,status}.md`
- Hand-written client core: `unifi/{client,requests,interceptors,unifi_errors,validation,json,api_paths,setting,setting_registry}.go`
- Codegen: `codegen/{resources,customize,generator,clients,download,utils,version,main}.go`, `codegen/{api.go.tmpl,apiv2.go.tmpl,customizations.yml}`
- Conventions: `CLAUDE.md`, `codegen/CLAUDE.md`, `.claude/rules/{go-conventions,testing}.md`
