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

**The matrix is EXHAUSTIVE, not curated.** Every legacy resource type and every
official resource surface (except the `Supporting` lookup helper) MUST appear —
either as its own row or merged into a shared row with its true counterpart.
Dropping a resource because it seems niche, infrastructural, or "just a setting"
is a bug. The whole point of this document is that a reader can find ANY resource
the library exposes and see where it is covered.

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

Some groups (`Info`, `Sites`) use a different receiver name — if the grep above
returns nothing for a group, fall back to `grep -nE 'func ' <file>.generated.go`
and read the signatures directly.

Record the verb set per official resource. Treat `ListXAll`/`ListXPage` pairs as
a single **List** capability, and classify the rest as **Create / Get / Update /
Delete** or named **actions** (e.g. `Adopt`, `Restart`, `UpdateRuleOrdering`). A
group may expose several distinct resources (e.g. `Firewall` → Policy + Zone);
enumerate the verbs **per resource noun**, not per group.

**IMPORTANT — Supporting is intentionally excluded.** The `Supporting` group is
a shared enum/lookup helper (countries, DPI categories, device tags, radius
profiles, WANs, etc.), not a real resource surface. It is omitted from the
matrix. Do the same for any future pure helper/lookup groups added to the
Official API.

### Step 2 — Collect ALL legacy resources

List every type exposed by the legacy `InternalClient` interface. This is the
authoritative legacy resource set — **every name it prints must end up in the
matrix.**

```bash
grep '==== client methods for' unifi/client.generated.go \
  | sed 's/.*for \(.*\) resource.*/\1/' | sort
```

The list splits into two shapes:

1. **Managed resources** (e.g. `Network`, `FirewallRule`, `Tag`, `Device`) —
   backed by lowercase CRUD methods. Enumerate the verb set:

   ```bash
   # replace FirewallZone with the resource type name
   grep -oE 'func \(c \*client\) (get|list|create|update|delete)FirewallZone\(' unifi/*.generated.go \
     | sed -E 's/.*\) //; s/FirewallZone\(//' | sort -u
   ```

2. **Settings singletons** (every `Setting*` type) — exposed as **public**
   `GetSettingX` / `UpdateSettingX` interface methods only (no create / list /
   delete). Confirm the verb set generically:

   ```bash
   for r in $(grep '==== client methods for' unifi/client.generated.go \
       | sed 's/.*for \(.*\) resource.*/\1/' | grep '^Setting'); do
     grep -oE "func \(c \*client\) (Get|Update)${r}\(" unifi/*.generated.go \
       | sed -E "s/.*\) (Get|Update)${r}\(/\1/" | sort -u | tr '\n' ' '
     echo "  <- $r"
   done
   ```

   A singleton's natural lifecycle is **Get + Update** — that IS its full scope,
   so a setting with both is ✅ on the legacy surface (see Step 4).

A type missing `create`/`update`/`delete` (for managed) is read-only on the
legacy surface, which Step 4 must reflect.

### Step 3 — Build the FULL resource union (omit nothing)

Produce the complete row set across both surfaces. The guiding rule is
**coverage, not curation**: start from the assumption that every legacy type and
every official resource gets a row, then only ever *merge* — never *drop*.

Merging rules:

- **Merge two surfaces into ONE row when they model the same real-world
  resource, even under different names.** Verify sameness by inspecting the
  models, not the labels. Known equivalences (re-verify each run):
  - Legacy `Network` ↔ Official `Networks` → "Networks"
  - Legacy `WLAN` ↔ Official `WifiBroadcasts` → "WiFi Networks (SSIDs)"
  - Legacy `FirewallZone` ↔ Official `Firewall.Zone` → "Firewall Zones"
  - Legacy `FirewallZonePolicy` ↔ Official `Firewall.Policy` → "Firewall Policies"
  - Legacy `DNSRecord` ↔ Official `DNSPolicies` → "DNS Records" — the official
    `DNSPolicy` union variants are `DnsARecord` / `DnsAaaaRecord` /
    `DnsCnameRecord` / … i.e. the SAME A/AAAA/CNAME records the legacy
    `DNSRecord` manages; only the naming differs. Note this in Comments.
  - Legacy `User` ↔ Official `Clients` → "Users (Network Clients)"
  - Legacy `Device` ↔ Official `Devices` → "Devices"
  When in doubt whether two are the same resource, open both models/structs and
  compare fields before merging. Different models that merely share a topic
  (e.g. `FirewallRule` vs `FirewallZonePolicy`) stay as SEPARATE rows.

- **Every remaining legacy type gets its own row** — including ones with no
  official counterpart: AP Groups, RADIUS Accounts/Profiles, Port Forwarding,
  Routing, Port Profiles, DHCP Options, Dynamic DNS, Content Filtering, Firewall
  Rules, Firewall Groups, Broadcast Groups, Channel Plan, Hotspot 2.0 / packages
  / operators, Maps, Heat Maps, Heat-Map Points, Spatial Records, Media Files,
  Schedule Tasks, Tags, Virtual Devices, WLAN Groups, User Groups, Dashboard,
  and **all 43 `Setting*` singletons**.

- **Every official-only resource gets its own row** — ACL Rules, Traffic
  Matching Lists, Hotspot Vouchers, Controller Info, Sites.

- The ONLY things you omit: the `Supporting` helper group (Step 1) and any future
  pure enum/lookup helper. Do not omit anything else — not settings, not "niche"
  types, not read-only resources. If you are tempted to drop a row, you are
  wrong; give it a row.

**Reconciliation check (do this before writing the doc):** take the legacy list
from Step 2 and the official resource list from Step 1, and confirm each name is
accounted for in the row set — either as its own row or as a named half of a
merged row. If any legacy type or official resource is unrepresented, you missed
it; add it. Settings are reconciled as a block (all 43 in the Settings section).

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
   listing/reading IS the full surface there. For a **settings singleton**, the
   natural lifecycle is **Get + Update** — both present ⇒ ✅.
3. **Map evidence → mark** using this rubric:

| Mark | Precise meaning |
|------|-----------------|
| ✅   | Every operation in the resource's expected lifecycle is present on this surface (full CRUD for a managed resource; Get+Update for a singleton; the complete read/action set for a read-only resource). |
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
- **Merged rows where the naming differs across surfaces** — note the alias
  (e.g. DNS Records being exposed as "DNS Policies" on the Official side).

Leave the Comments cell **blank** for rows where both surfaces are ✅ with
matching names, or where only one surface has ✅ and the other ❌ without
noteworthy explanation (e.g. routine legacy-only resources and settings).

### Step 6 — Emit the document

Write `docs/2.0.0/coverage_matrix.md` with:

1. A short intro paragraph stating the matrix is exhaustive over both surfaces.
2. The emoji legend table (✅/⚠️/❌).
3. The coverage tables, **split into category sections** (one `###` heading +
   one table per category). Recommended categories (adapt as the surfaces
   evolve, but keep every resource in exactly one section):
   - Networking
   - Firewall & Security
   - WiFi
   - Devices & Ports
   - Clients & Users
   - Hotspot & Guest
   - Maps, Floorplans & Media
   - System & Operations
   - Settings (singleton configuration) — the 43 `Setting*` types, kept in their
     own dedicated sub-table at the end.
4. Every table uses the columns `| Resource | Official API | Legacy Internal API | Comments |`.
   - Sort rows alphabetically by Resource name **within each section**.
   - Official/Legacy cells contain ONLY an emoji mark (no text).
   - Comments are plain prose, no markdown formatting inside the cell.

The format must be stable and consistent across runs so that diffs remain
readable. Use compact pipe tables (single space padding, no column-alignment
padding, no trailing whitespace). Do not vary column widths, reorder columns, or
reorder sections between runs.

---

## Constraints

- **Exhaustive coverage is mandatory.** Every legacy resource type from Step 2
  and every official resource from Step 1 (except `Supporting`) must appear in
  the matrix. Run the Step 3 reconciliation check before emitting.
- **NEVER fabricate resource or group names.** Every row must be grounded in a
  real type or group found in Steps 1–2.
- **Supporting is intentionally excluded** from the table. Do not add a row for
  it or document it as skipped; just omit it.
- Settings are included (grouped in their own section), never dropped.
- Do NOT add the coverage matrix link to `docs/compatibility_matrix.md`.
- The document is human-facing only; no Go code or CI test parses it.
- Regenerate the WHOLE document on each run — do not patch individual rows.
