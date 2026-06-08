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
**enumerate the concrete operations** it exposes — do not eyeball the file. List
every exported method on the group's client so the coverage call in Step 4 is
backed by real signatures:

```bash
# e.g. group "Firewall" → type firewallClient → file firewall.generated.go
grep -oE 'func \(c [a-zA-Z]+Client\) [A-Z][A-Za-z]+\(' unifi/official/firewall.generated.go \
  | sed -E 's/func \(c [a-zA-Z]+Client\) //; s/\($//' | sort -u
```

Record the verb set per official resource. Treat `ListXAll`/`ListXPage` pairs as
a single **List** capability, and classify the rest as **Create / Get / Update /
Delete** or named **actions** (e.g. `Adopt`, `Restart`, `UpdateRuleOrdering`). A
group may expose several distinct resources (e.g. `Firewall` → Policy + Zone);
enumerate the verbs **per resource noun**, not per group.

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
`unifi/`. As with the Official side, **enumerate the concrete operations** per
type rather than assuming uniform CRUD — the generator omits verbs a resource
does not support:

```bash
# replace FirewallZone with the resource type name
grep -oE 'func \(c \*client\) (get|list|create|update|delete)FirewallZone\(' unifi/*.generated.go \
  | sed -E 's/.*\) //; s/FirewallZone\(//' | sort -u
```

Record the verb set (`get`/`list`/`create`/`update`/`delete`) per legacy
resource. A type missing `create`/`update`/`delete` is read-only on the legacy
surface, which Step 4 must reflect.

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

### Step 4 — Assess coverage from the enumerated operations

Coverage is **not** a vibe. Decide each surface's mark by comparing the verb
sets you enumerated in Steps 1–2, not by skimming. For every row, first write
down the operation evidence, then map it to a mark:

1. **Build the per-surface operation set.** From Steps 1–2 you have the exact
   verbs each surface exposes for the resource (e.g. legacy
   `{get,list,create,update,delete}`; Official `{Create,Get,List,Update,Delete}`
   plus actions). If a surface has **zero** matching methods, it is ❌ — full
   stop.
2. **Define the resource's expected lifecycle.** For a managed resource that is
   normally create/read/update/delete. Read-only or action-only resources
   (e.g. `Info`, connected-`Clients`, `Sites`) have a narrower natural scope —
   listing/reading IS the full surface there.
3. **Map evidence → mark** using this rubric:

| Mark | Precise meaning |
|------|-----------------|
| ✅   | Every operation in the resource's expected lifecycle is present on this surface (full CRUD for a managed resource; the complete read/action set for a read-only resource). |
| ⚠️   | At least one operation is present but the set is incomplete vs. the expected lifecycle, OR the surface exposes a materially narrower model than its counterpart (e.g. read/list only, or actions without full configuration). |
| ❌   | No method on this surface targets the resource at all. |

Cross-check the two surfaces against each other: if one surface has full CRUD
and the other only `list`/`get`, the thin side is ⚠️ (with the gap named in the
Comments column per Step 5), **never** ✅. When in doubt between ✅ and ⚠️, the
deciding question is concrete: *"Which CRUD/lifecycle verb is missing?"* If you
can name a missing verb, it is ⚠️; if you cannot, it is ✅. Capture that reason —
it becomes the Comments text for ⚠️ rows.

### Step 5 — Write the Comments column

Fill in the Comments column ONLY for:
- **Partial coverage (⚠️)** — explain concisely what is missing or different.
- **Official-only resources (Official ✅, Legacy ❌)** — explain what the
  Official API adds.

Leave the Comments cell **blank** for rows where both surfaces are ✅, or where
only one surface has ✅ and the other ❌ without noteworthy explanation.

### Step 6 — Emit the document

Write `docs/2.0.0/coverage_matrix.md` with:

1. A short intro paragraph.
2. The emoji legend table (✅/⚠️/❌).
3. The resource table: `| Resource | Official API | Legacy Internal API | Comments |`.
   - Sort rows alphabetically by Resource name.
   - Official/Legacy cells contain ONLY an emoji mark (no text).
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
