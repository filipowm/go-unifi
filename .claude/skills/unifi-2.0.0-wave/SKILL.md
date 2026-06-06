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
model: opus
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

## Step 0: Find candidate work — one cheap query, NO bodies

Don't read issue bodies to figure out what's workable — that floods context. Run the finder once; it digests
every open milestone-2.0.0 issue into a compact table (status + unmet deps, **never** descriptions). Status is
computed fresh from the `Depends on #N` lines + each dep's open/closed state — there's no label to trust or
maintain:

```bash
git branch --show-current                                          # confirm you can branch off feat/2.0.0
${CLAUDE_SKILL_DIR}/references/find-candidates.sh
```

Columns: **READY** (eligible now), **BLOCKED** (has an unmerged `Depends on #N`, shown in `BLOCKED-BY`),
**CLAIMED** (`in-progress`/`in-review` — owned by a running/open wave). Resolve `$ARGUMENTS` against this
table; if it's vague ("next batch"), bring the READY rows to the user. **Do NOT pull any issue body yet** —
you only need a body once an issue is actually picked (Step 1). That deferral is the whole point of this
rewrite: scope from the table, read deeply only what you'll build.

Read the contract (`docs/2.0.0/README.md`) for the rules; pull the epic (`gh issue view 117`) only if you
genuinely need 2.0.0 context — it's not required to scope a wave.

**If no issue is READY, there is no wave — report and stop.** The table already says what blocks what (e.g.
"#121 BLOCKED by #119,#120"); relay that and stop. Never launch on an empty set, never strip a claim or treat
an unmerged dep as done to manufacture work. The user's options: wait for the blocker/claim to clear, pick a
READY issue, or — only if a claim is genuinely stale — confirm that explicitly before reclaiming it.

## Step 1: Grill the user (the core of this skill)

Now run the interview. Ask in rounds with `AskUserQuestion`; do not move to Step 2 until each item below is
**explicitly settled by the user** (not by you). Probe at least these, and chase down anything an answer
exposes:

- **Wave membership** — exactly which issues are in this wave? (Confirm the set from the Step 0 table; don't
  infer it.) **Only once the set is provisional, pull bodies for those issues** — `gh issue view <N> --json
  title,body,labels` — the body is the contract for every question below. Never pull bodies for issues you
  won't build; that's the context bloat this rewrite kills.
- **Per-issue scope & slicing** — is each issue genuinely one small, cohesive change? Should any be split or
  merged? Where are the boundaries?
- **Plan completeness** — does each issue body have a real implementation plan? If not, who writes it — you
  draft and they approve, or they'll fill the issue? (Issues are the source of truth; a vague issue must be
  fixed *before* implementation, e.g. via the `unifi-issue-author` skill.)
- **Acceptance criteria** — what does "done" mean for each, concretely and testably? Surface anything the
  issue leaves implicit.
- **Edge cases** — which of the README §1 edge cases apply per issue (version gating, dual-shape resources,
  error mapping, empty/error envelopes, backward compat)? Ask per issue. (Constants live in README §1.)
- **Overlap & dependencies** — deps come from the Step 0 table (`BLOCKED-BY`), not from re-reading bodies; the
  finder already excluded anything with an unmerged dep. What's left for you: do any two *selected* issues
  touch the same files/dirs (file-level overlap the finder can't see)? Two rules, both enforced in Step 2: (1)
  a blocker and its dependent may **never** share a wave; (2) an issue only joins a wave once **all** its deps
  are **merged/closed** (i.e. it shows READY). Get the user's read on overlap; don't just diff plans silently.
- **Breaking changes** — will any issue change public API/behavior? If so it needs the `breaking` label and a
  `docs/2.0.0/breaking_changes.md` entry. Confirm per issue.
- **Codegen impact** — does any issue change generated output? Those need `go generate` + a clean golden
  type-diff in Verify, and follow the generated-code rule (README §4): edit `codegen/customizations.yml`,
  never `*.generated.go`.

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

**Dependency gate (hard):** the Step 0 finder already did the heavy lifting — anything with an unmerged
`Depends on #N` came back **BLOCKED** and is off the table. So the gate here is simply: only **READY** issues
enter a wave, and never place a blocker and its dependent in the same wave. The template hard-fails on an
intra-wave dependency as a backstop, but you shouldn't reach it — a BLOCKED issue never gets selected. When in
doubt, sequence: smaller, dependency-clean waves beat a blocked one.

**Bundling into one PR (optional):** default is one issue → one worktree → one PR. If two or more issues are
so tightly coupled they belong in a SINGLE PR, give them a shared `groupSlug` in the wave array — they then
collapse into one worktree/branch/PR (a "unit") that closes all of them. Only group when the user confirms
the issues are genuinely one cohesive change; independent issues must stay separate per the one-issue-one-PR
contract. A grouped unit still must be disjoint from every OTHER unit in the wave. Sharing a worktree any
other way (multiple branches in one tree) is not supported — it would serialize the units and defeat the
per-unit parallelism.

## Step 2.5: Claim the wave (race guard for parallel runs)

Parallel runs are only safe if each issue is taken once. Before launching, **claim every issue in the wave**
so a concurrent run (or a future you) sees it's taken. Ensure the labels exist, add `in-progress`, then
re-read to confirm:

```bash
gh label create in-progress --color FBCA04 --description "Claimed by a running 2.0.0 wave" 2>/dev/null || true
gh label create in-review  --color 0E8A16 --description "2.0.0 wave PR open, awaiting review/merge" 2>/dev/null || true

for N in <wave issue numbers>; do gh issue edit "$N" --add-label in-progress; done
gh issue list --milestone 2.0.0 --search 'label:in-progress' --json number,title   # confirm your set is labeled
```

Label add is idempotent, so it *advertises* the claim but can't by itself win a truly simultaneous race —
the **hard lock is the worktree/branch existence guard inside the workflow** (Step 3): the first run to
create `../gu-2.0.0-<slug>` / `<type>/<issue#>-<slug>` owns that unit, and a second run that finds them existing
returns `collision:true` and skips it instead of clobbering. Labels make the claim visible; the guard makes
it safe. If a launch is aborted or a unit's gate stays red and no PR opens, **release the claim**:
`gh issue edit <N> --remove-label in-progress`.

## Step 3: Launch the wave Workflow

Everything runs through the Workflow tool (multi-phase, subagent-driven). The gate per issue is
**Implement → Verify (fix loop) → Review (architect ‖ test-lead) → Remediate (gated) → re-Verify**, exactly
as the README diagram specifies. Do **not** run the gate yourself in the main loop.

Use the template in `references/wave-workflow.js` — read it. Build the `WAVE_ISSUES` array from the issues
you settled with the user: for each, `{ number, title, slug, type, scope, plan, acceptance, edgeCases,
touchesCodegen, files, dependsOn?, groupSlug? }` (add `groupSlug` only to issues the user agreed to bundle
into one PR; carry `dependsOn` so the template's intra-wave dependency guard can fire). `title` is the plain
issue summary; the PR title is composed conventional-commit-style from `type`/`scope` in Step 4.
**Pass that array as the Workflow tool's `args`** (the template bails on an empty wave — this is the #1 way
to misfire). Launch via the Workflow tool. The template groups issues into **units** (one per distinct
`groupSlug || slug`) and manages one explicit worktree per unit at `../gu-2.0.0-<slug>` so
implement/review/remediate share the branch. Each unit's implement stage **guards against collision** — if
the worktree path or branch already exists (a concurrent run owns it), it returns `collision:true` and skips
that unit rather than clobbering.

Every hard rule (branch `<type>/<issue#>-<slug>` off `feat/2.0.0` never `main`; no hand-edits to
`*.generated.go`; the build/test/lint Verify gate + fix loop, incl. the Homebrew-Go PATH; codegen → `go
generate` + golden diff; docs synced in-PR; breaking changes → `breaking_changes.md`; conventional commits)
is already baked into the template's agent prompts from the README (§2–§5) — you don't re-enforce them in the
main loop, `references/wave-workflow.js` does.

**Model policy (baked into the template):** Implement+Verify and Remediate run on **Sonnet** (the high-volume
code/fix-loop work, guided by the contract and backstopped by the gate + review); the architect ‖ test-lead
**Review runs on Opus** (low-volume, high-leverage judgment). For an unusually subtle unit (e.g. the
codegen/polymorphism work) you may bump Implement to Opus for that run.

The **architect reviewer** judges design, not just correctness: KISS/DRY/SOLID, clean structure and
separation of concerns, maintainability/testability, ease of understanding, appropriate design patterns and
clean code, idiomatic Go, and — most of all — **developer experience for consumers of this library** (the
public API must be intuitive, consistent, and hard to misuse). Implementation keeps **comments short (≤2
lines, explaining WHY)**, going longer only for genuinely complex logic.

## Step 4: Open PRs and close issues

For each **unit** whose gate went green and did NOT collide (confirm with the user before opening PRs if
there's any doubt), open ONE PR from its worktree. A grouped unit's PR references every member issue:

```bash
gh pr create --base feat/2.0.0 --title "<type>(<scope>): <summary> (#<N>[, #<M>...])" --body "..."   # never --base main
# add --label breaking if the unit changed public API/behavior
# PR opened -> swap the claim to review state for every member issue:
for N in <unit issue numbers>; do gh issue edit "$N" --remove-label in-progress --add-label in-review; done
```

PRs targeting `feat/2.0.0` do **not** auto-close their issue (README §3). After the PR merges, clear
`in-review` and close every member issue manually, then remove the worktree:

```bash
for N in <unit issue numbers>; do
  gh issue edit "$N" --remove-label in-review
  gh issue close "$N" --comment "Done in <PR-url>."
done
git worktree remove ../gu-2.0.0-<slug>
```

If a unit collided or its gate stayed red and no PR opened, **release its claim** so it returns to the pool:
`gh issue edit <N> --remove-label in-progress`. Reserve `Closes #` keywords for the eventual `feat/2.0.0` →
`main` PR.

## Step 5: Report

Summarize per issue: PR link, gate result (tests/lint green), docs synced, breaking changes recorded,
deferred non-blocker findings (with where they were logged). If any issue's gate stayed red, say so plainly
with the failing output — never claim a wave passed when it didn't.

## After all waves

The final whole-codebase review (README §7) is its own wave: architect + test-lead over the entire
`feat/2.0.0` diff, run as a Workflow, before opening the `feat/2.0.0` → `main` PR. Use this same skill with
that as the scope — and grill the user on what "ready for main" means first.

## Before reporting done

First verify the wave against the **README §6 per-PR checklist** (issue completeness, branching, disjointness,
no generated-code edits, green build/test/lint + in-workflow gate, docs synced, conventional commits) — don't
re-list it here. On top of §6, confirm the wave-only gates §6 doesn't cover:

- [ ] Wave scope, slicing, overlap, grouping (`groupSlug`), and launch each confirmed by the user (not assumed).
- [ ] Dependency gate passed: every issue's `dependsOn` merged/closed; no blocker shares the wave with its dependent.
- [ ] Each issue claimed `in-progress` before launch; candidate listing excluded already-claimed issues.
- [ ] No worktree/branch collision clobbered.
- [ ] On PR open, member issues swapped `in-progress` → `in-review`; on abort/red, claim released.
- [ ] Member issues closed (clearing `in-review`) after merge; worktree removed.
