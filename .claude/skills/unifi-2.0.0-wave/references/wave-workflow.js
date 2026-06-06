// Template: a go-unifi 2.0.0 migration wave.
//
// One Workflow handles N non-overlapping issues. Each issue is an independent pipeline item, so the gate
// (Implement -> Verify -> Review -> Remediate -> re-Verify) runs per-issue and a fast issue isn't blocked by
// a slow one. The process contract is docs/2.0.0/README.md.
//
// HOW TO LAUNCH: build the issues array (see WAVE_ISSUES shape below) by reading each issue with `gh issue
// view <N> --json number,title,body` and extracting plan / acceptance / edgeCases / the files it will touch.
// Pass that array as the Workflow tool's `args`. Do NOT launch with an empty array — the run will bail.
//
// WORKTREE MODEL (important — this is why it's correct): we do NOT use framework `isolation:'worktree'`,
// because that gives every agent a *fresh* worktree off feat/2.0.0 and the review/remediate agents would
// never see the implementer's branch. Instead each issue owns ONE explicit worktree at a convention path
// `../gu-2.0.0-<slug>`, created by the implement stage and reused (via `git -C <path>`) by remediate and read
// by review. Worktrees share the repo's object DB + refs, and each has its own index, so the issues run
// concurrently without colliding — provided the wave is non-overlapping (verify before launching). The main
// loop opens the PR from each worktree, then `git worktree remove`s it.

export const meta = {
  name: 'unifi-2.0.0-wave',
  description: 'Run a go-unifi 2.0.0 migration wave: per-issue Implement -> Verify -> Review -> Remediate -> re-Verify, PRs into feat/2.0.0',
  phases: [
    { title: 'Implement', detail: 'one explicit worktree+branch per non-overlapping issue' },
    { title: 'Review', detail: 'architect ‖ test-lead per issue (read-only on the branch)' },
    { title: 'Remediate', detail: 'apply blocker/major findings only, then re-verify' },
  ],
}

// EDIT THIS (or pass via the Workflow tool's `args`): one entry per issue. Confirm files[] are disjoint
// across entries BEFORE launch — overlap means merge conflicts into feat/2.0.0 even with separate worktrees.
const WAVE_ISSUES = args && args.length ? args : [
  // { number: 123, title: '...', slug: 'openapi-dns', type: 'feat', scope: 'openapi',
  //   plan: '...', acceptance: '...', edgeCases: '...',
  //   touchesCodegen: true,                       // -> Verify must regenerate + check golden diffs
  //   files: ['unifi/dns.go', 'codegen/...'] },   // for the disjointness proof
]

const wt = (issue) => `../gu-2.0.0-${issue.slug}`
const branch = (issue) => `${issue.type}/2.0.0-${issue.slug}`

const CONTRACT = `
You are implementing ONE issue of a go-unifi 2.0.0 migration wave. The process contract is
docs/2.0.0/README.md — read it. Hard rules:
- Base branch is feat/2.0.0. NEVER touch main.
- The GitHub issue body is the contract (plan, acceptance criteria, edge cases). Honor it exactly.
- NEVER hand-edit *.generated.go (they say DO NOT EDIT). Change output via codegen/customizations.yml or a
  hand-written sibling .go (see codegen/CLAUDE.md). Generated CRUD is private; public wrappers are siblings.
- Go style: tabs, max line 200, ctx context.Context first arg, wrap errors with %w; gofumpt+goimports+gci.
- Write clean, idiomatic Go: KISS/DRY/SOLID, clear separation of concerns, small focused types/functions.
- Comments are SHORT — max 2 lines — explaining WHY, not what. Only exceed that for genuinely complex logic
  that can't be made self-evident by naming/structure. Don't narrate obvious code.
- Docs sync IN THE SAME change: docs/, root README, relevant CLAUDE.md, .claude/rules/. API breaking changes
  go in docs/2.0.0/breaking_changes.md in its established format (signature vs behavioral, migration snippet,
  impact).
- Conventional commits.
`

const verifyBlock = (issue) => `
Run the full quality gate IN THE WORKTREE and fix until green (this is the fix loop — do not stop while red):
  export PATH="/opt/homebrew/opt/go/bin:$PATH"   # golangci-lint needs Homebrew Go, not asdf's
  git -C ${wt(issue)} ... # all work happens in this worktree
  (cd ${wt(issue)} && go build ./... && go test -cover -coverprofile=coverage.out -covermode atomic ./... && golangci-lint run)
${issue.touchesCodegen ? `This issue changes codegen: also run \`go generate unifi/codegen.go\` (or \`make generate\`) and confirm the golden type-diff is clean — a codegen change that doesn't regenerate is NOT done.` : ''}
COMMIT all changes to ${branch(issue)} with conventional messages (review reads the committed branch).
Report the final command output verbatim.
`

const GATE_SCHEMA = {
  type: 'object', additionalProperties: false,
  properties: {
    branch: { type: 'string' },
    worktree: { type: 'string' },
    summary: { type: 'string', description: 'what changed' },
    buildPassed: { type: 'boolean' },
    testPassed: { type: 'boolean' },
    lintPassed: { type: 'boolean' },
    docsSynced: { type: 'array', items: { type: 'string' }, description: 'doc files updated' },
    breakingChanges: { type: 'string', description: 'breaking_changes.md entry, or "none"' },
  },
  required: ['branch', 'worktree', 'summary', 'buildPassed', 'testPassed', 'lintPassed', 'docsSynced', 'breakingChanges'],
}

const REVIEW_SCHEMA = {
  type: 'object', additionalProperties: false,
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object', additionalProperties: false,
        properties: {
          severity: { type: 'string', enum: ['blocker', 'major', 'minor', 'nit'] },
          issue: { type: 'string' }, fix: { type: 'string' },
        },
        required: ['severity', 'issue', 'fix'],
      },
    },
    verdict: { type: 'string' },
  },
  required: ['findings', 'verdict'],
}

if (!WAVE_ISSUES.length) {
  log('No issues supplied. Populate WAVE_ISSUES (or pass them via the Workflow `args`) and relaunch.')
  return { error: 'empty-wave' }
}

const results = await pipeline(
  WAVE_ISSUES,

  // Stage 1: Implement + Verify (fix loop) in an explicit, reusable worktree; commit to the branch.
  (issue) => agent(
    `${CONTRACT}\n\nISSUE #${issue.number}: ${issue.title}\nPLAN:\n${issue.plan}\nACCEPTANCE:\n${issue.acceptance}\nEDGE CASES:\n${issue.edgeCases}\n\nFirst create the worktree+branch:\n  git worktree add -b ${branch(issue)} ${wt(issue)} feat/2.0.0\nDo ALL work inside ${wt(issue)}. Implement the change, keep docs in sync.\n${verifyBlock(issue)}`,
    { label: `impl:#${issue.number}`, phase: 'Implement', schema: GATE_SCHEMA },
  ).then(r => ({ issue, gate: r })),

  // Stage 2: Review — architect ‖ test-lead, read-only on the committed branch (shared refs; no new worktree).
  // Short-circuit: if implement never went green, reviewing a broken branch is wasted — skip to reporting.
  ({ issue, gate }) => {
    if (!gate || !gate.buildPassed || !gate.testPassed || !gate.lintPassed) {
      log(`#${issue.number}: implement gate RED — skipping review/remediate.`)
      return { issue, gate, reviews: [], skipped: 'red-implement' }
    }
    return parallel([
    () => agent(
      `You are a software architect reviewing issue #${issue.number} of the go-unifi 2.0.0 migration. Read-only: inspect the diff with \`git -C ${gate ? gate.worktree : wt(issue)} diff feat/2.0.0...HEAD\` (do not modify anything). Contract: docs/2.0.0/README.md.\n\nJudge the design, not just correctness:\n- KISS, DRY, SOLID — is it the simplest thing that works, free of duplication and over-engineering?\n- Code structure & separation of concerns — does each package/type/function have one clear responsibility?\n- Maintainability & testability — is it easy to change and to test in isolation (seams, no hidden coupling)?\n- Ease of understanding — would a new contributor grok it quickly? Clear naming, low cognitive load.\n- Design patterns & clean code — appropriate (not forced) patterns; no smells.\n- Idiomatic Go — proper use of structs/interfaces/errors/zero values; small interfaces; accept-interfaces-return-structs where it fits; context first.\n- Developer experience for LIBRARY CONSUMERS — is the public API intuitive, consistent, hard to misuse, well-typed? This is a published Go library; its ergonomics matter most.\n- Comments — short (≤2 lines) and explain WHY; flag noisy/obvious comments AND missing rationale on complex bits.\nAlso: API design, the hybrid legacy/OpenAPI seam (APIStyle), version gating (floor 9.0.114, OpenAPI from 10.1.68), no hand-edited generated code, breaking-change handling.\n\nWHAT CHANGED:\n${gate ? gate.summary : '(implementation failed)'}`,
      { label: `arch:#${issue.number}`, phase: 'Review', schema: REVIEW_SCHEMA },
    ),
    () => agent(
      `You are a test lead reviewing issue #${issue.number} of the go-unifi 2.0.0 migration. Read-only: inspect the diff with \`git -C ${gate ? gate.worktree : wt(issue)} diff feat/2.0.0...HEAD\`. Focus: test coverage for the change and its edge cases (${issue.edgeCases}), that acceptance criteria are actually tested, golden type-diffs for codegen changes, no flaky/over-mocked tests, docs accuracy. Contract: docs/2.0.0/README.md and .claude/rules/testing.md.\n\nWHAT CHANGED:\n${gate ? gate.summary : '(implementation failed)'}`,
      { label: `test:#${issue.number}`, phase: 'Review', schema: REVIEW_SCHEMA },
    ),
    ]).then(reviews => ({ issue, gate, reviews: reviews.filter(Boolean) }))
  },

  // Stage 3: Remediate (gated on blocker/major) in the SAME worktree, then re-Verify. Skips cleanly if none.
  ({ issue, gate, reviews }) => {
    if (!reviews.length) return { issue, gate, reviews, remediated: false }   // red implement or no findings upstream
    const actionable = reviews.flatMap(r => r.findings.filter(f => f.severity === 'blocker' || f.severity === 'major'))
    if (!actionable.length) return { issue, gate, reviews, remediated: false }
    return agent(
      `${CONTRACT}\n\nISSUE #${issue.number}. Work in the EXISTING worktree ${gate.worktree} on branch ${gate.branch} (do NOT create a new worktree). Apply ONLY these blocker/major review findings, then re-run the full gate until green. Minor/nits: leave a note, do not fix here.\n\nFINDINGS:\n${actionable.map((f, i) => `${i + 1}. [${f.severity}] ${f.issue}\n   FIX: ${f.fix}`).join('\n')}\n${verifyBlock(issue)}`,
      { label: `remediate:#${issue.number}`, phase: 'Remediate', schema: GATE_SCHEMA },
    ).then(g => ({ issue, gate: g || gate, reviews, remediated: true }))
  },
)

const clean = results.filter(Boolean)
return {
  wave: clean.map(r => ({
    issue: r.issue.number,
    branch: r.gate && r.gate.branch,
    worktree: r.gate && r.gate.worktree,
    green: !!(r.gate && r.gate.buildPassed && r.gate.testPassed && r.gate.lintPassed),
    docsSynced: r.gate && r.gate.docsSynced,
    breakingChanges: r.gate && r.gate.breakingChanges,
    remediated: r.remediated,
    deferredFindings: r.reviews.flatMap(rv => rv.findings.filter(f => f.severity === 'minor' || f.severity === 'nit')),
  })),
  // NEXT (main loop, only for green issues): from each worktree, `gh pr create --base feat/2.0.0` (add the
  // `breaking` label if breakingChanges != "none"); after merge `gh issue close <N>`; then `git worktree
  // remove <worktree>`. Auto-close does not fire for PRs into feat/2.0.0.
}
