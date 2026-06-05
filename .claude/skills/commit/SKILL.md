---
name: commit
description: Create conventional commits of changes. Use when asked to commit, save changes, create a commit, or when finishing implementation work. Also use when the user says "commit this", "save my work", "create commits", or just "/commit".
argument-hint: "Describe the change you want to commit"
model: sonnet
---

Create conventional commits grouped by logical changes. Scope: $ARGUMENTS (if empty, commit all changes).

## Step 1: Gather context

Run these in parallel:

```bash
git status
git diff --stat
git diff --staged --stat
git log --oneline -10
```

This tells you what's changed, what's already staged, and what the recent commit style looks like.

## Step 2: Normalize the staging area

If files are already staged from a previous partial workflow, reset them so you can regroup logically:

```bash
git reset HEAD
```

This is safe — it only unstages, never discards work. Skip this if nothing is staged.

## Step 3: Check for secrets and forbidden files

Before staging anything, verify no secrets or sensitive files are in the changeset:

```bash
git diff --name-only | grep -iE '(secret|\.env$|credentials|token|password|key\.pem|\.key$)' || true
```

**False positives to ignore:** `*.env.example`, `.gitkeep`, `secrets.tf` (OpenTofu config, not actual secrets), `secrets/` dirs containing only `.gitkeep`. Use judgment — the grep catches filenames, not file contents.

If any real secrets are found, **do NOT commit those files**. Warn the user and skip them.

## Step 4: Review diffs and plan commit groups

For small changesets (<15 files), read the full diff:
```bash
git diff
```

For large changesets, use `--stat` output from Step 1 to understand the shape, then spot-check key files if grouping isn't obvious from paths alone.

**Plan all commit groups before executing any.** Write out the groups mentally — this prevents orphan files that don't fit any later commit. Each group should be a cohesive, self-contained change.

Grouping heuristics (in priority order):
1. **By host** — all compose files for one host = one commit (`compose/mac-mini/`, `compose/nas/`, `compose/rpi4/`)
2. **By stack/module** — a single OpenTofu stack or module = one commit, unless changes span many stacks (then group all terraform together)
3. **By concern** — CI workflows, Grafana dashboards, operational scripts, etc.
4. **By config layer** — `.gitignore` + `.env.example` together; CLAUDE.md + skills + rules together
5. **Unrelated single files** — separate commits or group with the nearest logical neighbor

When in doubt, fewer larger commits beat many tiny ones — a commit should tell a coherent story.

## Step 5: Execute commits

For each planned group, stage and commit in one step:

```bash
git add <file1> <file2> ... && git commit -m "$(cat <<'EOF'
<type>(<scope>): <description>

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

### Commit message format

Follow **Conventional Commits**:

```
<type>(<scope>): <short description>
```

**Types:** `feat`, `fix`, `chore`, `refactor`, `docs`, `ci`, `test`

**Scopes** (derive from the files changed):
- Host names: `mac-mini`, `rpi4`, `nas`
- Infra areas: `tofu`, `ci`, `caddy`, `komodo`, `renovate`
- Specific stacks: `immich`, `observability`, `autopirate`, etc.
- Use `claude` for CLAUDE.md/skills changes, `infra` for cross-cutting infra changes

**Rules:**
- Lowercase description, no period at the end
- Focus on **why**, not **what** — the diff shows what changed
- Keep the first line under 72 characters
- Add a blank line + body only if the why isn't obvious from the subject
- Always include the `Co-Authored-By` trailer

## Step 6: Verify

```bash
git status
git log --oneline -5
```

Confirm all intended changes are committed, nothing unexpected was staged, and the log looks clean.

## Rules

- NEVER push — only create local commits
- NEVER commit `.env` files (only `.env.example` is allowed)
- NEVER commit files matching `*secret*`, `*-secret*`, `*secret*.env`
- NEVER use `git add -A` or `git add .`
- NEVER amend existing commits unless explicitly asked
- If a pre-commit hook fails, fix the issue and create a NEW commit
