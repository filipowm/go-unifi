export const meta = {
  name: 'wave2-minor-hardening',
  description: 'Wave 2 post-review hardening: high-value MINOR/regression fixes the gated remediation (blocker/major-only) skipped. unifi-package only; implement -> verify.',
  phases: [
    { title: 'Harden', detail: 'ARCH-10 ServerError fidelity + restore CreateUser nested-meta; ARCH-11 size-cap explicit error+test; testhelpers race mutex; TEST-15 VersionContext error subtest' },
    { title: 'Verify', detail: 'build/test/-race/lint/regen+mock idempotent with fix loop' },
  ],
}

const ROOT = '/Users/filipowm/Documents/dev/workspaces/unifi/go-unifi'
const PATHFIX = 'export PATH="/opt/homebrew/opt/go/bin:$PATH"'
const MOQ = '/Users/filipowm/go/bin/moq'
const REGEN = `cd ${ROOT}/unifi && ${PATHFIX} && go run ../codegen -version-base-dir=../codegen 9.5.21`
const MOCKREGEN = `cd ${ROOT}/unifi && ${PATHFIX} && ${MOQ} -out client_mock.generated.go . Client`

const COMMON = `Repo root: ${ROOT}. go-unifi = Go client for the UniFi controller API. Branch chore/review-1.11.0.
HARD RULES:
- NEVER hand-edit *.generated.go. These fixes are all in HAND-WRITTEN unifi/*.go and *_test.go — none should require codegen/regen. If you somehow need a generated change, fix codegen source + \`${REGEN}\` (+ \`${MOCKREGEN}\` if the interface changes) — but you should NOT need to here.
- FORBIDDEN: NEVER run ANY git command (add/commit/rm/reset/stash/restore/checkout). The orchestrator owns ALL git. The ONLY git allowed is read-only \`git diff --stat\` for your own verification.
- Go uses TABS; run \`${PATHFIX}; gofmt -w <files>\` on every .go file you change. Lines <200 cols. context.Context first. Wrap errors with %w.
- Tests: testify; table-driven map[string]struct{} with t.Run + t.Parallel() on outer AND subtests; httptest round-trips; REUSE unifi/testhelpers_test.go (newControllerServer, controllerServer.client()/clientUserPass(), apiV1Path/apiV2).
- ALL go/gofmt/golangci-lint commands MUST prepend: ${PATHFIX}
- TDD: failing test first where applicable, then fix, then green. Do NOT weaken assertions.
- After regen the .unifi-version file may be rewritten to 9.5.21 — IGNORE it (do not restore/commit; orchestrator owns it).
Your final message MUST be the structured object only.`

const REPORT = {
  type: 'object', additionalProperties: false,
  required: ['lane', 'overallStatus', 'findings', 'verifyOutput', 'breakingChanges', 'blockers'],
  properties: {
    lane: { type: 'string' },
    overallStatus: { type: 'string', enum: ['all-green', 'partial', 'blocked'] },
    findings: { type: 'array', items: { type: 'object', additionalProperties: false,
      required: ['id', 'status', 'filesChanged', 'notes'],
      properties: { id: { type: 'string' }, status: { type: 'string', enum: ['done', 'partial', 'skipped', 'blocked'] }, filesChanged: { type: 'array', items: { type: 'string' } }, testsAdded: { type: 'array', items: { type: 'string' } }, notes: { type: 'string' } } } },
    breakingChanges: { type: 'array', items: { type: 'object', additionalProperties: false, required: ['what', 'migration'], properties: { id: { type: 'string' }, what: { type: 'string' }, migration: { type: 'string' } } } },
    verifyOutput: { type: 'string' }, blockers: { type: 'string' },
  },
}
const VERIFY_SCHEMA = {
  type: 'object', additionalProperties: false,
  required: ['allGreen', 'checks', 'summary'],
  properties: {
    allGreen: { type: 'boolean' },
    checks: { type: 'array', items: { type: 'object', additionalProperties: false, required: ['name', 'passed', 'detail'], properties: { name: { type: 'string' }, passed: { type: 'boolean' }, detail: { type: 'string' } } } },
    summary: { type: 'string' },
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

const VERIFY_PROMPT = `You are the verification gate for the Wave 2 minor-hardening pass. Repo root: ${ROOT}. Do NOT edit. Do NOT run mutating git. Prepend ${PATHFIX}. IGNORE any .unifi-version diff.
1. go build ./...
2. golangci-lint run   (expect 0 issues)
3. go test ./unifi/...
4. go test ./unifi/ -race
5. go test -short ./codegen/...   (offline)
6. go vet ./codegen/...
7. regen-reproducible (IDEMPOTENCY, NOT vs HEAD): H1=\`find ${ROOT}/unifi -name '*.generated.go' | sort | xargs shasum | shasum\`; run \`${REGEN}\`; H2=same. PASS iff H1==H2 (record both hashes).
8. mock-in-sync (IDEMPOTENCY): M1=\`shasum ${ROOT}/unifi/client_mock.generated.go\`; run \`${MOCKREGEN}\`; M2=same. PASS iff M1==M2.
allGreen = every check passed. Return the VERIFY object only.`

phase('Harden')
const impl = await agent(`${COMMON}

LANE unifi:minor-hardening. Apply these FIVE high-value review fixes the gated remediation skipped (all in the unifi package). Read the exact current code before editing.

(1) ARCH-10 ServerError fidelity (unifi/requests.go + unifi/unifi_errors.go): A soft HTTP-200 with top-level meta.rc=="error" is surfaced via metaEnvelopeError(body)->Meta.error(), which builds a *ServerError populating ONLY ErrorCode+Message, leaving StatusCode/RequestMethod/RequestURL zero — so ServerError.Error() renders "Server error (0) for  : <msg>", losing the HTTP context the non-2xx HandleError path populates. FIX: enrich the soft-error *ServerError with the response context. metaEnvelopeError currently takes only (body []byte); change it to also receive the *http.Response (e.g. metaEnvelopeError(resp *http.Response, body []byte)) — or have decodeResponseBody set StatusCode=resp.StatusCode, RequestMethod=resp.Request.Method, RequestURL=resp.Request.URL.String() on the returned *ServerError before returning it. Guard against resp.Request being nil. Keep errors.Is(err, ErrNotFound)==false for these (a soft rc:error is NOT a 404). Add/extend a test asserting the surfaced *ServerError carries StatusCode==200, the request method, and the URL (not zero/empty).

(2) ARCH-10 restore CreateUser nested-meta check (unifi/user.go + unifi/user_wrappers_test.go): The centralized handleResponse check only probes the TOP-LEVEL meta envelope. The stamgr group-create response is NESTED: {meta:{rc}, data:[{Meta:{rc}, data:[...]}]}. The Wave-2 b1 lane REMOVED CreateUser's former per-object check (respBody.Data[0].Meta.error()), so a nested rc=="error" with empty inner data now silently falls through to the len(inner)!=1 guard and returns ErrNotFound instead of the real server message — a regression AND an unnecessary breaking change. FIX: RESTORE the nested check in CreateUser, placed AFTER the \`if len(respBody.Data) != 1 { return nil, ErrNotFound }\` guard (~line 58) and BEFORE the inner \`if len(respBody.Data[0].Data) != 1\` guard (~line 65): \`if err := respBody.Data[0].Meta.error(); err != nil { return nil, err }\`. Confirm the inner object type actually has a Meta field of type Meta (it does per the test fixture {"Meta":{"rc":...}}). This is INTENTIONAL business logic now (top-level handled centrally, nested handled here) — add a brief comment saying so and that it resolves the old TODO without losing the nested soft-error. UPDATE unifi/user_wrappers_test.go: the case currently asserting that a nested data[0].Meta.rc=="error" + empty inner yields ErrNotFound must change to assert it yields a *ServerError carrying the inner rc/msg (errors.As to *ServerError, NOT errors.Is ErrNotFound). Keep the genuine empty-inner-without-error case (inner len 0, no meta error) still -> ErrNotFound. Do NOT weaken any other assertion. NOTE for the orchestrator (put in notes): this ELIMINATES the previously-documented "ARCH-10-user" breaking change — CreateUser nested soft-error behavior is now preserved, not broken.

(3) ARCH-11 size-cap explicit overflow error + test (unifi/requests.go + unifi/requests_test.go): decodeResponseBody reads via io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize)) (64 MiB). A body exceeding the cap is silently truncated and then fails json.Decode with a generic "unable to decode body" — the cap is undiagnosable. FIX: read with io.LimitReader(resp.Body, maxResponseBodySize+1); if len(body) > maxResponseBodySize, return an explicit error like fmt.Errorf("response body exceeded %d bytes", maxResponseBodySize) BEFORE attempting decode. Add a test that injects a SMALL cap or feeds a body larger than maxResponseBodySize and asserts the explicit "exceeded N bytes" error (NOT a json error). To keep the test cheap without allocating 64 MiB: make maxResponseBodySize a package var (not const) so the test can temporarily lower it (save/restore via defer), OR factor the limit into decodeResponseBody as a parameter/field. Pick the lower-risk option that keeps production default at 64 MiB and existing callers unchanged. Ensure the meta-probe (ARCH-10) uses the SAME already-read body bytes (no second read).

(4) testhelpers request-recording race (unifi/testhelpers_test.go): controllerServer records each served request by appending to an unsynchronized slice cs.requests from inside the httptest handler goroutine (~line 51), while tests read it via lastRequest()/cs.requests. FIX: add a sync.Mutex to controllerServer; lock around the append in the handler AND around every read (lastRequest, any requests accessor, and len/index reads). Make the helper concurrency-safe by construction. Verify with \`go test ./unifi/ -race -count=5\` that the previously race-prone paths are clean. Do NOT change the helper's public shape used by existing tests beyond adding the mutex + guarded accessors (if tests read cs.requests directly, add a guarded method like requestCount()/requestsSnapshot() and migrate those reads).

(5) TEST-15 VersionContext slow-path error subtest (unifi/context_test.go): VersionContext's cancellation + happy paths are covered, but the NON-cancellation fetch-error path (sysinfo endpoint returns 500 -> GetSystemInformationContext errors -> VersionContext must surface empty string + the error) is not. Add a subtest: an httptest sysinfo endpoint that 500s; call VersionContext(context.Background()); assert "" AND require.Error with the *ServerError surfaced (mirror TestVersion's error-swallow case but for the ctx variant which RETURNS the error). Use testhelpers.

After ALL: ${PATHFIX}; gofmt -w every changed .go file; go vet ./unifi/...; go test ./unifi/...; go test ./unifi/ -race -count=5; golangci-lint run ./unifi/.... These are hand-written changes only — confirm regen+mock stay idempotent (no *.generated.go should change). Set lane="unifi:minor-hardening". Report each as a finding (id = the ARCH/TEST id), and in breakingChanges note the REMOVAL of the ARCH-10-user breaking change (CreateUser nested behavior restored).`,
  { label: 'unifi:minor-hardening', phase: 'Harden', schema: REPORT })

phase('Verify')
let verify = await agent(VERIFY_PROMPT, { label: 'verify', phase: 'Verify', schema: VERIFY_SCHEMA })
let attempts = 0
while (!verify.allGreen && attempts < 3) {
  attempts++
  await agent(`${COMMON}\nMinor-hardening verification FAILED. Prepend ${PATHFIX}. Failing:\n${JSON.stringify(verify.checks.filter(c => !c.passed), null, 2)}\nFix with MINIMAL correct changes preserving intent. Hand-written unifi only; do NOT hand-edit *.generated.go. Keep tabs+gofmt; do NOT weaken tests. Re-run failing checks. Return REMEDIATE object.`, { label: `fix#${attempts}`, phase: 'Verify', schema: REMEDIATE_SCHEMA })
  verify = await agent(VERIFY_PROMPT, { label: `re-verify#${attempts}`, phase: 'Verify', schema: VERIFY_SCHEMA })
}

return { stage: verify.allGreen ? 'complete' : 'verify-failed', impl, verify }
