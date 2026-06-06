---
name: unifi-issue-author
description: >-
  Author go-unifi 2.0.0 GitHub issues — the planning/authoring counterpart to unifi-2.0.0-wave. Use this
  whenever the user wants to CREATE or FLESH OUT work items rather than implement them: "create issues for
  2.0.0", "decompose epic #117", "seed the next batch of issues", "plan the OpenAPI migration into tasks",
  "write acceptance criteria for #N", "this issue is too vague, flesh it out", "turn this into a proper
  issue", "what should the next wave's issues be?". Trigger even when the word "issue" isn't used — any
  request to plan, scope, slice, decompose, or write up 2.0.0 work as trackable tasks means authoring. Do
  NOT trigger to IMPLEMENT/build/PR work (that's unifi-2.0.0-wave) or for user-facing upgrade questions
  ("how do I migrate to 2.0.0?" — that's the migration guide).
argument-hint: "what to plan, e.g. 'decompose epic #117', 'next batch of OpenAPI issues', 'flesh out #142'"
model: opus
---

Author 2.0.0 work as GitHub issues that are **ready to implement** — each a small, cohesive, contract-shaped
contract that `unifi-2.0.0-wave` can pick up and build without further clarification. Scope: $ARGUMENTS.

The process contract is **`docs/2.0.0/README.md` §1** — it defines what a complete issue is (body sections +
metadata + slicing). This skill is the *authoring* layer; the wave skill is the *execution* layer. You write
the contract; the wave fulfills it. When the two disagree, the README wins; fix this skill to match.

**This skill stops at "issue filed." It never implements, branches, or opens PRs.** When the issues are
ready, hand off to `unifi-2.0.0-wave`.

## The prime directive: the issue IS the contract, so get it right before it exists

A vague issue is worse than no issue — it sends the wave's subagents off to build the wrong thing in an
isolated worktree, and you only find out at review. The whole point of authoring is to front-load every
decision into the issue body so implementation is mechanical. So **interrogate the user the same way the
wave does: assume nothing, ask everything consequential.** Use `AskUserQuestion`; batch 2–4 related
questions; keep going in rounds until the issue could be handed to a stranger and built correctly.

Treat the user as the authority on intent and the epic (#117) as the authority on the overall shape. Your
job is to turn fuzzy intent into precise, testable, disjoint work items — *with* the user, not by guessing.

## Step 0: Load the contract and the existing landscape

Read before asking — informed questions beat blank ones:

```bash
cat docs/2.0.0/README.md                                  # §1 = the issue contract; §5 = docs that must sync
gh issue view 117 --json title,body                       # the epic: scope, enumerated breaking changes
gh issue list --milestone 2.0.0 --json number,title,labels,state   # what already exists — don't duplicate
git branch --show-current                                 # confirm feat/2.0.0 exists as the base
```

If you're **fleshing out an existing issue** (`$ARGUMENTS` names #N), pull it and treat its current body as a
draft to complete, not replace wholesale:

```bash
gh issue view <N> --json number,title,body,labels,milestone,state
```

## Step 1: Settle scope and slicing (the core of this skill)

Run the interview. Do not write any issue body until each item is **explicitly settled by the user**:

- **Objective** — what is the user actually trying to land? Restate it and confirm.
- **Skeleton-first check (README §1)** — has the architecture/scaffolding skeleton already landed on
  `feat/2.0.0`? If not, the FIRST authored issue MUST be that skeleton (it designs where the OpenAPI
  generator, spec source, and `integration/v1` runtime live) and everything else depends on it. Sanity-check
  by looking for the expected dirs/files, then confirm with the user — don't assume.
- **Slicing** — decompose the objective into **one-issue-one-small-PR** units. Each must be cohesive and
  independently shippable. Push back when the user lumps too much into one issue, or splits one change across
  several. Where are the seams?
- **Disjointness** — for a batch meant to run as one wave, which files/dirs does each issue touch? They must
  be **disjoint** so the wave can parallelize them (README §1). Flag the shared-file traps —
  `codegen/customizations.yml`, root `README.md`, `docs/`, `.claude/rules/` are co-touched by many issues.
  If two issues collide, decide *with the user*: sequence them, merge them, or carve the boundary differently.
- **Dependencies** — does any issue need another's output first (a package/seam/decision that must land
  before it can be built)? Keep the tree flat; only record a HARD dependency that forces sequencing. Every
  hard dependency you find MUST be recorded in two places: a `Depends on #N` line in the issue body (Step 3)
  and a `dependsOn: [N]` entry in the wave handoff (Step 4). There is no blocked label — the wave's
  `find-candidates.sh` computes blocked-ness from the `Depends on #N` line + each dep's open/closed state, so
  getting that line right is what makes the dependency real. **A blocker and its dependent can never share a
  parallel wave** — they run in separate, sequenced waves with the blocker merged first; the wave template
  hard-fails if you violate this, so get the deps right here.
- **Per-issue acceptance** — what does "done" mean, concretely and testably? Every issue's acceptance always
  includes "build + test + lint pass" and "docs updated"; add the change-specific, objectively-checkable
  criteria on top.
- **Edge cases** — which of the README §1 edge cases apply per issue (version gating, dual-shape resources,
  error mapping, empty/error envelopes, backward compat)? Name the specific ones in each issue's body.
  (Constants — floors, error-mapping targets — live in README §1; don't re-derive them.)
- **Breaking?** — will the change alter public API/behavior? Then it needs the `breaking` label and the issue
  must call for a `docs/2.0.0/breaking_changes.md` entry. Confirm per issue.
- **Codegen?** — does it change generated output? Then the plan must say "edit `codegen/customizations.yml`
  (or add a hand-written sibling), never `*.generated.go`; run `go generate`; check the golden type-diff."

## Step 2: Draft each issue body to the §1 contract

For every issue, write the body with **exactly these sections** (README §1 — the wave reads them verbatim):

```markdown
## Description
<2–4 sentences: what changes and why.>

## Implementation plan
1. <ordered, concrete steps that NAME the files/dirs touched>
2. ...

## Acceptance criteria
- [ ] <change-specific, objectively verifiable>
- [ ] `go build ./...`, full `go test`, `golangci-lint run` all green
- [ ] Docs synced (list which: docs/, root README, relevant CLAUDE.md, .claude/rules/)
- [ ] <if breaking> `docs/2.0.0/breaking_changes.md` entry added
- [ ] <if codegen> regenerated via `go generate`; golden type-diff clean; only customizations.yml edited

## Known edge cases
- <version gating / dual-shape / error mapping / breaking — whichever apply, or "none">
```

Keep plans concrete and file-named — that's what makes them mechanical to build and lets the wave prove
disjointness. The issue title is a **plain, descriptive summary — NO conventional-commit prefix** (e.g.
"Add DNS resource behind the APIStyle seam", not "feat(openapi): …"). The type/scope live in the type label
and the wave handoff; the wave composes the conventional-commit *PR* title from them at PR time.

**Play every drafted issue back to the user** (title, body sections, labels, files-touched, breaking?,
codegen?) and get explicit confirmation or correction before creating anything. This read-back is mandatory —
issues are expensive to un-create and the wave trusts them blindly.

## Step 3: Create the issues

Only after sign-off. Ensure the type labels exist, then create each issue with milestone 2.0.0, the type
label (+ `breaking` where it applies), and a link to the epic:

```bash
# labels are idempotent; create if missing (types: feat/fix/refactor/docs/chore/test/ci, plus breaking)
gh label create feat --color 0E8A16 2>/dev/null || true   # repeat per needed type/label

gh issue create \
  --title "<plain descriptive summary — NO conventional-commit prefix>" \
  --milestone 2.0.0 \
  --label <type> [--label breaking] \
  --body "$(cat <<'EOF'
<the §1 body from Step 2>

---
Part of epic #117.
Depends on #<N>[, #<M>]    # OMIT this line entirely when the issue has no hard dependency
EOF
)"
```

Type/scope are NOT in the title — they live in the type label (and the Step 4 handoff); the wave composes the
conventional-commit PR title from them. GitHub has no native parent/child or "blocked-by" for issues, so both
the epic link and any hard dependency are **body conventions**: `Part of epic #117.` and `Depends on #N` — the
wave's candidate finder parses that `Depends on` line to compute blocked-ness, so it must be exact. After
creating, confirm the URLs back to the user.

## Step 4: Emit a wave-ready handoff

The whole point is a clean handoff to `unifi-2.0.0-wave`. For the batch you just created (or that's now
ready), emit the `WAVE_ISSUES` array the wave consumes — one object per issue — so launching a wave is
copy-paste, not re-derivation:

```js
[
  { number: <N>, title: '<plain summary, no prefix>', slug: '<short-slug>', type: '<feat|fix|refactor|...>', scope: '<...>',
    plan: '<the implementation plan>', acceptance: '<the acceptance criteria>', edgeCases: '<...>',
    touchesCodegen: <true|false>, files: ['<disjointness proof: every file/dir touched>'],
    dependsOn: [<issue numbers that must be MERGED first>],   // [] if none — dependents belong in a LATER wave
    /* groupSlug: '<shared-slug>'  // only if two issues should land in ONE PR */ },
  // ...
]
```

`dependsOn` must match the `Depends on #N` line in the issue body. The wave template HARD-FAILS if a unit and
a unit it depends on are launched in the same wave, so be honest here. State plainly which issues are ready
to run NOW (no unmet dependency) vs which must wait for a blocker to merge: "Issues X are ready — run
`unifi-2.0.0-wave` with this array. Issue Y depends on X and goes in a later wave." Do **not** start
implementing — that's a separate, user-initiated decision and a separate skill.

## Step 5: Report

Summarize: each created issue (number, URL, type, breaking?, codegen?), the proposed wave grouping and why
it's disjoint, and the recommended first wave (skeleton alone if it isn't landed yet). Call out anything you
deferred or couldn't pin down so the user can close the gap before a wave runs.

## Checklist (verify before reporting done)

- [ ] Objective, slicing, disjointness, dependencies each confirmed by the user (not assumed).
- [ ] Skeleton-first respected — if the scaffolding hasn't landed, it's the first/only issue in the batch.
- [ ] Every issue has all four §1 body sections + testable acceptance (incl. build/test/lint + docs synced).
- [ ] Issue titles are plain descriptive summaries — NO conventional-commit prefix (type lives in the label).
- [ ] Hard dependencies recorded in BOTH the issue body (`Depends on #N`) and the handoff (`dependsOn: [N]`); a blocker and its dependent are never in the same wave.
- [ ] Metadata set: type label, `breaking` where applicable, milestone 2.0.0, body links #117.
- [ ] Batch issues touch disjoint files/dirs; collisions resolved (sequence/merge/recarve) with the user.
- [ ] Read-back delivered and confirmed before any `gh issue create`.
- [ ] Wave-ready `WAVE_ISSUES` array emitted; handoff to `unifi-2.0.0-wave` named, not auto-started.
