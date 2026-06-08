---
name: unifi-coverage-matrix
description: >-
  Regenerate docs/2.0.0/coverage_matrix.md — the resource-centric API coverage
  table showing which UniFi resources are covered by the Legacy Internal API,
  the Official OpenAPI surface, or both. Use whenever the Official API groups or
  legacy resources change and the matrix needs to reflect the new state.
argument-hint: "optional: path overrides, e.g. '--output docs/2.0.0/coverage_matrix.md'"
---

# unifi-coverage-matrix skill

Regenerate `docs/2.0.0/coverage_matrix.md` by grounding every row in live repo
sources. Never fabricate resource names or group names — every entry must exist
in the codebase.

---

## Procedure

### Step 1 — Collect Official API groups

Grep the live `unifi/official/client.generated.go` for the accessor methods on
the `Client` interface. Each method name is one official group. Discover them
generically — never hardcode the list, so new groups are automatically picked up:

```bash
grep -oE '^\t[A-Z][A-Za-z]+\(\) [A-Z][A-Za-z]+Client$' unifi/official/client.generated.go \
  | grep -v 'Supporting'
```

Each matched name (the part before `()`) is one official group. For each group,
inspect its `<group>.generated.go` file in `unifi/official/` to understand what
operations it exposes (full CRUD vs. read-only vs. actions-only).

**IMPORTANT — Supporting is intentionally excluded.** The `Supporting` group is
a shared enum/lookup helper (countries, DPI categories, device tags, etc.), not
a real resource surface. It is omitted from the matrix. Do the same for any
future pure helper/lookup groups added to the Official API.

### Step 2 — Collect legacy resources

List the types exposed by the legacy `InternalClient` interface:

```bash
grep '==== client methods for' unifi/client.generated.go \
  | sed 's/.*for \(.*\) resource.*/\1/' | sort
```

Each named type is a legacy resource backed by a `*.generated.go` file in
`unifi/`.

### Step 3 — Curate the resource union

Build a UNION of meaningful resource concepts across both surfaces. Rules:

- Map multiple related official operations or legacy types onto ONE conceptual
  row where they clearly represent the same real-world resource (e.g.
  `FirewallZone` legacy + Official `Firewall.Zone` ops → "Firewall Zones";
  `FirewallZonePolicy` legacy + Official `Firewall.Policy` ops → "Firewall Policies").
- Keep concepts distinct when the models differ enough to matter (e.g.
  "Firewall Policies" vs "Firewall Zones" are separate rows).
- Include legacy-only concepts that have no Official equivalent (Port Forwarding,
  Routing, etc.).
- Include Official-only concepts that have no legacy equivalent (ACL Rules,
  Sites, Traffic Matching Lists, etc.).
- Omit trivially internal or infrastructure-only types (compile-time stubs,
  version constants).
- Settings resources (`Setting*`) are singleton configuration, not managed
  resources in the CRUD sense; omit them unless a direct Official counterpart
  exists.

### Step 4 — Assess coverage honestly

For each row, apply the following judgment per surface:

| Mark | Meaning |
|------|---------|
| ✅   | Covered — full CRUD or the natural scope of this resource is accessible |
| ⚠️   | Partially covered — some operations exist but the surface is incomplete |
| ❌   | Not covered — no equivalent on this surface |

Coverage is the skill's honest, grounded judgment based on the actual method
signatures found in Steps 1–2. Do not assert full coverage if significant
operations are missing.

### Step 5 — Write the Comments column

Fill in the Comments column ONLY for:
- **Partial coverage (⚠️)** — explain concisely what is missing or different.
- **Official-only resources (Legacy ❌, Official ✅)** — explain what the
  Official API adds.

Leave the Comments cell **blank** for rows where both surfaces are ✅, or where
only one surface has ✅ and the other ❌ without noteworthy explanation.

### Step 6 — Emit the document

Write `docs/2.0.0/coverage_matrix.md` with:

1. A short intro paragraph.
2. The emoji legend table (✅/⚠️/❌).
3. The resource table: `| Resource | Legacy Internal API | Official API | Comments |`.
   - Sort rows alphabetically by Resource name.
   - Legacy/Official cells contain ONLY an emoji mark (no text).
   - Comments are plain prose, no markdown formatting inside the cell.

The format must be stable and consistent across runs so that diffs remain
readable. Do not add trailing spaces, vary column widths, or reorder columns.

---

## Constraints

- **NEVER fabricate resource or group names.** Every row must be grounded in a
  real type or group found in Steps 1–2.
- **Supporting is intentionally excluded** from the table. Do not add a row for
  it or document it as skipped; just omit it.
- Do NOT add the coverage matrix link to `docs/compatibility_matrix.md`.
- The document is human-facing only; no Go code or CI test parses it.
- Regenerate the WHOLE document on each run — do not patch individual rows.
