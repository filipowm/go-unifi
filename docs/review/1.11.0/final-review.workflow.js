export const meta = {
  name: 'final-whole-codebase-review',
  description: 'Final 1.11.0 review (process contract §2.8): whole-codebase architect + test-lead adversarial review across 8 dimensions, per-finding verification, gated remediation, closing summary.',
  phases: [
    { title: 'Baseline', detail: 'establish in-workflow ground truth: build/lint/test/-race/offline-codegen/vet + coverage + regen/mock idempotency' },
    { title: 'Review', detail: '8 parallel read-only dimension reviewers (architect + test-lead lenses) over the cumulative Wave 0-2 result' },
    { title: 'Verify-findings', detail: 'adversarially verify each finding: is it real, in-scope for 1.11.0, and what severity (conservative — committed tree is already green)' },
    { title: 'Synthesize', detail: 'dedup + classify confirmed findings; draft the closing summary' },
    { title: 'Remediate', detail: 'fix confirmed blocker/major only (receiving-code-review discipline); re-verify' },
  ],
}

const ROOT = '/Users/filipowm/Documents/dev/workspaces/unifi/go-unifi'
const PATHFIX = 'export PATH="/opt/homebrew/opt/go/bin:$PATH"'
const MOQ = '/Users/filipowm/go/bin/moq'
const REGEN = `cd ${ROOT}/unifi && ${PATHFIX} && go run ../codegen -version-base-dir=../codegen 9.5.21`
const MOCKREGEN = `cd ${ROOT}/unifi && ${PATHFIX} && ${MOQ} -out client_mock.generated.go . Client`

const CONTEXT = `Repo: ${ROOT} — go-unifi, a Go client for the UniFi controller API. Branch chore/review-1.11.0 (HEAD=50a65fd).
This is the FINAL whole-codebase review of the completed 1.11.0 review-remediation effort (Waves 0,1,2 — all P0/P1/P2 findings). P3 findings (ARCH-23..31, TEST-17..20) were DISCARDED and are OUT OF SCOPE; ARCH-12 (codegen-emit-wrappers) was SKIPPED — do NOT propose them. The cumulative changes vs main touch the hand-written client core (client.go, requests.go, interceptors.go, unifi_errors.go, validation.go, api_paths.go, json.go, sysinfo.go, user.go, setting.go, setting_registry.go), the codegen pipeline (resources.go, customize.go, generator.go, clients.go, download.go, utils.go, version.go, main.go, api.go.tmpl, apiv2.go.tmpl, common.tmpl, customizations.yml), and all the generated *.generated.go + tests.
Source of truth for findings already addressed: docs/review/1.11.0/{summary,plan,architect-review,test-review,breaking_changes,implementation-log}.md. Binding decisions O1-O5 + plan §0 (e.g. TLS secure-by-default via *bool; permissive booleanishString; centralized Meta rc:error gated on meta present; UseLocking dropped from request path; Get/Delete reqBody foot-gun DOCUMENTED not removed).
HARD RULES for any go command: prepend ${PATHFIX}. NEVER hand-edit *.generated.go (fix at codegen source + regen; + \`${MOCKREGEN}\` if the interface changes). IGNORE any .unifi-version diff (it is a user-owned uncommitted file — never touch/stage/commit it). Do NOT run mutating git (no add/commit/rm/reset/stash/restore/checkout); read-only git diff/log/show/status is allowed.`

const REVIEW_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['dimension', 'persona', 'overallAssessment', 'regressionsFound', 'findings'],
  properties: {
    dimension: { type: 'string' }, persona: { type: 'string' },
    overallAssessment: { type: 'string' }, regressionsFound: { type: 'boolean' },
    findings: { type: 'array', items: { type: 'object', additionalProperties: false,
      required: ['id', 'severity', 'area', 'file', 'description', 'suggestion'],
      properties: {
        id: { type: 'string' }, // short stable handle, e.g. "FR-requests-1"
        severity: { type: 'string', enum: ['blocker', 'major', 'minor', 'nit'] },
        area: { type: 'string' }, file: { type: 'string' }, relatesTo: { type: 'string' },
        description: { type: 'string' }, suggestion: { type: 'string' },
      } } },
  },
}
const VERDICT_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['id', 'isReal', 'inScope', 'severity', 'recommendation', 'rationale'],
  properties: {
    id: { type: 'string' }, isReal: { type: 'boolean' }, inScope: { type: 'boolean' },
    severity: { type: 'string', enum: ['blocker', 'major', 'minor', 'nit'] },
    recommendation: { type: 'string', enum: ['fix', 'document', 'skip'] },
    rationale: { type: 'string' },
  },
}
const VERIFY_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['allGreen', 'checks', 'coverage', 'summary'],
  properties: {
    allGreen: { type: 'boolean' },
    checks: { type: 'array', items: { type: 'object', additionalProperties: false, required: ['name', 'passed', 'detail'], properties: { name: { type: 'string' }, passed: { type: 'boolean' }, detail: { type: 'string' } } } },
    coverage: { type: 'string' }, summary: { type: 'string' },
  },
}
const SYNTH_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['verdict', 'confirmedFindings', 'closingSummary', 'mustFixIds'],
  properties: {
    verdict: { type: 'string', enum: ['ship', 'ship-with-followups', 'needs-remediation'] },
    confirmedFindings: { type: 'array', items: { type: 'object', additionalProperties: false,
      required: ['id', 'severity', 'inScope', 'recommendation', 'file', 'description', 'suggestion'],
      properties: { id: { type: 'string' }, severity: { type: 'string' }, inScope: { type: 'boolean' }, recommendation: { type: 'string' }, file: { type: 'string' }, description: { type: 'string' }, suggestion: { type: 'string' } } } },
    mustFixIds: { type: 'array', items: { type: 'string' } },
    closingSummary: { type: 'string' },
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

const VERIFY_PROMPT = `${CONTEXT}
You are the FINAL-REVIEW baseline/verification gate. Do NOT edit. Prepend ${PATHFIX}. From repo root, capture output tails. IGNORE .unifi-version.
Run and record each: (1) go build ./... ; (2) golangci-lint run ; (3) go test ./unifi/... ; (4) go test ./unifi/ -race ; (5) go test -short ./codegen/... with GOPROXY=off (must be OFFLINE) ; (6) go vet ./codegen/... ; (7) go test -short ./codegen/ -race ; (8) regen-idempotency: H1=\`find ${ROOT}/unifi -name '*.generated.go' | sort | xargs shasum | shasum\`, run \`${REGEN}\`, H2=same — PASS iff H1==H2 ; (9) mock-idempotency: M1=shasum of unifi/client_mock.generated.go, run \`${MOCKREGEN}\`, M2=same — PASS iff M1==M2.
Also capture coverage: go test -cover ./unifi/ and go test -short -cover ./codegen/ (put both numbers in the coverage field).
allGreen = every check passed. Return the VERIFY object only.`

// ---------------- Phase: Baseline ----------------
phase('Baseline')
const baseline = await agent(VERIFY_PROMPT, { label: 'baseline-verify', phase: 'Baseline', schema: VERIFY_SCHEMA })
const baselineNote = `IN-WORKFLOW BASELINE (ground truth, already run — do NOT re-run the full suite; do targeted spot-checks only): allGreen=${baseline.allGreen}; coverage=${baseline.coverage}; ${baseline.summary}`

// ---------------- Phase: Review (8 parallel dimensions, read-only) ----------------
phase('Review')
const DIMS = [
  { key: 'client-config', persona: 'architect', focus: `Client construction & configuration. Files: unifi/client.go, unifi/api_paths.go, unifi/interceptors.go. Scrutinize: ARCH-09 (does ANY path still mutate the caller's *ClientConfig — URL/UserAgent/Interceptors? check newClientFromConfig, buildHTTPClient, buildInterceptors), ARCH-06 (TLS truly secure-by-default: is there ANY path that skips verification without an explicit VerifySSL=&false? is the WARN emitted exactly when disabled? is the *bool API sound), ARCH-18 (AddInterceptor value sig + dedup-by-concrete-type: panic-safe? all call sites updated? config.Interceptors deduped the same way), ARCH-04 (concurrency: sysInfo behind sysInfoMu, CSRF behind its mutex, coarse lock truly gone, UseLocking a harmless no-op), TEST-09 (apiStyleFromStatus pure fn + APIStyle offline seam coherent). Is the public construction API coherent end-to-end?` },
  { key: 'requests-pipeline', persona: 'architect', focus: `Request/response pipeline. Files: unifi/requests.go, unifi/unifi_errors.go. Scrutinize: ARCH-11 (decode-on-body: io.EOF=empty, respBody==nil the only unconditional skip, chunked & ContentLength==0-with-body both decode; the 64 MiB cap + explicit overflow error), ARCH-10/O5 (centralized Meta rc:error: body buffered EXACTLY once and reused for the meta probe AND the real decode — NO double network read; gated strictly on a meta block present so v2 bare bodies don't fabricate errors; rc=="" treated as ok), ARCH-05 interplay (HandleError still keeps status/method/URL on empty/non-JSON bodies; capped read), TEST-12 (buildMultipartUpload pure + X-Requested-With + MIME). Any correctness trap, double-read, unbounded buffer, or error-taxonomy split?` },
  { key: 'error-model', persona: 'architect', focus: `Error & validation model coherence. Files: unifi/unifi_errors.go, unifi/validation.go, unifi/json.go. Scrutinize: the whole error taxonomy — ServerError (Unwrap/Is, 404->ErrNotFound, soft-200 rc:error now carries status/method/URL), ErrNotFound identity survives %w chains, ValidationError (Unwrap, deterministic sorted Error(), errors.As guard in Validate), Meta.error() semantics (rc ok/empty->nil, error->ServerError). json.go unmarshalers: numberOrString (null->""), permissive booleanishString (never hard-errors), emptyStringInt branches. Are errors.Is/As contracts consistent and documented? Any sentinel that can be accidentally swallowed?` },
  { key: 'settings-wrappers', persona: 'architect', focus: `Settings registry, drift guards & hand-written wrappers. Files: unifi/setting.go, unifi/setting_registry.go, unifi/user.go, unifi/sites.go, unifi/device.go, unifi/client_interface_test.go (the reflection drift guard), and the generated setting_*.generated.go self-registration. Scrutinize: ARCH-08 (generated drift-proof registry via init() self-registration — init-order safe? the reflection guard's allowlist not hiding real interface gaps?), ARCH-03 (all setting keys present), ARCH-13 interaction (CreateUser keeps its nested-meta check AND its inner ErrNotFound for genuine not-found; generated Create/Update no longer return ErrNotFound). Is the ErrNotFound contract (get/list-single only) actually honored across BOTH generated and hand-written wrappers? Any remaining drift surface?` },
  { key: 'codegen-templates', persona: 'architect', focus: `Codegen templates & client/interface emission. Files: codegen/api.go.tmpl, codegen/apiv2.go.tmpl, codegen/common.tmpl, codegen/clients.go, codegen/generator.go (parse site). Scrutinize: ARCH-20 (shared common.tmpl partial correctly parsed into BOTH templates; no rendered-output change), ARCH-13 (create/update emit the descriptive fmt.Errorf, GET-single still ErrNotFound), ARCH-19 (query-params rendered AFTER the /%s id segment for get/update/delete; guard rejects '?' in id-suffixed resourcePath; DescribedFeature URLs well-formed), TEST-15 (the 4 *Context methods generated into the Client interface; no-ctx kept). Confirm regen reproducibility holds (baseline already checked). Any malformed URL, template-define collision, or interface incoherence?` },
  { key: 'codegen-resources', persona: 'architect', focus: `Codegen field inference, customization & robustness. Files: codegen/resources.go, codegen/customize.go. Scrutinize: ARCH-14 (warn on dropped field + on CamelCase collision with a DIFFERENT JSONName; deterministic JSON-key sort in BOTH processFields and nested fieldInfoFromMap; UNIFI_CODEGEN_STRICT promotes to hard error and aborts via errors.As(strictViolation); golden type-diff meaningful), ARCH-21 (customizations applied EXACTLY once — the double-apply is gone; resource- vs field-level overrides cleanly split; the per-resource behaviors SettingGlobalAp 6E->SixE / SettingUsg *Timeout->emptyStringInt except ArpCacheTimeout / SettingMgmt XSshKeys / MdnsEnabled intact). Does the collision check miss the whitespace-prefixed base-struct keys (a known nit) — is that a real gap for 9.5.21? Any nondeterminism or silent data loss left?` },
  { key: 'codegen-download-security', persona: 'architect', focus: `Codegen download/extraction security & robustness. Files: codegen/download.go, codegen/utils.go, codegen/version.go. Scrutinize: ARCH-15 (context+timeout threaded through DownloadAndExtract/downloadJar; nil-client gets a default timeout; https + Ubiquiti-host validation before fetch; firmware channel/product re-validated; Body.Close before status check), ARCH-16 (atomic extraction: temp dir + rename + .extract-complete sentinel; a crashed/partial extract is NOT accepted and re-extracts; temp cleaned on failure; the RemoveAll->Rename publish window is sentinel-safe), TEST-02/ARCH-17 (sanitizeExtractedPath zip-slip, copyWithLimit bomb cap, oversize entry). Any path-traversal, TOCTOU, unbounded read, or partial-state-accepted bug?` },
  { key: 'tests-and-docs', persona: 'test-lead', focus: `Whole test-suite quality + breaking-change/doc accuracy. Read across unifi/*_test.go and codegen/*_test.go + docs/review/1.11.0/breaking_changes.md + docs/configuration.md + docs/migrating_from_upstream.md (if present). Scrutinize: (a) is \`go test -short ./codegen\` genuinely OFFLINE (no live test slips through; verify the -short gating) and is the live set still runnable without -short; (b) -race cleanliness of the whole suite incl. the testhelpers mutex (TEST-14) and the package-global maxResponseBodySize/cap test serialization; (c) convention adherence (testify, table map[string]struct{}, t.Parallel on outer+subtests, httptest) and any weakened assertions / smoke-only tests / flaky timing; (d) coverage adequacy on the NEW critical paths (handleResponse meta+cap, buildMultipartUpload, *Context cancellation, download atomic/host, strict mode, golden); (e) DOC ACCURACY: does breaking_changes.md match the ACTUAL public API delta vs main (run \`git -C ${ROOT} diff main -- 'unifi/*.go' ':!unifi/*_test.go' ':!unifi/*.generated.go'\` and the generated interface) — are ALL four Wave-2 breaks + the Wave-0/1 breaks documented, accurate, with correct migration; any UNDOCUMENTED break or stale/incorrect doc claim? Is the mock in sync with the interface?` },
]

const reviewed = await pipeline(
  DIMS,
  (d) => agent(`${CONTEXT}

You are a LEAD ${d.persona === 'test-lead' ? 'TEST/SDET' : 'SOFTWARE ARCHITECT'} performing an ADVERSARIAL, READ-ONLY final review. Dimension: "${d.key}".
${baselineNote}
FOCUS:\n${d.focus}

Read the actual current code (and git diff vs main where useful) — do not rely on the implementation-log's claims; VERIFY them against the source. Run TARGETED spot-checks only (single tests, grep, go doc) — the full suite + -race + coverage are already green per the baseline, so do NOT re-run them wholesale. Hunt for: real regressions, correctness traps, security gaps, incoherent/undocumented public API, missing or weak tests on critical new paths, and any binding-decision (O1-O5/§0) violated. Be skeptical and specific. Do NOT propose P3/ARCH-12 work (out of scope). For each issue: a stable id "FR-${d.key}-N", severity (blocker|major|minor|nit), exact file, crisp description, concrete suggestion. If the dimension is clean, return an empty findings array and say so. persona="${d.persona}". Return the REVIEW object only.`,
    { label: `review:${d.key}`, phase: 'Review', schema: REVIEW_SCHEMA }),
  // verify each finding from this dimension as soon as the dimension completes
  (review, d) => parallel((review.findings || []).map((f) => () =>
    agent(`${CONTEXT}

You are an ADVERSARIAL VERIFIER for ONE final-review finding. Be conservative: the committed tree is already green and all P0/P1/P2 findings are implemented — your job is to decide whether this finding is a REAL, IN-SCOPE defect worth changing committed code for, or noise.
FINDING (id ${f.id}, claimed severity ${f.severity}, file ${f.file}):
${f.description}
SUGGESTION: ${f.suggestion}

Verify against the ACTUAL source (read the file, run a targeted test/grep if needed; prepend ${PATHFIX}). Decide:
- isReal: does the defect actually exist as described in the current code? (If the code already handles it, isReal=false.)
- inScope: is it within the 1.11.0 remediation scope (a regression or gap in P0/P1/P2 work, or an inaccurate breaking-change doc)? P3 findings (ARCH-23..31/TEST-17..20), ARCH-12, and pre-existing-unrelated issues are OUT OF SCOPE (inScope=false). A binding-decision (O1-O5/§0) outcome is NOT a defect.
- severity: your independent re-assessment (blocker|major|minor|nit). Default DOWN when uncertain.
- recommendation: "fix" (only if isReal && inScope && severity in {blocker,major}), "document" (real+in-scope but minor/nit or doc-only), or "skip" (not real, out-of-scope, or subjective/wrong).
Default to skip/refute when genuinely uncertain. Return the VERDICT object only (id="${f.id}").`,
      { label: `verify:${f.id}`, phase: 'Verify-findings', schema: VERDICT_SCHEMA })
      .then((v) => ({ ...f, verdict: v }))
      .catch(() => null)
  )),
)

const allVerified = reviewed.flat().filter(Boolean).filter((x) => x && x.verdict)

// ---------------- Phase: Synthesize ----------------
phase('Synthesize')
const synth = await agent(`${CONTEXT}

You are the FINAL-REVIEW SYNTHESIZER. Below are all dimension findings with their adversarial verdicts (isReal/inScope/severity/recommendation). Dedup overlapping findings, drop those with isReal=false or inScope=false, and classify the rest.
${baselineNote}
VERIFIED FINDINGS (JSON):
${JSON.stringify(allVerified.map((x) => ({ id: x.id, file: x.file, severity: x.severity, description: x.description, suggestion: x.suggestion, verdict: x.verdict })), null, 2)}

Produce:
- verdict: "ship" (no confirmed in-scope blocker/major; at most minors/nits), "ship-with-followups" (only minors/nits worth noting), or "needs-remediation" (>=1 confirmed in-scope blocker/major).
- confirmedFindings: the deduped real+in-scope findings (keep their recommendation), each with file/severity/description/suggestion.
- mustFixIds: ids of confirmed findings with recommendation=="fix" (in-scope blocker/major only).
- closingSummary: a tight, honest closing assessment of the ENTIRE 1.11.0 effort (Waves 0-2): what was delivered, the cumulative public breaking changes, test/coverage posture, residual risks/deferred nits, and the overall quality verdict. Write it for the maintainer. Be direct, no fluff.
Return the SYNTH object only.`,
  { label: 'synthesize', phase: 'Synthesize', schema: SYNTH_SCHEMA })

// ---------------- Phase: Remediate (gated) ----------------
let remediation = null, finalVerify = baseline
const mustFix = (synth.confirmedFindings || []).filter((f) => synth.mustFixIds.includes(f.id))
if (mustFix.length) {
  phase('Remediate')
  remediation = await agent(`${CONTEXT}

Apply remediation for the FINAL-REVIEW findings confirmed as in-scope blocker/major. Prepend ${PATHFIX}. Do NOT run mutating git. NEVER hand-edit *.generated.go (fix at codegen source + regen; + \`${MOCKREGEN}\` if the interface changes). IGNORE .unifi-version.
FINDINGS TO FIX (JSON):
${JSON.stringify(mustFix, null, 2)}

For EACH: minimal correct fix preserving the 1.11.0 intent + binding decisions; keep tabs+gofmt; do NOT weaken tests; add/adjust a test where behavior changes. Apply receiving-code-review discipline: if on close inspection a finding is actually not-real, out-of-scope, or technically wrong, SKIP it with a precise reason — do NOT blindly comply. After fixes: go build ./... ; go test ./unifi/... ; go test ./unifi/ -race ; go test -short ./codegen/... ; golangci-lint run ; regen + mock idempotent — all green. Return the REMEDIATE object.`,
    { label: 'remediate', phase: 'Remediate', schema: REMEDIATE_SCHEMA })
  finalVerify = await agent(VERIFY_PROMPT, { label: 'final-verify', phase: 'Remediate', schema: VERIFY_SCHEMA })
}

return {
  stage: 'complete',
  baseline,
  reviewDimensions: reviewed.flat().filter(Boolean).map((x) => ({ id: x.id, severity: x.severity, verdict: x.verdict })),
  synthesis: synth,
  mustFixCount: mustFix.length,
  remediation,
  finalVerify,
}
