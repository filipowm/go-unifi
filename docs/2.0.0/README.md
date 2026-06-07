# go-unifi 2.0.0 — implementation process baseline

> **Read this BEFORE executing ANY 2.0.0 work.** This is the *process* spec for Claude Code workflows —
> how work is sliced, tracked, branched, reviewed, and merged. It is **not** a user migration guide and
> **not** a design doc. Plans live in GitHub issues, never in the repo. API breaks are logged in
> [`breaking_changes.md`](breaking_changes.md).

## What 2.0.0 is — read epic #117, not this

The *what/why/scope* of 2.0.0 (OpenAPI 3.1 retarget, the hybrid legacy↔OpenAPI transition, the `APIStyle`
seam, runtime/auth targets, version floors, and the enumerated breaking changes) lives in
[epic #117 — *Migrate to UniFi OS Server & official OpenAPI*](https://github.com/filipowm/go-unifi/issues/117),
milestone **2.0.0**. Read it there, don't restate it here. This README is the **process** contract only; the
migration constants the process references are listed once below under §1 (Known edge cases).

---

## 1. GitHub Issues are the single source of truth

There are **no plan/spec/design `.md` files in the repo** for migration work. **Every change = one GitHub
issue**, and the issue body is the contract.

**Issue body (required sections):**
- **Description** — concise statement of the change (2–4 sentences).
- **Implementation plan** — ordered, concrete steps naming the files/dirs touched.
- **Acceptance criteria** — objectively verifiable checklist; always includes "build + test + lint pass"
  and "docs updated".
- **Known edge cases** — version gating (floor 9.0.114, OpenAPI from 10.1.78), dual-shape resources,
  error mapping (`meta.rc==error` → `*ServerError`, 404 → `ErrNotFound`), any breaking change.

**Issue metadata (required):** type label (`feat`/`fix`/`refactor`/`docs`/`chore`/`test`/`ci`), plus
`breaking` when public API/behavior changes; milestone **2.0.0**; link to parent epic **#117**.

**Dependencies:** GitHub has no native "blocked-by", so a hard dependency is a body convention — a
`Depends on #N[, #M]` line. There is **no** dependency/blocked label: blocked-ness is *computed* from the
`Depends on #N` lines plus each dep's open/closed state, so it can never go stale. `find-candidates.sh` (in
the `unifi-2.0.0-wave` skill) does this in one cheap query — listing READY / BLOCKED / CLAIMED issues without
reading any body. **Skeleton-first falls out for free:** while the scaffolding skeleton is open, everything
that `Depends on` it computes as BLOCKED, so the skeleton is the only ready candidate.

**Slicing:**
- One feature/refactor/change = **one issue = one small PR**. Small and cohesive.
- **Architecture-first:** the VERY FIRST work item is an architecture/scaffolding issue+PR that designs the
  folder/submodule/package structure — where the OpenAPI generator, the spec source, and the
  `integration/v1` runtime live — and lands the skeleton on `feat/2.0.0` **before** any fan-out. Seed a
  first batch of issues early; add more as the picture clarifies.
- **Non-overlapping waves:** issues in a parallel batch MUST touch **disjoint files/dirs** so branches
  never conflict. Overlap is the cardinal sin. Confirm disjointness before fan-out.

---

## 2. Branching model

**NEVER commit to or open PRs against `main`.** Base for everything is **`feat/2.0.0`** (already exists).
Each issue branches off `feat/2.0.0` (after the skeleton has landed) and PRs back into `feat/2.0.0`. Keep
the hierarchy **flat**; stack PRs only when a hard dependency forces it. Run parallel branches in **git
worktrees** for isolation.

```
main
 └── feat/2.0.0                          (integration branch — base for ALL 2.0.0 work)
      ├── feat/115-openapi-skeleton      (architecture-first; lands BEFORE fan-out)
      ├── feat/123-openapi-dns           (off feat/2.0.0 after skeleton; own worktree)
      ├── refactor/130-apistyle-seam     (parallel; disjoint files)
      └── ...                            (flat; stack ONLY on hard dependency)
```

**Branch naming:** `<type>/<issue#>-<short-slug>` — e.g. `feat/123-openapi-generator`,
`refactor/130-apistyle-seam`, `docs/118-breaking-changes`. The leading issue number makes GitHub
auto-link the branch to that issue's **Development** section; a grouped PR lists every member number
(`feat/123-124-openapi-dns`).

**Conventional commits** everywhere; PR title follows the same convention; reference the issue.

```
<type>(<scope>): <summary>      # imperative, lowercase, no trailing period
types: feat fix chore docs refactor test ci   scope optional, e.g. (codegen) (client) (openapi)

feat(codegen): add OpenAPI 3.1 resource generator behind APIStyle seam (#123)
fix(client): map meta.rc==error to *ServerError (#131)
docs(2.0.0): record API-key-only auth in breaking_changes.md (#118)
```

```bash
git worktree add -b feat/123-openapi-dns ../gu-2.0.0-openapi-dns feat/2.0.0   # -b <type>/<issue#>-<slug> <path (../gu-2.0.0-<slug>)> <start-point>
# …work, verify…
gh pr create --base feat/2.0.0 --title "feat(openapi): DNS resource (#123)"   # never --base main
```

---

## 3. Lifecycle of one change

Everything runs through **Claude Code Workflows** (the Workflow tool): multi-phase, subagent-driven. One
workflow may cover one or more issues and produce one or more PRs. The quality gate lives **INSIDE the
workflow, never in the main loop**.

```
GH issue (plan + AC + edges, linked to #117, milestone 2.0.0)
        │
        ▼  branch off feat/2.0.0  (git worktree for isolation)
   ┌──── INSIDE WORKFLOW (per wave) ───────────────────────────┐
   │  Implement                                                 │
   │     ▼                                                      │
   │  Verify  → build / test / lint ──(fail)──▶ fix loop ──┐    │
   │     ▼ (pass)                                          │    │
   │  Review  → architect ‖ test-lead (parallel)          │    │
   │     ▼                                                 │    │
   │  Remediate (gated: blocker/major only) ──────────────┘    │
   │     ▼                                                      │
   │  re-Verify (build / test / lint pass)                     │
   └───────────────────────────────────────────────────────────┘
        │
        ▼  docs synced in the SAME PR
PR → feat/2.0.0  (checklist green) → merge → close issue MANUALLY (gh issue close <n>)
```

- **Verify** has its own fix loop — never leave the phase red.
- **Remediate is gated:** it fires only on blocker/major findings; minor/nits are logged on the issue or
  deferred to a follow-up.
- **Issues do NOT auto-close.** `Closes #N` only auto-closes when a PR merges into the **default branch
  (`main`)**; these PRs target `feat/2.0.0`, so after merge the workflow MUST close the issue manually
  (`gh issue close <n>`). Reserve `Closes #` keywords for the eventual `feat/2.0.0` → `main` PR.
- **After every issue/PR: all tests + lint pass. Non-negotiable.**
- A **wave** is one parallel batch of provably non-overlapping issues; one worktree per concurrent branch.
  Plan the next wave only after the current one merges (keeps the tree flat, lets the picture clarify).

---

## 4. Commands & style

```bash
go build ./...
go test -cover -coverprofile=coverage.out -covermode atomic ./...
go test -run TestName ./unifi                # single test
golangci-lint run                            # gofumpt + goimports + gci; tabs; max line 200
go generate unifi/codegen.go                 # only when regenerating resources
```

Local `Makefile` wraps these: `build | test | test-fast | cover | lint | fmt | check | generate`.

- **Never hand-edit `*.generated.go`** (they start with `DO NOT EDIT`; CI regenerates). Change output via
  `codegen/internal/customizations.yml` or add a hand-written sibling `.go` (see
  [`codegen/CLAUDE.md`](../../codegen/CLAUDE.md)). Generated CRUD is private (`getUser`); public wrappers
  (`GetUser`) are hand-written siblings.
- `ctx context.Context` is the first arg of every client method. Wrap returned errors with `%w`.

---

## 5. Docs that MUST stay in sync (same PR as the change)

- `docs/` (getting_started, configuration, codegen, usage_examples, advanced_topics, file_uploads,
  compatibility_matrix, migrating_from_upstream, …)
- root **README.md**
- the relevant **CLAUDE.md** (root and/or `codegen/CLAUDE.md`)
- **`.claude/rules/`** (`go-conventions.md`, `testing.md`)
- API breaking changes → **`docs/2.0.0/breaking_changes.md`**

---

## 6. Per-PR checklist

A terse index of the rules above — each links back to its authoritative section.

- [ ] Issue complete (description, plan, acceptance criteria, edge cases) + metadata. *(§1)*
- [ ] Branched off `feat/2.0.0`; PR targets `feat/2.0.0`, never `main`. *(§2)*
- [ ] Small, cohesive, disjoint from sibling PRs in the wave. *(§1)*
- [ ] No hand-edits to `*.generated.go`. *(§4)*
- [ ] `go build`, full `go test`, `golangci-lint run` all green. *(§4)*
- [ ] In-workflow gate completed. *(§3)*
- [ ] Docs synced in this PR, incl. `breaking_changes.md` for API breaks. *(§5)*
- [ ] Conventional commits + PR title; issue closed manually after merge. *(§2–3)*

---

## 7. Final gate

After **all** waves merge to `feat/2.0.0`, run a thorough **whole-codebase review by software architect +
test lead** — itself a Claude Code workflow — before `feat/2.0.0` is considered ready and a `feat/2.0.0` →
`main` PR is opened.