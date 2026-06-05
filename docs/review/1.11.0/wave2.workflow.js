export const meta = {
  name: 'wave2-p2-quality',
  description: 'Wave 2: P2 quality & codegen robustness (~17 findings). Phase A codegen regen (ARCH-13/14/19/20/21 + TEST-15 iface), Phase B unifi-core || codegen-runtime, Phase C verify+review+remediate.',
  phases: [
    { title: 'Codegen-regen', detail: 'ARCH-20/21 (template de-dup + apply-once), ARCH-14 (drop/collision warns+golden), ARCH-13 (no ErrNotFound on create/update), ARCH-19 (query-param), TEST-15 (ctx-variant iface + mock regen) — serialized' },
    { title: 'Quality', detail: 'unifi: ARCH-09/10/11/18 + TEST-11/12/13u/14/15-impl || codegen-runtime: ARCH-15/16 + TEST-13c/16' },
    { title: 'Verify', detail: 'authoritative build/test/lint/-race/regen-reproducible with fix loop' },
    { title: 'Review', detail: 'architect + test-lead adversarial read-only review' },
    { title: 'Remediate', detail: 'fix blocker/major findings, re-verify' },
  ],
}

const ROOT = '/Users/filipowm/Documents/dev/workspaces/unifi/go-unifi'
const PATHFIX = 'export PATH="/opt/homebrew/opt/go/bin:$PATH"'
const MOQ = '/Users/filipowm/go/bin/moq' // pre-installed v0.7.1; regenerates client_mock.generated.go byte-identical OFFLINE
const REGEN = `cd ${ROOT}/unifi && ${PATHFIX} && go run ../codegen -version-base-dir=../codegen 9.5.21   # offline regen against pinned 9.5.21 cache`
const MOCKREGEN = `cd ${ROOT}/unifi && ${PATHFIX} && ${MOQ} -out client_mock.generated.go . Client   # offline mock regen (REQUIRED whenever the Client interface changes)`

const COMMON = `Repo root: ${ROOT}. go-unifi = Go client for the UniFi controller API. Branch chore/review-1.11.0.
HARD RULES (CLAUDE.md, codegen/CLAUDE.md, .claude/rules/):
- NEVER hand-edit *.generated.go. To change generated output, edit codegen/customizations.yml, codegen/*.tmpl templates, or codegen/*.go, THEN regenerate: \`${REGEN}\`. The generator writes into unifi/ from CWD=unifi using the cached codegen/v9.5.21 (offline, no network). After regen, .unifi-version gets rewritten to 9.5.21 — IGNORE that file: do NOT restore, stage, or commit it (the orchestrator owns it).
- WHENEVER the generated Client interface changes (unifi/client.generated.go), you MUST also regenerate the moq mock OFFLINE: \`${MOCKREGEN}\`. moq is the pre-installed binary at ${MOQ} (do NOT use \`go run ...moq@latest\` — it needs network and will fail). client_mock.generated.go MUST stay in sync or the build breaks on \`var _ Client = &ClientMock{}\`.
- FORBIDDEN: you must NEVER run ANY git command (no git add/commit/rm/reset/stash/restore/checkout). The orchestrator (main loop) owns ALL git. To delete a generated file, use the regen mechanism (excludeGeneration) — never \`git rm\`. To remove a hand file, use \`rm\` only if explicitly instructed, otherwise leave it.
- Go uses TABS; run \`${PATHFIX}; gofmt -w <files>\` on every .go file you change. Lines <200 cols. Methods take context.Context first. Wrap errors with %w.
- Tests: testify assert/require; table-driven map[string]struct{} with t.Run + t.Parallel() on outer AND subtests; net/http/httptest for round-trips. Internal tests \`package unifi\`/\`package main\`; public-API tests \`package unifi_test\`. Mirror existing style (unifi/requests_test.go, unifi/testhelpers_test.go, unifi/interceptors_test.go, codegen/resources_test.go, codegen/download_test.go, codegen/utils_test.go). REUSE the shared unifi/testhelpers_test.go (newControllerServer, controllerServer.client(), apiV1Path/apiV2) — do NOT spin up ad-hoc servers when that helper fits.
- ALL go/gofmt/golangci-lint/moq commands MUST prepend: ${PATHFIX}
- TDD: failing test first, then fix, then green. Do NOT weaken assertions to pass.
- Full problem statements: docs/review/1.11.0/architect-review.md and test-review.md (by ID); decisions O1-O5 + §0 in plan.md are BINDING (esp. O5: centralize Meta rc:error in handleResponse gated on a meta block present; O3: Get/Delete reqBody foot-gun is documented NOT removed).
Your final message MUST be the structured object (data for the orchestrator), nothing else.`

const REPORT = {
  type: 'object', additionalProperties: false,
  required: ['lane', 'overallStatus', 'findings', 'verifyOutput', 'breakingChanges', 'blockers'],
  properties: {
    lane: { type: 'string' },
    overallStatus: { type: 'string', enum: ['all-green', 'partial', 'blocked'] },
    findings: { type: 'array', items: { type: 'object', additionalProperties: false,
      required: ['id', 'status', 'filesChanged', 'notes'],
      properties: {
        id: { type: 'string' }, status: { type: 'string', enum: ['done', 'partial', 'skipped', 'blocked'] },
        filesChanged: { type: 'array', items: { type: 'string' } },
        testsAdded: { type: 'array', items: { type: 'string' } },
        notes: { type: 'string' },
      } } },
    breakingChanges: { type: 'array', items: { type: 'object', additionalProperties: false,
      required: ['what', 'migration'], properties: { id: { type: 'string' }, what: { type: 'string' }, migration: { type: 'string' } } } },
    verifyOutput: { type: 'string' },
    blockers: { type: 'string' },
  },
}
const VERIFY_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['allGreen', 'checks', 'summary'],
  properties: {
    allGreen: { type: 'boolean' },
    checks: { type: 'array', items: { type: 'object', additionalProperties: false,
      required: ['name', 'passed', 'detail'], properties: { name: { type: 'string' }, passed: { type: 'boolean' }, detail: { type: 'string' } } } },
    summary: { type: 'string' },
  },
}
const REVIEW_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['persona', 'overallAssessment', 'regressionsFound', 'findings'],
  properties: {
    persona: { type: 'string' }, overallAssessment: { type: 'string' }, regressionsFound: { type: 'boolean' },
    findings: { type: 'array', items: { type: 'object', additionalProperties: false,
      required: ['severity', 'area', 'description', 'suggestion'],
      properties: { severity: { type: 'string', enum: ['blocker', 'major', 'minor', 'nit'] }, relatesTo: { type: 'string' }, area: { type: 'string' }, file: { type: 'string' }, description: { type: 'string' }, suggestion: { type: 'string' } } } },
  },
}
const REMEDIATE_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['applied', 'skipped', 'filesChanged', 'notes'],
  properties: {
    applied: { type: 'array', items: { type: 'object', additionalProperties: false, required: ['finding', 'action'], properties: { finding: { type: 'string' }, action: { type: 'string' } } } },
    skipped: { type: 'array', items: { type: 'object', additionalProperties: false, required: ['finding', 'reason'], properties: { finding: { type: 'string' }, reason: { type: 'string' } } } },
    filesChanged: { type: 'array', items: { type: 'string' } }, notes: { type: 'string' },
  },
}

const VERIFY_PROMPT = `You are the Wave 2 verification gate. Repo root: ${ROOT}. Do NOT edit files. Do NOT run git. Prepend ${PATHFIX}. Run from repo root, capture output tails. IGNORE any .unifi-version diff.
1. go build ./...
2. golangci-lint run   (expect 0 issues)
3. go test ./unifi/...
4. go test ./unifi/ -race
5. go test -short ./codegen/...   (MUST run fully OFFLINE)
6. go vet ./codegen/...
7. regen-reproducible (IDEMPOTENCY test — do NOT compare against git HEAD; Wave 2 intentionally changes generated output and it is NOT yet committed). Steps: (i) H1=\`find ${ROOT}/unifi -name '*.generated.go' | sort | xargs shasum | shasum\`; (ii) run \`${REGEN}\`; (iii) H2=same shasum command. H1 MUST EQUAL H2 — i.e. regenerating an already-correct tree changes NOTHING (this is the true reproducibility property). Record as check "regen-reproducible" passed iff H1==H2. (Record both hashes in detail.)
8. mock-in-sync (IDEMPOTENCY test): (i) M1=\`shasum ${ROOT}/unifi/client_mock.generated.go\`; (ii) run \`${MOCKREGEN}\`; (iii) M2=same. M1 MUST EQUAL M2 (regenerating the mock changes nothing). Record as check "mock-in-sync" passed iff M1==M2.
NOTE: after running regen the .unifi-version file may be rewritten to 9.5.21 — IGNORE it entirely (do not restore/commit; the orchestrator owns it). The shasum checks above only cover *.generated.go, never .unifi-version.
allGreen = every check passed. Return the VERIFY structured object only.`

const FIX_PROMPT = (v) => `Wave 2 verification FAILED. Repo root: ${ROOT}. Prepend ${PATHFIX}. Do NOT run git (except the read-only verify diffs). Failing checks:\n${JSON.stringify(v.checks.filter(c => !c.passed), null, 2)}\nFix with MINIMAL correct changes preserving Wave 2 intent. NEVER hand-edit *.generated.go (fix at codegen source + regen; if the interface changed, also \`${MOCKREGEN}\`). Keep tabs+gofmt. Do NOT weaken tests. Re-run the failing checks to confirm. Return the REMEDIATE object {applied, skipped, filesChanged, notes}.`

// =================== PHASE A: codegen regen (serialized, alone) ===================
// All template/codegen-source changes that regenerate the unifi tree. Sequenced so each agent
// starts from a green, regenerated tree. Order: refactors that must be ZERO-DIFF first
// (ARCH-20/21), then ARCH-14 (warns+golden, zero functional diff), then the INTENDED-diff
// changes (ARCH-13, ARCH-19), then TEST-15 (interface grows -> regen + mock regen).
phase('Codegen-regen')

const a1 = await agent(`${COMMON}

LANE A1 cg:arch20-21 (codegen templates + customize/generator; regen REQUIRED, MUST be zero functional diff). Read architect-review.md ARCH-20 and ARCH-21.

ARCH-20 (factor shared template partial): codegen/api.go.tmpl and codegen/apiv2.go.tmpl DUPLICATE their top block — the package header + imports (~lines 20-36 in each) and the shared {{ define "field" }} / {{ define "field-customUnmarshalType" }} / {{ define "typecast" }} blocks (~lines 1-19). Factor the truly-identical shared pieces into a common partial template file (e.g. codegen/common.tmpl or codegen/_header.partial.tmpl) and parse it into BOTH templates. Find where templates are parsed (codegen/generator.go / clients.go — look for template.New/.Parse/.ParseFS/embed). Add the partial to that parse set so both api and apiv2 templates can invoke the shared defines. Do NOT change the RENDERED output — this is a pure refactor. Verify by regen + zero-diff.

ARCH-21 (apply customizations exactly once / split resource-level vs field-level / drop dead double-apply): In codegen/customize.go, ResourceCustomization.ApplyTo + applyCurrentProcessor compose FieldProcessors such that customization processors can run in a confusing order. Audit for any DOUBLE-APPLICATION of the same customization (the same field override applied twice). If a customization is genuinely applied twice, fix so it applies EXACTLY once; cleanly SEPARATE resource-level overrides (resourcePath, excludeFunctions) from field-level overrides (the per-field FieldProcessor). Remove any dead/duplicated apply path. PRESERVE existing generated output — the existing per-resource behavior (SettingGlobalAp 6E->SixE, SettingUsg *Timeout->emptyStringInt but NOT ArpCacheTimeout, SettingMgmt XSshKeys nested) MUST remain identical after regen. If after careful audit there is NO real double-apply (the current composition is intentional/correct), then make this a SAFETY change: add a focused unit test in codegen/customize_test.go proving each customization applies exactly once (e.g. a FieldProcessor that counts invocations per field), and document the ordering contract in a comment — report ARCH-21 as 'done' with notes explaining the finding was a naming/clarity issue not a real double-apply. Do NOT invent a behavioral change that alters generated output.

CRITICAL: after your changes, run \`${REGEN}\` then \`git -C ${ROOT} diff --stat -- 'unifi/*.generated.go'\` — it MUST be EMPTY (you changed nothing the renderer emits). If it is non-empty, your refactor changed output — investigate and fix until zero-diff. (The read-only \`git diff --stat\` is the ONLY git command you may run.)
Verify: ${PATHFIX}; go build ./... ; go vet ./codegen/... ; go test -short ./codegen/... ; go test ./unifi/... ; regen zero-diff. gofmt -w changed .go files. Set lane="cg:arch20-21". Report ARCH-20, ARCH-21.`,
  { label: 'cg:arch20-21', phase: 'Codegen-regen', schema: REPORT })

const a2 = await agent(`${COMMON}

LANE A2 cg:arch14 (codegen field-drop/collision robustness; regen MUST be zero functional diff for the current 9.5.21 catalog). Read architect-review.md ARCH-14. Start from the current (already-refactored by A1) tree.

Implement ALL:
(1) WARN on dropped field: codegen/resources.go processFields currently swallows per-field inference errors with a bare \`continue\`. Change it to log.Warnf (use the codegen logger — see TEST-13c; if a logger is being threaded, use it, else the package \`log\`) with resource name + JSON key + raw validation when fieldInfoFromValidation errors. Same for fieldInfoFromMap: skip ONLY the failing nested child (mirror processFields) rather than discarding the whole nested struct + siblings.
(2) WARN on CamelCase collision: before assigning t.Fields[fieldInfo.FieldName], check whether that FieldName already exists with a DIFFERENT JSONName; if so log.Warnf a collision warning (resource + both JSON keys + the colliding Go name). Do not silently overwrite without warning.
(3) DETERMINISTIC ordering: sort the JSON field keys before processing (sort the map keys into a slice, range that) in BOTH processFields and the nested fieldInfoFromMap, so collision resolution + output is reproducible run-to-run.
(4) STRICT MODE: add a strict flag (an env var like UNIFI_CODEGEN_STRICT=1, or a field on the customizer/options) that makes a dropped or colliding field a hard ERROR (fail the generation) instead of a warning. Wire it so CI can opt in; default OFF (warn only) to preserve current behavior.
(5) GOLDEN TYPE-DIFF TEST: add a golden/snapshot test (codegen/resources_test.go or a new codegen/resources_golden_test.go — check if one already exists and extend it) that renders the generated Go types for a representative resource (or the full set) and diffs against a committed golden, so a controller-version regex change that flips a field type (int<->float64<->string) or drops a field is caught in CI. Use the offline 9.5.21 cache as the fixture source. Provide an update mechanism (e.g. -update flag or UPDATE_GOLDEN env) and commit the initial golden.

CRITICAL: this must NOT change generated output for the current catalog — regen + \`git -C ${ROOT} diff --stat -- 'unifi/*.generated.go'\` MUST be EMPTY. The sort changes resolution order but the current catalog must have no actual collisions that change the winner (verify: if sorting changes output, that itself reveals a real latent collision — report it loudly in notes and ensure the chosen winner is deterministic).
Verify: ${PATHFIX}; go build ./... ; go vet ./codegen/... ; go test -short ./codegen/... ; regen zero-diff. gofmt -w. Set lane="cg:arch14". Report ARCH-14.`,
  { label: 'cg:arch14', phase: 'Codegen-regen', schema: REPORT })

const a3 = await agent(`${COMMON}

LANE A3 cg:arch13-19 (codegen templates; INTENDED generated diff). Read architect-review.md ARCH-13 and ARCH-19. Start from A2's tree.

ARCH-13 (stop returning ErrNotFound from successful create/update in the v1 template): codegen/api.go.tmpl — the create{{.StructName}} (~line 152) and update{{.StructName}} (~line 190) blocks do \`if len(respBody.Data) != 1 { return nil, ErrNotFound }\`. Returning the 'not found' sentinel from a successful create/update is semantically wrong and inconsistent with apiv2.go.tmpl (which returns the decoded body). Replace BOTH with a distinct, descriptive error, e.g. \`return nil, fmt.Errorf("unexpected response: expected 1 %s, got %d", "{{ .StructName }}", len(respBody.Data))\` (ensure \`fmt\` is imported in the template's import block — it likely already is; the blank-var trick may need adjusting). Keep the GET single-resource path (~line 123) returning ErrNotFound — that one is correct (get-of-one not found). Then regen. EXPECTED DIFF: every *_create*/*update* in unifi/*.generated.go that had the ErrNotFound branch now returns the fmt.Errorf form. Document in codegen/CLAUDE.md that ErrNotFound is ONLY for get/list-single, never create/update.
NOTE: this BREAKS the Wave-1 wrapper test that asserted CreateUser returns ErrNotFound on len!=1 (unifi/user_wrappers_test.go or similar). You MUST update that test to assert the new error semantics instead (it should NO LONGER be errors.Is(ErrNotFound); assert a non-nil error whose message mentions the unexpected count). Find every test asserting ErrNotFound on a create/update path and fix them. This is a documented BREAKING CHANGE (record in breakingChanges).

ARCH-19 (first-class query-param support + interim guard): codegen has no query-param support; DescribedFeature smuggles \`described-features?includeSystemFeatures=true\` into resourcePath (customizations.yml ~line 474, flagged TODO hack). The apiv2 template appends \`/%s\` for get/update/delete -> \`described-features?includeSystemFeatures=true/%s\` (broken; id after query string). DescribedFeature is in excludeResources (only listDescribedFeature wired, where the query happens to terminate cleanly) so it's a latent footgun. Implement BOTH:
  (a) INTERIM GUARD (required): in codegen (resources.go/customize.go/generator.go — wherever resourcePath is consumed or resources are validated), make the generator WARN-and/or-REJECT when a resourcePath contains '?' for any resource that emits id-suffixed URLs (get/update/delete). For list-only resources (all id-suffixed funcs excluded) a trailing query is tolerated. Pick the least-surprising behavior: reject (error) in strict mode, warn otherwise, OR auto-route the query correctly per (b).
  (b) FIRST-CLASS query-param support (preferred, do if tractable): add an optional structured form to customizations.yml ResourceCustomization — e.g. \`queryParams: { includeSystemFeatures: "true" }\` (a map) OR a structured \`resourcePath: { path: "...", query: "..." }\`. Plumb it through ResourceCustomization (customize.go) into the Resource so templates can render the path and append the query string AFTER the \`/%s\` id segment (for get/update/delete) and after the bare path (for list/create). Update api.go.tmpl + apiv2.go.tmpl path-building accordingly. Migrate DescribedFeature to use the new structured query-param form (remove the '?' from resourcePath). Regen.
If (b) is too invasive to land safely, ship (a) the guard + convert DescribedFeature to the guard-clean form (or keep it list-only) and report ARCH-19 'partial' with a precise note on what's left. Either way: NO broken URL must be emitted, and the generator must not silently emit a path with '?' followed by '/%s'.
EXPECTED DIFF: if (b), described_feature.generated.go list path changes shape (query appended correctly); guard adds no output. Verify the emitted DescribedFeature URLs are well-formed.

After: ${PATHFIX}; go build ./... ; go vet ./codegen/... ; go test -short ./codegen/... ; go test ./unifi/... (fix the broken ErrNotFound-on-create tests). Regen; the generated diff must be EXACTLY the intended ARCH-13/ARCH-19 changes (no collateral). gofmt -w. Set lane="cg:arch13-19". Report ARCH-13 (breaking) + ARCH-19.`,
  { label: 'cg:arch13-19', phase: 'Codegen-regen', schema: REPORT })

const a4 = await agent(`${COMMON}

LANE A4 cg:test15-iface (codegen customizations.yml + hand impl + regen + MOCK regen; INTENDED interface diff). Read test-review.md TEST-15. Start from A3's tree.

GOAL: add ctx-accepting variants for Version/Login/Logout/GetSystemInformation so cancellation/deadline behavior is unit-testable, WITHOUT removing the existing no-ctx methods (keep them for source-compat; they delegate to the ctx variants with context.Background or c.newRequestContext). Add EXACTLY these four PUBLIC methods, ctx-first:
  - LoginContext(ctx context.Context) error
  - LogoutContext(ctx context.Context) error
  - VersionContext(ctx context.Context) (string, error)   // NOTE: returns (string,error) — the ctx variant SHOULD surface the fetch error rather than swallowing it like Version() string does
  - GetSystemInformationContext(ctx context.Context) (*SysInfo, error)

STEPS:
(1) INTERFACE (codegen): these four methods belong in the generated Client interface. Add four entries to codegen/customizations.yml under client.functions, mirroring the existing Login/Logout/Version/GetSystemInformation entries (look at lines ~14-27, ~247) but with a ctx param and the signatures above. Keep the existing four entries too. Regen so unifi/client.generated.go gains the four *Context methods in the interface.
(2) MOCK regen (REQUIRED, the interface changed): \`${MOCKREGEN}\`. Confirm client_mock.generated.go now has the four *Context funcs and still builds.
(3) HAND IMPL: implement the four *Context methods on *client in the appropriate hand-written files (LoginContext/LogoutContext in unifi/client.go near Login/Logout; VersionContext in client.go near Version; GetSystemInformationContext in unifi/sysinfo.go near GetSystemInformation). Refactor so the EXISTING no-ctx methods delegate to the ctx variants:
   - Login() error  => return c.LoginContext(c.newRequestContext())   (or context.Background — match how Login currently derives its context; preserve the c.timeout behavior by keeping newRequestContext)
   - Logout() error => return c.LogoutContext(c.newRequestContext())
   - Version() string => v, _ := c.VersionContext(c.newRequestContext()); return v   (preserve the swallow-to-"" behavior of the legacy method)
   - GetSystemInformation() (*SysInfo, error) => return c.GetSystemInformationContext(c.newRequestContext())
   Thread the passed ctx through to the actual HTTP calls (Do/Get/Post) instead of deriving a fresh one internally, so a cancelled/deadline ctx actually aborts. PRESERVE: Version()'s sysInfo cache + double-checked locking (ARCH-01 from W0) — VersionContext must use the same cache path; GetSystemInformationContext must keep the sysInfoMu write. Per TEST-15, separate the pure cache-decision from IO where reasonable so the cached-vs-fetch branch is testable.
(4) TESTS: add unifi tests (use testhelpers_test.go newControllerServer) proving: VersionContext/GetSystemInformationContext/LoginContext/LogoutContext abort on a pre-cancelled context (context.WithCancel then cancel) — assert errors.Is(err, context.Canceled) or a wrapped form; and the happy path. Add the cached-fast-path assertion for VersionContext.
BREAKING CHANGE: the Client interface gains four methods (any external implementer of Client breaks). Record in breakingChanges. The no-ctx methods are unchanged (non-breaking for callers).

After: ${PATHFIX}; regen (zero-diff except the intended interface additions); ${MOCKREGEN}; go build ./... ; go vet ./unifi/... ; go test ./unifi/... ; go test ./unifi/ -race. gofmt -w changed hand files. Set lane="cg:test15-iface". Report TEST-15 (breaking).`,
  { label: 'cg:test15-iface', phase: 'Codegen-regen', schema: REPORT })

// ---- Phase A checkpoint: tree must be green before Phase B starts ----
let cp = await agent(`${COMMON}\nPHASE-A CHECKPOINT (read-only verify; do NOT edit, do NOT run mutating git). Prepend ${PATHFIX}. Run: go build ./... ; go test ./unifi/... ; go test -short ./codegen/... ; go vet ./codegen/... ; golangci-lint run. Confirm: (a) unifi/client.generated.go interface contains LoginContext/LogoutContext/VersionContext/GetSystemInformationContext; (b) client_mock.generated.go builds (var _ Client = &ClientMock{} compiles); (c) regen-reproducible: \`${REGEN}\` then \`git -C ${ROOT} diff --stat -- 'unifi/*.generated.go'\` EMPTY; (d) mock-in-sync: \`${MOCKREGEN}\` then \`git -C ${ROOT} diff --stat -- unifi/client_mock.generated.go\` EMPTY. IGNORE .unifi-version. allGreen=every check passed. Return VERIFY object.`,
  { label: 'phaseA-checkpoint', phase: 'Codegen-regen', schema: VERIFY_SCHEMA })
if (!cp.allGreen) {
  await agent(`${COMMON}\nPhase A left the tree not-green. Prepend ${PATHFIX}. Do NOT run mutating git. Failing:\n${JSON.stringify(cp.checks.filter(c => !c.passed), null, 2)}\nFix at codegen source + regen (+ ${MOCKREGEN} if the interface changed); NEVER hand-edit *.generated.go. Return REMEDIATE object.`, { label: 'phaseA-fix', phase: 'Codegen-regen', schema: REMEDIATE_SCHEMA })
  cp = await agent(`${COMMON}\nRe-verify Phase A. Prepend ${PATHFIX}. go build ./... ; go test ./unifi/... ; go test -short ./codegen/... ; golangci-lint run; regen zero-diff; ${MOCKREGEN} zero-diff. IGNORE .unifi-version. Return VERIFY object.`, { label: 'phaseA-recheck', phase: 'Codegen-regen', schema: VERIFY_SCHEMA })
}
if (!cp.allGreen) {
  return { stage: 'phaseA-failed', a1, a2, a3, a4, checkpoint: cp, note: 'Codegen-regen phase not green; main loop must intervene before Phase B.' }
}

// =================== PHASE B: parallel disjoint packages (unifi || codegen-runtime) ===================
phase('Quality')

const [unifiStream, codegenStream] = await parallel([
  // ---- STREAM 1: unifi package (sequential agents within; one Go package = no intra-parallel) ----
  async () => {
    const b1 = await agent(`${COMMON}

LANE unifi:requests (unifi package). Scope build/test to ./unifi ONLY (codegen edited concurrently in another stream — do NOT run \`go build ./...\` or touch codegen/). Read architect-review.md ARCH-10, ARCH-11; plan.md O5 (BINDING). Files: unifi/requests.go, unifi/unifi_errors.go, unifi/user.go.

ARCH-11 (decode-on-body, not ContentLength==0): unifi/requests.go handleResponse (~line 105) returns early without decoding when \`resp.ContentLength == 0\`. A server/proxy/HTTP2 path can deliver a non-empty JSON body while reporting ContentLength==0 -> respBody left zero-valued, caller sees empty with no error. FIX: decide on the BODY, not the header. Keep \`respBody == nil\` as the only unconditional skip. Otherwise stream-decode the body and treat io.EOF (truly empty body) as 'no content': \`if err := dec.Decode(respBody); err != nil { if errors.Is(err, io.EOF) { return nil }; return <wrapped err> }\`. Make sure the existing chunked (ContentLength == -1) case still works. Add a test: a 200 with a real JSON body but ContentLength unset/0 (you can force this with an httptest handler that does NOT set Content-Length and flushes, or by constructing the response) decodes correctly into respBody; and a genuinely empty body returns nil with respBody untouched.

ARCH-10 + O5 (centralize Meta rc:error / 200-with-error): The v1 API can return HTTP 200 with meta.rc=="error" (soft failure). Meta.error() (unifi/unifi_errors.go) detects this but is called in exactly ONE place — unifi/user.go CreateUser (~line 62, with a TODO questioning whether it's still needed). Every other decoded {meta,data} 200 ignores meta.rc. FIX per O5: centralize in the hand-written handleResponse — after a successful decode, IF the decoded respBody carries a Meta envelope with rc present AND rc != "ok"/"" , surface a *ServerError (via Meta.error()). GATE it to only trigger when a meta block is actually present (do NOT couple every response to the v1 envelope when there's no meta). Implementation approach: the generated response structs embed a \`Meta\` field (json:"meta") — you can detect/extract it generically. Options: (i) decode into a json.RawMessage tee or a small probe struct \`struct{ Meta *Meta \`json:"meta"\` }\` from a buffered copy of the body, then if Meta!=nil && Meta.RC=="error" return Meta.error(); (ii) use reflection on respBody for a Meta field. Pick the cleanest that does NOT double-read the network stream incorrectly (buffer the body once into a []byte, decode the probe from the bytes, then decode respBody from the same bytes). Coordinate with ARCH-11 (both touch the decode path) — implement them together coherently in handleResponse. Then REMOVE the one-off Meta.error() call in user.go CreateUser and its TODO (the centralized check now covers it). Add tests: a 200 with meta.rc="error" + empty data -> errors.As to *ServerError carrying the rc/msg (NOT ErrNotFound); a 200 with meta.rc="ok" -> normal decode; a 200 with NO meta block (e.g. a v2-style bare body) -> normal decode, no spurious error.

CONVENTIONS: wrap errors with %w; keep the capped-body-read discipline from ARCH-05 (W1) — don't reintroduce unbounded reads. Both changes live in handleResponse; ensure the buffered-body approach respects the existing maxErrorBodySize / a sane cap for the success path too (avoid unbounded buffering of huge success bodies — cap or stream sensibly; document the choice).
After: ${PATHFIX}; gofmt -w; go vet ./unifi/...; go test ./unifi/... . Set lane="unifi:requests". Report ARCH-10, ARCH-11 (+ breakingChanges if the 200-rc-error behavior change is observable to callers — it IS a behavior change: note it).`,
      { label: 'unifi:requests', phase: 'Quality', schema: REPORT })

    const b2 = await agent(`${COMMON}

LANE unifi:client-interceptors (unifi package, AFTER unifi:requests). Scope build/test to ./unifi ONLY. Read architect-review.md ARCH-09, ARCH-18. Files: unifi/client.go, unifi/interceptors.go.

ARCH-09 (stop mutating caller-owned ClientConfig): unifi/client.go MUTATES the caller's *ClientConfig: \`config.URL = strings.TrimRight(config.URL, "/")\` (~line 323) and \`config.UserAgent = defaultUserAgent\` when empty (~line 292-296 in buildInterceptors). A caller's struct should not be silently rewritten. FIX: operate on LOCALS / a normalized copy. Either shallow-copy the config at the top of newBareClient (\`cfg := *config\`) and use cfg everywhere, or thread the normalized URL/UserAgent as locals into buildHTTPClient/buildInterceptors without writing back to config. Ensure NOTHING writes through the caller's pointer. Add a test: construct a ClientConfig with a trailing-slash URL and empty UserAgent, build a client (use the APIStyle offline override so no network), then assert the ORIGINAL config.URL still has its trailing slash and config.UserAgent is still "" (proving non-mutation), while the client behaves normalized (requests go to the trimmed URL — assert via httptest/testhelpers).

ARCH-18 (interceptor dedup + AddInterceptor signature): unifi/interceptors.go + client.go. (a) Change \`AddInterceptor(interceptor *ClientInterceptor)\` to \`AddInterceptor(interceptor ClientInterceptor)\` (value, not pointer-to-interface) — matches how the slice []ClientInterceptor stores values and how ClientConfig.Interceptors is typed. Update the dedup logic and ALL call sites (unifi/client_test.go ~line 156,159 passes &dummy — update to pass dummy; check any others). (b) Dedup semantics: today both AddInterceptor and buildInterceptors use slices.Contains over []ClientInterceptor comparing interface values with == — which panics if a dynamic type is non-comparable (struct with slice/map/func) and only dedups identical pointers. Replace with dedup BY CONCRETE TYPE using reflect.TypeOf (only one interceptor of a given concrete type) — OR drop dedup entirely and document that uniqueness is the caller's responsibility. Choose dedup-by-concrete-type (safer, matches the 'only one CSRF/APIKey' intent). Ensure no == on potentially-non-comparable values remains. Apply the SAME dedup in buildInterceptors. Update unifi/client_interface_test.go note for AddInterceptor if its signature description changed. Add tests: adding two distinct instances of the same concrete interceptor type results in ONE; adding different types keeps both; a non-comparable interceptor type does not panic.
BREAKING CHANGE: AddInterceptor signature *ClientInterceptor -> ClientInterceptor. Record in breakingChanges.
After: ${PATHFIX}; gofmt -w; go vet ./unifi/...; go test ./unifi/... ; go test ./unifi/ -race. Set lane="unifi:client-interceptors". Report ARCH-09, ARCH-18 (breaking).`,
      { label: 'unifi:client-interceptors', phase: 'Quality', schema: REPORT })

    const b3 = await agent(`${COMMON}

LANE unifi:tests (unifi package, AFTER client-interceptors). Scope build/test to ./unifi ONLY. Pure test additions + a small pure-function extraction. Read test-review.md TEST-11, TEST-12, TEST-13 (part 2+3), TEST-14.

TEST-12 (extract pure buildMultipartUpload + upload tests): unifi/requests.go — UploadFile/UploadFileFromReader/createFormFile/escapeQuotes are 0% covered and entangled with os.File + executeRequest. EXTRACT a pure function \`buildMultipartUpload(reader io.Reader, filename, fieldName string) (body *bytes.Buffer, contentType string, err error)\` that does field defaulting (fieldName defaults to "file"), MIME detection (mimetype.DetectReader + the documented buffer-twice workaround), createFormFile + escapeQuotes for Content-Disposition. Have UploadFileFromReader call buildMultipartUpload then set the body + Content-Type + the mandatory \`X-Requested-With: XMLHttpRequest\` header, then executeRequest. Keep os.Open isolated in UploadFile only. This is a behavior-preserving refactor of hand-written code (NOT generated) — fine to edit. Tests (unifi/requests_test.go or a new unifi/upload_test.go): buildMultipartUpload field defaulting to "file", escapeQuotes with quotes/backslashes (direct test), detected Content-Type via bytes.Reader fixtures; an UploadFileFromReader httptest round-trip parsing the multipart form asserting part name defaults to "file", detected Content-Type, file bytes round-trip, and X-Requested-With==XMLHttpRequest present; an UploadFile test asserting filepath.Base filename + an os.Open-failure (nonexistent path) -> wrapped "unable to open file for upload" error.

TEST-11 (unmarshaler + Logout/Version/Meta.error branches): add table-driven internal-package tests (package unifi). unifi/json_test.go: numberOrString {1->"1", "auto"->"auto", ""->"", "null"->"" (per ARCH-07 W1), invalid -> error}; booleanishString {true/"true"/"enabled"/"1"->true, false/"false"/"disabled"/"0"/""/null->false, never errors — pin the PERMISSIVE behavior from W0/ARCH-02 (NOTE: the review text predates the permissive fix; assert permissive, not the old hard-error)}; emptyStringInt malformed cases (non-numeric quoted string, unterminated quote) hitting Unquote/Atoi error branches + MarshalJSON nil/zero->"" and non-zero->int branches. Also: TestLogout mirroring Login (API-key path issues NO request; user/pass path POSTs to logout path) — use testhelpers newControllerServer; TestVersion both cached-fast-path and error-returns-"" path (failing sysinfo server). Direct Meta.error() unit test: rc=="ok"->nil; rc=="error"-> *ServerError carrying ErrorCode+Message. (If VersionContext etc. exist from Phase A, you MAY also pin them, but TEST-15's own tests cover those — focus TEST-11 on the legacy methods + unmarshalers.)

TEST-13 part 2+3 (unifi side only — the codegen logger part is handled by the other stream): unifi/validation.go — let newValidator accept optional extra validators (e.g. \`newValidator(extra ...CustomValidator)\` or honor the existing RegisterCustomValidator seam) so a test can register a one-off validator WITHOUT mutating the shared \`customValidators\` global. unifi/api_paths.go — OldStyleAPI/NewStyleAPI are exported *vars compared by pointer identity (&OldStyleAPI ~line 104); make the API-path sets value-returning (e.g. newStyleAPI()/oldStyleAPI() returning copies) OR document them immutable and switch identity comparison to a style enum, so parallel tests cannot corrupt shared state. Choose the lower-risk option that keeps determineApiStyle working and the W1 apiStyleFromStatus tests passing; if switching to an enum is too invasive, at minimum make the sets value-returning copies and update the comparison. Add focused tests for the new seam(s). Keep generated callers compiling.

TEST-14 (consolidate test helpers): unifi/testhelpers_test.go already exists (newControllerServer, controllerServer.client(), apiV1Path/apiV2 — seeded in W1). MIGRATE the remaining ad-hoc test infra to it and kill swallowed-error patterns: helpers_test.go (newNewStyleClient/runTestServer), api_paths_test.go, client_test.go, concurrency_test.go, requests_test.go, sysinfo_test.go, validation_test.go each spin up their own server/client. Where a test's needs are met by newControllerServer/controllerServer.client(), migrate it; where a test genuinely needs something special (e.g. concurrency_test's race harness, api_paths_test's status-probe server), LEAVE it but route through the shared helper if feasible and remove duplicated boilerplate. Do NOT break any existing test. If full consolidation of a given file is risky, consolidate what's safe and report the rest as partial with reasons. Prefer require.NoError over swallowed errors.

After: ${PATHFIX}; gofmt -w; go vet ./unifi/...; go test ./unifi/... ; go test ./unifi/ -race ; report coverage delta via go test -cover ./unifi/. Set lane="unifi:tests". Report TEST-11, TEST-12, TEST-13(unifi), TEST-14.`,
      { label: 'unifi:tests', phase: 'Quality', schema: REPORT })

    return { b1, b2, b3 }
  },
  // ---- STREAM 2: codegen runtime (codegen package, NO regen — disjoint from unifi) ----
  async () => {
    const c1 = await agent(`${COMMON}

LANE codegen:download (codegen package; do NOT regen, do NOT touch unifi/). Scope build/test to ./codegen ONLY (unifi edited concurrently). Read architect-review.md ARCH-15, ARCH-16. Files: codegen/download.go, codegen/utils.go.

ARCH-15 (download pipeline timeouts/cancellation + integrity/host validation): codegen/download.go. (a) downloadJar uses \`http.NewRequestWithContext(context.Background(), ...)\` — not cancellable. Thread a context.Context through DownloadAndExtract -> downloadJar so callers can cancel/deadline. Update the caller in codegen/main.go (and any generator.go path) to pass a context (a context.Background() with a sensible timeout, or wire to the existing flow). (b) The injected *http.Client (W1 seam) may have no timeout — ensure a sensible default timeout when the client is nil/has none (e.g. construct the default client WITH a timeout, or set a deadline via context). (c) Integrity/host validation: validate the download host/URL scheme (https + expected host) before fetching; if a checksum is available from the firmware/version metadata, verify it after download (if no checksum source exists, at least validate Content-Length sanity + host + scheme and document that checksum pinning needs an upstream source). Re-validate the firmware channel/product where that data is available (see fwupdate.go/version.go). Keep changes minimal and offline-testable.
ARCH-16 (atomic extraction): DownloadAndExtract extracts in place; if extraction fails halfway, the outputDir is left partial and ensurePath returns created==false on re-run -> the partial tree is silently ACCEPTED and never re-extracted. FIX: make extraction atomic — extract into a temp dir (os.MkdirTemp as a sibling of outputDir) then os.Rename into place on success, OR write a \`.complete\` sentinel file after a fully-successful extract and have the 'already extracted?' check require the sentinel (treat a dir without the sentinel as incomplete -> re-extract). Pick one; temp-dir+rename is cleaner if the JSON files all live under outputDir. Ensure re-runs after a failed/partial extract correctly re-extract. Clean up temp dirs on failure.

Tests (codegen/download_test.go / codegen/utils_test.go): use the W1 injected-client httptest seam + the tiny ar(data.tar.xz(ace.jar(json))) fixture. Cover: context cancellation aborts the download (pre-cancelled ctx -> error); a non-https or wrong-host URL is rejected; atomic extraction — simulate a mid-extract failure (e.g. a corrupt/truncated jar or a zip entry that errors) and assert the outputDir is NOT left in a state that a re-run treats as complete (re-run re-extracts / sentinel absent); happy path still produces the JSON files + sentinel/renamed dir. Keep everything offline (\`go test -short ./codegen/...\`).
After: ${PATHFIX}; gofmt -w; go vet ./codegen/...; go test -short ./codegen/... . Set lane="codegen:download". Report ARCH-15, ARCH-16 (+ breakingChanges: DownloadAndExtract/downloadJar signature gains ctx — note it; it's internal codegen API).`,
      { label: 'codegen:download', phase: 'Quality', schema: REPORT })

    const c2 = await agent(`${COMMON}

LANE codegen:logger-utils (codegen package, AFTER codegen:download; do NOT regen, do NOT touch unifi/). Scope build/test to ./codegen ONLY. Read test-review.md TEST-13 (part 1), TEST-16. Files: codegen/main.go, codegen/generator.go, codegen/options/*, codegen/utils.go, codegen/customize.go(resources.go) for the logger consumers.

TEST-13 part 1 (thread the codegen logger instead of the package global): codegen has \`var log = logrus.New()\` (main.go ~line 16) mutated by setupLogging and read via test.NewLocal(log) in tests — a race surface and a barrier to parallel output-asserting tests. THREAD a logger through generate/options: introduce a logger field (a *logrus.Logger or a small Logger interface) on the generator options/customizer and pass it down to the code that logs (generator.go, resources.go warnings from ARCH-14, customize.go ExcludedClientFunctions warns, clients.go). Keep \`var log\` as a default for the CLI path (main.go can construct the logger and inject it), but production generate() calls should use the injected logger, enabling parallel, output-asserting, race-free tests. Update TestSetupLogging / TestExcludedClientFunctions_UnknownActionWarns / TestGenerate to use an injected logger + local hook instead of mutating the global; make the previously-serialized logger tests parallel-safe. COORDINATE: the ARCH-14 lane (Phase A, already done) added log.Warnf calls in resources.go — make those use the threaded logger too (read what A2 did; adapt). Do NOT break the existing generation flow.

TEST-16 (utils_test.go + inject codegen v2 base dir): add codegen/utils_test.go (if A-phase didn't already create one for ARCH-16; if it exists, EXTEND it, do not clobber). Cover: ensurePath over existing dir -> (false,nil); missing path -> creates + (true,nil); a FILE path -> "isn't a directory" error. findProjectRoot: chdir into a t.TempDir containing a synthetic go.mod -> returns that root (use t.Chdir if available in go1.26, else save/restore cwd); assert findCodegenDir joins "codegen". Then DECOUPLE generateCode from cwd: inject the codegen/v2 base dir into generateCode (a parameter or an options field) instead of discovering it via findCodegenDir at runtime, so generation is unit-testable without the real repo layout. Keep the CLI defaulting to findCodegenDir when not injected. Add a test that generateCode works with an injected base dir pointing at a fixture.

After: ${PATHFIX}; gofmt -w; go vet ./codegen/...; go test -short ./codegen/... ; go test ./codegen/ -race -run 'TestSetupLogging|TestEnsurePath|TestFindProjectRoot|TestFindCodegenDir|TestExcluded' . Set lane="codegen:logger-utils". Report TEST-13(codegen), TEST-16.`,
      { label: 'codegen:logger-utils', phase: 'Quality', schema: REPORT })

    return { c1, c2 }
  },
])

// =================== PHASE C: verify -> review -> remediate ===================
phase('Verify')
let verify = await agent(VERIFY_PROMPT, { label: 'verify', phase: 'Verify', schema: VERIFY_SCHEMA })
let attempts = 0
while (!verify.allGreen && attempts < 3) {
  attempts++
  await agent(FIX_PROMPT(verify), { label: `fix#${attempts}`, phase: 'Verify', schema: REMEDIATE_SCHEMA })
  verify = await agent(VERIFY_PROMPT, { label: `re-verify#${attempts}`, phase: 'Verify', schema: VERIFY_SCHEMA })
}
if (!verify.allGreen) {
  return { stage: 'verify-failed', phaseA: { a1, a2, a3, a4 }, phaseB: { unifiStream, codegenStream }, verify, note: 'Could not reach green after 3 fix attempts; main loop must intervene.' }
}

phase('Review')
const W2_INTENT = `Wave 2 implemented (branch chore/review-1.11.0, P2 quality & codegen robustness):
PHASE A (codegen, regen): ARCH-20 (shared template partial de-dup), ARCH-21 (apply-customizations-once / split resource vs field overrides), ARCH-14 (warn on dropped/colliding fields + deterministic key sort + strict mode + golden type-diff), ARCH-13 (v1 template no longer returns ErrNotFound from successful create/update — distinct error instead; ErrNotFound is get/list-single only — BREAKING), ARCH-19 (query-param support / guard rejecting '?' in id-suffixed resourcePath), TEST-15 (added LoginContext/LogoutContext/VersionContext/GetSystemInformationContext to the Client interface + hand impl delegating, mock regenerated — BREAKING interface growth).
PHASE B unifi: ARCH-11 (decode-on-body, io.EOF=empty, not ContentLength==0), ARCH-10 (centralized Meta rc:error in handleResponse gated on meta present; user.go TODO resolved), ARCH-09 (stop mutating caller ClientConfig URL/UserAgent), ARCH-18 (AddInterceptor takes ClientInterceptor not *ClientInterceptor + dedup by concrete type — BREAKING), TEST-11 (unmarshaler/Logout/Version/Meta.error tests), TEST-12 (pure buildMultipartUpload + upload tests), TEST-13-unifi (newValidator extra validators + value-returning API-path sets), TEST-14 (consolidated testhelpers).
PHASE B codegen: ARCH-15 (download ctx/timeout + host/scheme validation), ARCH-16 (atomic extraction temp-dir+rename or .complete sentinel), TEST-13-codegen (threaded logger), TEST-16 (utils_test + injected v2 base dir).`

const [arch, test] = await parallel([
  () => agent(`You are a LEAD SOFTWARE ARCHITECT doing an ADVERSARIAL, READ-ONLY review of Wave 2. Do NOT edit. Do NOT run mutating git. Repo root: ${ROOT}. Prepend ${PATHFIX} for go commands; IGNORE .unifi-version.
Review the WORKING-TREE changes since the last commit (use \`git -C ${ROOT} diff\` / \`git -C ${ROOT} status\` READ-ONLY to see what changed, and read the changed files). Hand-written unifi: requests.go, unifi_errors.go, client.go, interceptors.go, user.go, validation.go, api_paths.go, sysinfo.go, json.go + new/changed *_test.go. codegen: api.go.tmpl, apiv2.go.tmpl, the new shared partial, customize.go, resources.go, generator.go, main.go, download.go, utils.go, customizations.yml + tests. Generated: client.generated.go (interface +4 ctx methods), *_create/*_update (ErrNotFound->fmt.Errorf), described_feature, client_mock.generated.go.
${W2_INTENT}
Review RIGOROUSLY:
(1) ARCH-10/11 handleResponse — is the body buffered ONCE and decoded correctly (no double network read)? Is the Meta-rc-error check correctly GATED on a meta block present (no false positives on v2 bare bodies, no over-matching)? Does it correctly distinguish meta.rc=error from ErrNotFound? Is success-body buffering capped (no unbounded memory)? Run go test ./unifi/.
(2) ARCH-13 — does NO generated create/update still return ErrNotFound (grep the generated tree)? Is the GET-single path still ErrNotFound? Is the new error informative? Were all tests asserting the old behavior updated correctly (not just deleted)?
(3) ARCH-19 — are emitted DescribedFeature (and any query-param) URLs well-formed (id NOT after '?')? Is the guard actually triggered for id-suffixed resourcePaths containing '?'?
(4) ARCH-09 — is the caller ClientConfig truly never mutated (check buildHTTPClient + buildInterceptors + newBareClient — no write through the pointer)?
(5) ARCH-18 — is dedup-by-concrete-type correct and panic-safe (no == on non-comparable)? Are ALL AddInterceptor call sites updated to the value signature?
(6) TEST-15 — do the *Context methods actually thread ctx to the HTTP call (cancellation truly aborts)? Does Version() still cache + swallow as before? Is the mock in sync (var _ Client = &ClientMock{} builds)?
(7) ARCH-14 — deterministic sort correct; warnings fire on real drop/collision; strict mode fails the build; golden test meaningful. ARCH-20/21 — zero functional diff confirmed (regen reproducible)?
(8) ARCH-15/16 — ctx cancellation works; atomic extraction genuinely prevents accepting a partial dir; host/scheme validation sound.
(9) regressions, %w wrapping, conventions, concurrency (go test ./unifi/ -race).
For each issue: severity blocker|major|minor|nit, specific file + concrete fix. persona="architect". Return the REVIEW structured object only.`,
    { label: 'review:architect', phase: 'Review', schema: REVIEW_SCHEMA }),
  () => agent(`You are a LEAD TEST/SDET doing an ADVERSARIAL, READ-ONLY review of Wave 2 TEST quality + coverage. Do NOT edit. Do NOT run mutating git. Repo root: ${ROOT}. Prepend ${PATHFIX}; IGNORE .unifi-version.
Read the new/changed *_test.go in unifi/ (requests/upload, json, client, interceptors, validation, api_paths, sysinfo, testhelpers, *Context tests) and codegen/ (download, utils, customize, resources golden, main/generator logger tests). Run \`go test -cover ./unifi/\`, \`go test ./unifi/ -race\`, \`go test -short ./codegen/...\`, \`go test ./codegen/ -race\`.
${W2_INTENT}
Review RIGOROUSLY:
(1) ARCH-10/11 tests — is the ContentLength==0-with-body case actually exercised (not just asserted trivially)? Is meta.rc=error vs ErrNotFound distinction tested for BOTH branches incl. no-meta passthrough?
(2) ARCH-13 — are the updated create/update tests asserting the NEW error (not errors.Is(ErrNotFound)) and is there a guard that ErrNotFound is NOT returned on create/update?
(3) TEST-15 — do the *Context tests use a REAL pre-cancelled/deadline context and assert prompt abort (errors.Is(context.Canceled/DeadlineExceeded)), not a time.Sleep hack? Cached-fast-path asserted?
(4) TEST-12 — buildMultipartUpload field-defaulting/escapeQuotes/MIME + the X-Requested-With round-trip + os.Open-failure all covered?
(5) TEST-11 — unmarshaler branches (incl. error branches) + Logout API-key-vs-POST + Version cached-vs-error + Meta.error both branches covered? booleanishString asserted PERMISSIVE (not the stale hard-error from the review text)?
(6) TEST-13 — does the threaded logger let the logger tests run PARALLEL/race-free now (no global mutation)? newValidator extra-validator seam + value-returning api-path sets tested?
(7) TEST-16 — ensurePath/findProjectRoot/findCodegenDir failure modes pinned; generateCode injected-base-dir tested?
(8) ARCH-14 golden test — does it actually catch a type-flip/field-drop (reason about it)? ARCH-15/16 — cancellation + atomic-extraction-partial-failure tests meaningful?
(9) coverage genuinely up vs baseline (unifi was 9.1%); any weakened assertions, flaky timing, missing t.Parallel; offline-ness of \`go test -short ./codegen\` preserved.
For each issue: severity blocker|major|minor|nit, specific file + suggestion. persona="test-lead". Return the REVIEW structured object only.`,
    { label: 'review:test-lead', phase: 'Review', schema: REVIEW_SCHEMA }),
])

const mustFix = [...arch.findings, ...test.findings].filter(f => f.severity === 'blocker' || f.severity === 'major')
let remediation = null, finalVerify = verify
if (mustFix.length) {
  phase('Remediate')
  remediation = await agent(`Apply remediation for Wave 2 review findings classified blocker/major. Repo root: ${ROOT}. Prepend ${PATHFIX}. Do NOT run mutating git. NEVER hand-edit *.generated.go (fix at codegen source + regen; if interface changed, also \`${MOCKREGEN}\`). IGNORE .unifi-version.
Findings:\n${JSON.stringify(mustFix, null, 2)}\n
${COMMON}
For EACH finding: apply a minimal correct fix preserving Wave 2 intent; keep tabs+gofmt; do NOT weaken tests. Apply receiving-code-review discipline: if a finding is out-of-scope/subjective/technically wrong, SKIP it with a clear technical reason — do NOT blindly comply. After fixes: go build ./...; go test ./unifi/...; go test ./unifi/ -race; go test -short ./codegen/...; golangci-lint run; regen zero-diff; ${MOCKREGEN} zero-diff — all green. Return the REMEDIATE object {applied, skipped, filesChanged, notes}.`,
    { label: 'remediate', phase: 'Remediate', schema: REMEDIATE_SCHEMA })
  finalVerify = await agent(VERIFY_PROMPT, { label: 're-verify-final', phase: 'Remediate', schema: VERIFY_SCHEMA })
}

return {
  stage: 'complete',
  phaseA: { a1, a2, a3, a4 },
  phaseB: { unifiStream, codegenStream },
  verify, review: { architect: arch, testLead: test }, mustFixCount: mustFix.length, remediation, finalVerify,
}
