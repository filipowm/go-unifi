# go-unifi 2.0.0 — implementation process baseline

> **Read this BEFORE executing ANY 2.0.0 work.** This is the *process* spec for Claude Code workflows —
> how work is sliced, tracked, branched, reviewed, and merged. It is **not** a user migration guide and
> **not** a design doc. Plans live in GitHub issues, never in the repo. API breaks are logged in
> [`breaking_changes.md`](breaking_changes.md).

## What 2.0.0 is

Epic [#117 — *Migrate to UniFi OS Server & official OpenAPI*](https://github.com/filipowm/go-unifi/issues/117),
milestone **2.0.0**. Retarget codegen from reverse-engineered per-resource field JSONs to Ubiquiti's
official **OpenAPI 3.1** spec (`integration.json`, bundled in `unifi-uos_sysvinit.deb`). Keep the existing
download/version/extract pipeline; retire fragile regex type-inference. **Hybrid transition:** the legacy
and OpenAPI generators run side by side; migrate resource-by-resource; never delete a legacy path before
its OpenAPI replacement is validated. `APIStyle` (in `unifi/api_paths.go`) is the seam. Runtime targets
`/proxy/network/integration/v1/`, API-key auth only. Version floor 9.0.114; OpenAPI from 10.1.68;
dual-shape resources (e.g. DNS) pick shape by controller version. Breaking changes are enumerated in #117
and recorded in [`breaking_changes.md`](breaking_changes.md).

---

## 1. GitHub Issues are the single source of truth

There are **no plan/spec/design `.md` files in the repo** for migration work. **Every change = one GitHub
issue**, and the issue body is the contract.

**Issue body (required sections):**
- **Description** — concise statement of the change (2–4 sentences).
- **Implementation plan** — ordered, concrete steps naming the files/dirs touched.
- **Acceptance criteria** — objectively verifiable checklist; always includes "build + test + lint pass"
  and "docs updated".
- **Known edge cases** — version gating (floor 9.0.114, OpenAPI from 10.1.68), dual-shape resources,
  error mapping (`meta.rc==error` → `*ServerError`, 404 → `ErrNotFound`), any breaking change.

**Issue metadata (required):** type label (`feat`/`fix`/`refactor`/`docs`/`chore`/`test`/`ci`), plus
`breaking` when public API/behavior changes; milestone **2.0.0**; link to parent epic **#117**.

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
      ├── feat/2.0.0-openapi-skeleton    (architecture-first; lands BEFORE fan-out)
      ├── feat/2.0.0-openapi-dns         (off feat/2.0.0 after skeleton; own worktree)
      ├── refactor/2.0.0-apistyle-seam   (parallel; disjoint files)
      └── ...                            (flat; stack ONLY on hard dependency)
```

**Branch naming:** `<type>/2.0.0-<short-slug>` — e.g. `feat/2.0.0-openapi-generator`,
`refactor/2.0.0-apistyle-seam`, `docs/2.0.0-breaking-changes`.

**Conventional commits** everywhere; PR title follows the same convention; reference the issue.

```
<type>(<scope>): <summary>      # imperative, lowercase, no trailing period
types: feat fix chore docs refactor test ci   scope optional, e.g. (codegen) (client) (openapi)

feat(codegen): add OpenAPI 3.1 resource generator behind APIStyle seam (#123)
fix(client): map meta.rc==error to *ServerError (#131)
docs(2.0.0): record API-key-only auth in breaking_changes.md (#118)
```

```bash
git worktree add -b feat/2.0.0-openapi-dns ../gu-123 feat/2.0.0   # -b <new-branch> <path> <start-point>
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

- The gate per wave is **Implement → Verify → Review → Remediate → re-Verify**.
- **Verify** has its own fix loop — never leave the phase red.
- **Review** runs two subagents concurrently: software architect ‖ test lead.
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
  `codegen/customizations.yml` or add a hand-written sibling `.go` (see
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

- [ ] Issue exists; links epic **#117**; on milestone **2.0.0**; body has description, plan, acceptance
      criteria, edge cases; type label (+ `breaking` if applicable) set.
- [ ] Branched off `feat/2.0.0`; PR targets `feat/2.0.0` (**never `main`**).
- [ ] Change is small, cohesive, and disjoint from sibling PRs in the same wave.
- [ ] No hand-edits to `*.generated.go`.
- [ ] `go build ./...`, full `go test`, and `golangci-lint run` all green (tabs, ≤200 cols, ctx first, `%w`).
- [ ] In-workflow gate completed: Implement → Verify → Review (architect ‖ test-lead) → Remediate (gated)
      → re-Verify.
- [ ] Docs synced in this PR: `docs/`, root README, relevant `CLAUDE.md`, `.claude/rules/`.
- [ ] API breaking changes recorded in `docs/2.0.0/breaking_changes.md`.
- [ ] Conventional-commit messages and PR title. After merge, close the issue manually (`gh issue close <n>`) —
      it will NOT auto-close from a merge into `feat/2.0.0`.

---

## 7. Final gate

After **all** waves merge to `feat/2.0.0`, run a thorough **whole-codebase review by software architect +
test lead** — itself a Claude Code workflow — before `feat/2.0.0` is considered ready and a `feat/2.0.0` →
`main` PR is opened.