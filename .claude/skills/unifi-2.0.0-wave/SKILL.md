---
name: unifi-2.0.0-wave
description: >-
  Execute a go-unifi 2.0.0 migration wave end-to-end via a Claude Code Workflow. Use this whenever the user
  wants to DO 2.0.0 implementation work — "run a 2.0.0 wave", "next batch", "work issue #N", "implement the
  OpenAPI migration", "start the codegen retarget", "ship the next 2.0.0 PRs", or names any issue under epic
  #117 / milestone 2.0.0 and asks you to build it. Trigger even when the user doesn't say the word "wave":
  any request to implement, migrate, refactor, or PR something for 2.0.0 means a wave. Do NOT trigger for
  user-facing migration questions ("how do I upgrade to 2.0.0?") — that's the migration guide, not this.
argument-hint: "issue number(s) or what to work on, e.g. '#123 #124' or 'next OpenAPI batch'"
---

Run ONE 2.0.0 migration wave: turn selected GitHub issues into small, non-overlapping PRs against
`feat/2.0.0`, with the quality gate running **inside a Workflow** (never the main loop). Scope: $ARGUMENTS.

The process contract is **`docs/2.0.0/README.md`** — it is the source of truth for branching, the gate, docs
sync, and the per-PR checklist. This skill is the *executable* layer on top of it. When the two ever
disagree, the README wins; fix this skill to match.

## The prime directive: explore first, assume nothing

A wave is expensive and hard to unwind — it creates branches, runs many subagents, and opens PRs. Getting
the scope, slicing, and overlap wrong wastes all of that and creates merge conflicts. So **before launching
anything, interrogate the user.** Treat the user as the authority on intent and the issues as the authority
on detail; your job is to surface every gap between them and resolve it *with the user*, not to paper over
it with a guess.

**Hard rule: make NO consequential decision by assumption. Use the `AskUserQuestion` tool.** That includes —
which issues are in the wave, how each is sliced, what "done" means, whether two issues overlap or depend on
each other, whether a missing plan/criterion should be your draft or theirs, and whether to launch. If you
catch yourself thinking "they probably mean…", "I'll assume…", or "the obvious default is…" — stop and ask.
Batch related questions (2–4 at a time) so it's a brisk grilling, not an interrogation lamp. Keep going in
rounds until there is genuinely nothing left to clarify. Only the truly mechanical (exact git command
syntax, file formatting) needs no asking.

## Step 0: Load the contract and the candidate work

Read these before asking anything — informed questions beat blank ones:

```bash
cat docs/2.0.0/README.md                 # the process contract
git branch --show-current                # confirm you can branch off feat/2.0.0
gh issue view 117 --json title,body      # the epic, for context
```

Resolve the candidate issues from `$ARGUMENTS`. For each, pull the body — **the issue is the contract**
(description, plan, acceptance criteria, edge cases):

```bash
gh issue view <N> --json number,title,body,labels,milestone,state
```

If `$ARGUMENTS` is vague ("next batch"), list open milestone-2.0.0 issues and bring candidates to the user:

```bash
gh issue list --milestone 2.0.0 --state open --json number,title,labels
```

## Step 1: Grill the user (the core of this skill)

Now run the interview. Ask in rounds with `AskUserQuestion`; do not move to Step 2 until each item below is
**explicitly settled by the user** (not by you). Probe at least these, and chase down anything an answer
exposes:

- **Wave membership** — exactly which issues are in this wave? (Confirm the set; don't infer it.)
- **Skeleton precondition** — has the architecture/scaffolding skeleton already landed on `feat/2.0.0`? If
  not, that skeleton IS the wave and must run alone first (everything else branches off it). Ask; don't
  assume it's there. (Sanity-check by looking for the expected dirs/files the skeleton issue defined, then
  confirm with the user.)
- **Per-issue scope & slicing** — is each issue genuinely one small, cohesive change? Should any be split or
  merged? Where are the boundaries?
- **Plan completeness** — does each issue body have a real implementation plan? If not, who writes it — you
  draft and they approve, or they'll fill the issue? (Issues are the source of truth; a vague issue must be
  fixed *before* implementation, e.g. via the `unifi-issue-author` skill.)
- **Acceptance criteria** — what does "done" mean for each, concretely and testably? Surface anything the
  issue leaves implicit.
- **Edge cases** — version gating (floor 9.0.114, OpenAPI from 10.1.68), dual-shape resources, error mapping,
  empty/error envelopes, backward compat. Ask which apply per issue.
- **Overlap & dependencies** — do any two issues touch the same files/dirs, or does one depend on another's
  output? This decides whether they can run in the same parallel wave (see Step 2). Get the user's read;
  don't just diff plans silently.
- **Breaking changes** — will any issue change public API/behavior? If so it needs the `breaking` label and a
  `docs/2.0.0/breaking_changes.md` entry. Confirm per issue.
- **Codegen impact** — does any issue change generated output? Those need `go generate` + a clean golden
  type-diff in Verify, and must edit `codegen/customizations.yml`, never `*.generated.go`.

When you believe you understand the wave, **play it back** to the user (one issue per line: slug, scope,
acceptance, edge cases, files touched, breaking?, codegen?) and ask them to confirm or correct before any
launch. This read-back is mandatory.

## Step 2: Prove the wave is non-overlapping — and confirm it with the user

The cardinal rule (README §1): issues in one parallel wave must touch **disjoint files/dirs** so branches
don't conflict at merge. Build the file/dir set per issue (from the plans you settled in Step 1) and
intersect them. Watch the shared-file traps — `codegen/customizations.yml`, root `README.md`, `docs/`,
`.claude/rules/` are co-touched by many issues; sub-file overlap still conflicts.

Present the overlap analysis and your proposed wave grouping to the user via `AskUserQuestion`. If anything
intersects or one issue depends on another, ask whether to **sequence** them (separate single-issue waves)
or **merge** them — do not silently pick. Only fan out a parallel wave the user has signed off on.

## Step 3: Launch the wave Workflow

Everything runs through the Workflow tool (multi-phase, subagent-driven). The gate per issue is
**Implement → Verify (fix loop) → Review (architect ‖ test-lead) → Remediate (gated) → re-Verify**, exactly
as the README diagram specifies. Do **not** run the gate yourself in the main loop.

Use the template in `references/wave-workflow.js` — read it. Build the `WAVE_ISSUES` array from the issues
you settled with the user: for each, `{ number, title, slug, type, scope, plan, acceptance, edgeCases,
touchesCodegen, files }`. **Pass that array as the Workflow tool's `args`** (the template bails on an empty
wave — this is the #1 way to misfire). Launch via the Workflow tool. The template manages one explicit
worktree per issue at `../gu-2.0.0-<slug>` so implement/review/remediate share the branch.

Things the workflow enforces (all from the README — keep them in the agent prompts): branch
`<type>/2.0.0-<slug>` off `feat/2.0.0`, never `main`; never hand-edit `*.generated.go`; Verify =
`go build ./...` + full `go test` + `golangci-lint run` green with a fix loop (heads-up: `golangci-lint`
needs Homebrew Go on PATH — `export PATH="/opt/homebrew/opt/go/bin:$PATH"`); codegen issues also run
`go generate` + golden diff; docs synced in the same PR; breaking changes → `breaking_changes.md`;
conventional commits.

The **architect reviewer** judges design, not just correctness: KISS/DRY/SOLID, clean structure and
separation of concerns, maintainability/testability, ease of understanding, appropriate design patterns and
clean code, idiomatic Go, and — most of all — **developer experience for consumers of this library** (the
public API must be intuitive, consistent, and hard to misuse). Implementation keeps **comments short (≤2
lines, explaining WHY)**, going longer only for genuinely complex logic.

## Step 4: Open PRs and close issues

For each issue whose gate went green (confirm with the user before opening PRs if there's any doubt):

```bash
gh pr create --base feat/2.0.0 --title "<type>(<scope>): <summary> (#<N>)" --body "..."   # never --base main
# add --label breaking if the issue changed public API/behavior
```

PRs targeting `feat/2.0.0` do **not** auto-close their issue (auto-close fires only on merge to the default
branch). After the PR merges, close it manually, then remove the worktree:

```bash
gh issue close <N> --comment "Done in <PR-url>."
git worktree remove ../gu-2.0.0-<slug>
```

Reserve `Closes #` keywords for the eventual `feat/2.0.0` → `main` PR.

## Step 5: Report

Summarize per issue: PR link, gate result (tests/lint green), docs synced, breaking changes recorded,
deferred non-blocker findings (with where they were logged). If any issue's gate stayed red, say so plainly
with the failing output — never claim a wave passed when it didn't.

## After all waves

The final whole-codebase review (README §7) is its own wave: architect + test-lead over the entire
`feat/2.0.0` diff, run as a Workflow, before opening the `feat/2.0.0` → `main` PR. Use this same skill with
that as the scope — and grill the user on what "ready for main" means first.

## Per-PR checklist (mirror of README §6 — verify before reporting done)

- [ ] Wave scope, slicing, overlap, and launch were each confirmed by the user (not assumed).
- [ ] Issue has description, plan, acceptance criteria, edge cases; links #117; milestone 2.0.0; labels set.
- [ ] Branched off `feat/2.0.0`; PR targets `feat/2.0.0` (never `main`).
- [ ] Change is small, cohesive, disjoint from sibling PRs in the wave.
- [ ] No hand-edits to `*.generated.go`.
- [ ] `go build`, full `go test`, `golangci-lint run` all green (+ `go generate`/golden diff if codegen).
- [ ] In-workflow gate completed: Implement → Verify → Review (architect ‖ test-lead) → Remediate → re-Verify.
- [ ] Docs synced in this PR; breaking changes in `docs/2.0.0/breaking_changes.md`.
- [ ] Conventional commits + PR title; issue closed manually after merge; worktree removed.
