/*
 * docs-update-verify — orchestrate UPDATING and VERIFYING the go-unifi documentation website.
 *
 * WHAT IT DOES
 *   The docs site lives in website/ (Fumadocs; MDX under website/content/docs/<section>/). When the library
 *   API or the committed OpenAPI spec changes, the affected pages must be UPDATED and the whole site
 *   re-VERIFIED: accuracy against source, every Go snippet compiles, internal links/#anchors resolve, MDX
 *   structure is valid, and cross-document facts stay consistent. This workflow does both, in four phases:
 *     Setup       (always) regenerate the go doc dumps + (re)bootstrap the snippet scratch module, and — when
 *                 the change touches the OpenAPI surface — regenerate website/lib/go-crosswalk.ts.
 *     Update      (only when a change is described) triage which sections are affected, then one updater agent
 *                 per affected section edits its MDX pages (disjoint files) keeping snippets compiling.
 *     Verify      one report-only reviewer per section + one whole-site adversarial reviewer (structured output).
 *     Synthesize  consolidate every finding into one de-duplicated, severity-grouped fix list (returned).
 *
 * HOW TO INVOKE
 *   Workflow({ name: 'docs-update-verify', args: 'Official API added a Foo() group' })  // UPDATE + VERIFY
 *   Workflow({ name: 'docs-update-verify' })                                            // VERIFY-ONLY (Update skipped)
 *   `args` is a free-text description of what changed (e.g. "spec bumped to 10.2.0"). Empty/undefined =>
 *   verify-only: the Update phase is skipped entirely and the site is fully audited as-is.
 *
 * NOTE: every run regenerates the go doc dumps (/tmp/gu-api/*.txt) and refreshes the scratch Go module
 *   (/tmp/gu-docs-snippets) so snippet compilation checks always run against the current source.
 */

export const meta = {
  name: 'docs-update-verify',
  description: 'Update (when a change is described) and always verify the go-unifi docs website: accuracy vs source, compiling Go snippets, resolving links/anchors, valid MDX, cross-doc consistency.',
  whenToUse: 'After a library API or committed OpenAPI spec change, to update the affected docs pages and re-verify the whole site. Run with no args for a verify-only full audit.',
  phases: [
    { title: 'Setup', detail: 'regenerate go doc dumps + scratch snippet module; regenerate go-crosswalk.ts when the OpenAPI surface changed' },
    { title: 'Update', detail: 'triage affected sections, then one updater agent per section edits its MDX (skipped in verify-only mode)' },
    { title: 'Verify', detail: 'one report-only reviewer per section + a whole-site adversarial reviewer (structured findings)' },
    { title: 'Synthesize', detail: 'dedup + severity-group all findings into one actionable fix list' },
  ],
}

// ---------------------------------------------------------------------------
// Constants — paths are relative to the repo root (the agents' working dir) unless absolute.
// ---------------------------------------------------------------------------
const REPO = '/Users/filipowm/Documents/dev/workspaces/unifi/go-unifi'
const DOCS = 'website/content/docs'
const CROSSWALK = 'website/lib/go-crosswalk.ts'
const GO = '/opt/homebrew/opt/go/bin/go'         // Homebrew Go — the shell `go` is too old for go 1.26.0.
const SNIP = '/tmp/gu-docs-snippets'             // scratch module for compiling MDX Go snippets.
const DUMPS = '/tmp/gu-api'                       // authoritative go doc dumps live here.
const VET_ALL = `${GO} -C ${SNIP} vet -buildvcs=false ./...`
const vetSub = (sub) => `${GO} -C ${SNIP} vet -buildvcs=false ./${sub}/...`

// ---------------------------------------------------------------------------
// Findings schema — defined ONCE, reused by every reviewer (and the adversarial reviewer).
// ---------------------------------------------------------------------------
const FINDINGS = {
  type: 'object',
  additionalProperties: false,
  properties: {
    summary: { type: 'string', description: 'one-line overall verdict for this section' },
    findings: {
      type: 'array',
      items: {
        type: 'object',
        additionalProperties: false,
        properties: {
          file: { type: 'string', description: 'path relative to repo root, or "OTHER:<path>"' },
          severity: { type: 'string', enum: ['blocker', 'major', 'minor', 'nit'] },
          category: { type: 'string', enum: ['accuracy', 'compile', 'link', 'anchor', 'structure', 'consistency', 'other'] },
          issue: { type: 'string', description: 'what is wrong, with evidence (source file:line for accuracy)' },
          fix: { type: 'string', description: 'concrete suggested fix' },
        },
        required: ['file', 'severity', 'category', 'issue', 'fix'],
      },
    },
  },
  required: ['summary', 'findings'],
}

// ---------------------------------------------------------------------------
// Shared BASE — repo facts, vet command, MDX rules, never-invent-API. Both updaters and reviewers build on it.
// ---------------------------------------------------------------------------
const BASE = `You are working on the go-unifi documentation website (Fumadocs). Your working directory IS the repo root (${REPO}).

REPO FACTS / KEY PATHS (relative to the repo root unless absolute):
- Docs MDX: ${DOCS}/<section>/*.mdx. Sidebar sections: getting-started, guides, advanced, reference, migrating, developers (order via meta.json files).
- The OpenAPI reference pages under ${DOCS}/reference/api are GENERATED at build time from the committed spec; each operation page's "Go (go-unifi)" sample comes from ${CROSSWALK} (operationId -> Go call), which must stay in sync with the spec + unifi/official/*.generated.go.
- Committed OpenAPI spec: codegen/openapi/integration-*.json (currently integration-10.1.78.json).
- Go source: unifi/** (Internal/legacy client) and unifi/official/** (Official OpenAPI client; the unifi -> official dependency is one-way). Codegen pipeline: codegen/**. Also relevant: Makefile, CLAUDE.md, .claude/rules/.
- Authoritative API signatures are the go doc dumps: ${DUMPS}/unifi.txt, ${DUMPS}/unifi_official.txt, ${DUMPS}/unifi_features.txt. Grep the dumps and the source; NEVER invent or guess an API — every method/type/field/constant/version you cite must exist.

GO SNIPPET COMPILATION (always use Homebrew Go — the shell go is too old):
- A scratch Go module is bootstrapped at ${SNIP} (module gudocs, go 1.26.0, replace github.com/filipowm/go-unifi/v2 => the repo root, require github.com/google/uuid v1.6.0).
- Materialize each Go example as a .go file under a per-section subdir ${SNIP}/<scratch>/ and verify with: ${GO} -C ${SNIP} vet -buildvcs=false ./<scratch>/...  (exit 0 = clean). Whole-module check: ${VET_ALL}.
- NEVER run "go mod tidy" (only the Setup phase does that, with network). The Go shown in MDX must match exactly what compiles.

MDX RULES:
- Frontmatter MUST have title + description. No body H1 (the title renders the page H1).
- Globally-available components (no imports): Callout, Card/Cards, Steps/Step, Tabs/Tab, Accordion/Accordions, TypeTable.
- Internal links are absolute /docs/... paths; every link target and every #anchor must resolve (heading slug = lowercase with non-alphanumerics turned to hyphens, via github-slugger).

CANONICAL FACTS (must read identically across all pages): API-key floor controller 9.0.114; Official API (integration/v1) floor 10.1.78; Go 1.26; auth header X-Api-Key; the module path is /v2. Two surfaces: Internal (default; methods on the client and via c.Internal(); site-name keyed) vs Official (c.Official(); uuid.UUID keyed).`

// ---------------------------------------------------------------------------
// Section catalog — shared by Update (triage + fan-out) and Verify (reviewers).
// Each section owns a DISJOINT set of files and its own scratch subdir.
// ---------------------------------------------------------------------------
const SECTIONS = [
  {
    label: 'getting-started',
    scratch: 'getting',
    files: `${DOCS}/getting-started/*.mdx (index, installation, authentication, connecting, quickstart)`,
    focus: 'install + import paths (unifi, unifi/official, unifi/features), Go 1.26, the /v2 module, API-key auth (X-Api-Key, controller floor 9.0.114, ErrOldStyleUnsupported on old controllers), building the client (ClientConfig basics), the two surfaces.',
    sources: 'docs/getting_started.md, README.md, unifi/client.go (ClientConfig + NewClient doc comments), unifi/api_paths.go (ApiKeyHeader/X-Api-Key), unifi/unifi_errors.go (ErrOldStyleUnsupported), and ' + DUMPS + '/unifi.txt',
    extra: '',
  },
  {
    label: 'guides',
    scratch: 'guides',
    files: `${DOCS}/guides/*.mdx (all guide pages — surfaces, pagination/filtering, sites, networks, devices, clients-and-users, firewall, wireless, settings, feature-flags, file-uploads, traffic-flows, error-handling, testing)`,
    focus: 'task-oriented examples whose Internal AND Official method names/signatures are exact; choosing-a-surface; official pagination (List...Page -> official.Page[T], Limit clamped to 200; List...All -> iter.Seq2; official.Collect; the filter DSL); the site-name vs uuid.UUID duality.',
    sources: 'unifi/*.go resource files (network.go, device.go, user.go, firewall_*.go, wlan_*.go, setting*.go, traffic_flow.go, portalfile.go, ...), unifi/official/*.go (official.go, pagination.go, sites.go, ...), README.md, and ' + DUMPS + '/unifi.txt + ' + DUMPS + '/unifi_official.txt',
    extra: '',
  },
  {
    label: 'advanced',
    scratch: 'advanced',
    files: `${DOCS}/advanced/*.mdx (index, configuration, raw-http, interceptors, logging, validation, concurrency, compatibility, troubleshooting)`,
    focus: 'ClientConfig field semantics (Timeout, UserAgent, HttpTransportCustomizer vs HttpRoundTripperProvider precedence, ErrorHandler, CustomValidators); raw Do/Get/Post/Put/Patch/Delete + path resolution (new vs old API style); interceptor precedence and the body-read hazard; Logger/LoggingLevel; validation modes; goroutine-safety (UseLocking is a deprecated no-op); the compatibility matrix numbers; the json.go custom-unmarshaler quirks.',
    sources: 'unifi/client.go, unifi/requests.go, unifi/interceptors.go, unifi/api_paths.go, unifi/logging.go, unifi/validation.go, unifi/json.go, docs/configuration.md, docs/advanced_topics.md, docs/compatibility_matrix.md, and ' + DUMPS + '/unifi.txt',
    extra: '',
  },
  {
    label: 'reference',
    scratch: 'reference',
    files: `${DOCS}/reference/*.mdx (index, client, configuration-types, errors, official, resources, settings, feature-constants) + ${DOCS}/reference/api/index.mdx (OpenAPI overview) + ${CROSSWALK} (the Go crosswalk)`,
    focus: 'exhaustive-claim accuracy: the Client interface (embeds InternalClient; adds Logger/BaseURL/Version/Do.../Internal()/Official()), the ClientConfig field list + enums, the error types/sentinels, the curated resource->method map, the typed Setting pairs, the features.* constants — plus the OpenAPI overview prose and the operationId->Go crosswalk.',
    sources: 'unifi/client.go, unifi/unifi_errors.go, unifi/validation.go, unifi/logging.go, unifi/official/** (+ *.generated.go), unifi/setting.go, unifi/setting_registry.go, unifi/features/*, codegen/openapi/integration-*.json, and ' + DUMPS + '/*.txt',
    extra: `OPENAPI OVERVIEW + CROSSWALK are part of THIS section:
- ${DOCS}/reference/api/index.mdx prose must match the committed spec codegen/openapi/integration-*.json (runtime base path /proxy/network/integration/v1, auth X-Api-Key, the filter DSL, the error model).
- ${CROSSWALK} maps each spec operationId to a Go (go-unifi) call and must stay in sync with the spec + unifi/official/*.generated.go. To check it, materialize its Go samples as .go files under ${SNIP}/crosswalk/ and run ${vetSub('crosswalk')} (exit 0). Spot-check operationId->Go correctness against the "// <Name> maps to <METHOD> /<path> on the Official API" comments in unifi/official/*.generated.go (right group accessor, method, arg arity).`,
  },
  {
    label: 'developers-migrating',
    scratch: 'devmig',
    files: `${DOCS}/developers/*.mdx (index, code-generation, customizations, regenerating, testing, release-process, contributing) + ${DOCS}/migrating/*.mdx (index, from-1.x, breaking-changes, from-paultyng)`,
    focus: 'codegen pipeline (generated *.generated.go vs hand-written split; codegen/internal/customizations.yml; two version axes .unifi-version 9.5.21 + .unifi-version-official 10.1.78); exact regen commands; testing/release/contributing conventions; faithful migration guides (1.x->2.0, from-paultyng) and the 2.0.0 breaking-changes log.',
    sources: 'codegen/** (codegen/CLAUDE.md, internal/*, official/*), unifi/codegen.go, Makefile, CLAUDE.md, .claude/rules/*, .github/workflows/*, .goreleaser.yaml, docs/2.0.0/*.md, docs/migrating_from_upstream.md',
    extra: '',
  },
]

// ---------------------------------------------------------------------------
// Mode + run context (computed before any prompt that references it).
// ---------------------------------------------------------------------------
const changeDesc = (args && String(args).trim()) ? String(args).trim() : ''
const mode = changeDesc ? 'update+verify' : 'verify-only'
const changeLine = changeDesc || 'VERIFY-ONLY: no specific change was supplied — audit the entire site for accuracy and consistency.'

// ---------------------------------------------------------------------------
// Prompt builders — updaters EDIT, reviewers REPORT ONLY; both share BASE + the section spec.
// ---------------------------------------------------------------------------
const sectionSpec = (s) => `ASSIGNED FILES (you own ONLY these — disjoint from other agents): ${s.files}
SECTION FOCUS: ${s.focus}
SOURCES TO READ (authoritative — grep before you write): ${s.sources}
${s.extra ? s.extra + '\n' : ''}`

const updaterPrompt = (s) => `${BASE}

ROLE: UPDATER for the "${s.label}" docs section. You EDIT files. Apply the change below to the relevant pages, then leave them accurate, compiling, and structurally valid. Do NOT touch other sections' files, any meta.json, or anything outside your file list and your scratch subdir ${SNIP}/${s.scratch}/.

WHAT CHANGED (apply this): ${changeDesc}

${sectionSpec(s)}
DO:
1. Read the affected MDX pages and the sources/dumps. Update prose, snippets, tables, TypeTables, and cross-links so they reflect the change. If a page is genuinely unaffected, leave it unchanged.
2. Keep EVERY Go snippet compiling: write each example as a .go file under ${SNIP}/${s.scratch}/ and run ${vetSub(s.scratch)} until it exits 0. The MDX Go must match exactly what compiled. Never invent API — grep unifi/** or the dumps.
3. Keep frontmatter (title + description), components, and absolute /docs/... links valid; no body H1.
Return a short report: files edited, snippets compiled (with the final clean vet output), and anything you could NOT verify against source.`

const reviewerPrompt = (s) => `${BASE}

ROLE: REVIEWER for the "${s.label}" docs section. REPORT ONLY — do NOT edit any file. Be adversarial and concrete; cite source file:line for every accuracy claim.

CONTEXT — what recently changed (give it extra scrutiny, but audit the WHOLE section): ${changeLine}

${sectionSpec(s)}
CHECK each assigned page:
(a) ACCURACY — every method/type/field/constant/version exists with the stated signature/behavior (grep the dumps ${DUMPS}/*.txt and unifi/**). Flag hallucinations and wrong claims.
(b) COMPILE — extract each Go snippet, exactly as written in the MDX, into ${SNIP}/${s.scratch}/ and run ${vetSub(s.scratch)}; flag any snippet that would not compile. Do NOT run go mod tidy.
(c) LINKS & ANCHORS — every internal /docs/... target exists; every #anchor resolves to a real heading on the target page (read target pages to confirm headings).
(d) STRUCTURE — frontmatter title + description present; no body H1; only allowed components; no stray imports.
Return findings via the structured-output schema. Report ONLY real problems (don't pad). severity: blocker (broken/wrong and user-visible), major (incorrect claim), minor (clarity/polish), nit (cosmetic). category: accuracy|compile|link|anchor|structure|consistency|other.`

const adversarialPrompt = () => `${BASE}

ROLE: WHOLE-SITE ADVERSARIAL REVIEWER. REPORT ONLY — do NOT edit any file. You did NOT author any of this; try to break it.

CONTEXT — what recently changed: ${changeLine}

1. Pick ~30 specific, FALSIFIABLE claims spread across getting-started / guides / advanced / reference / developers / migrating (method signatures, field names, version numbers, behavioral assertions like "returns ErrNotFound", "Limit clamped to 200", "fails at construction"). TRY TO FALSIFY each against unifi/**, unifi/official/**, and the dumps ${DUMPS}/*.txt. Report ONLY confirmed discrepancies (category accuracy), each with its source file:line.
2. CROSS-DOCUMENT CONSISTENCY (category consistency): are the version floors (9.0.114 API-key, 10.1.78 Official), Go 1.26, the /v2 module, and the Internal-vs-Official framing stated consistently everywhere? Flag any contradiction between two pages, naming BOTH files.
3. CROSSWALK spot-check: confirm ~8 ${CROSSWALK} operationId->Go mappings match the "// <Name> maps to <METHOD> /<path> on the Official API" comments in unifi/official/*.generated.go (right group accessor, method, arg arity).
Return findings via the structured-output schema (same severities/categories as the section reviewers).`

// ---------------------------------------------------------------------------
// Triage schema — which sections the change affects (enum kept in sync with SECTIONS).
// ---------------------------------------------------------------------------
const SECTIONS_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  properties: {
    sections: { type: 'array', items: { type: 'string', enum: SECTIONS.map((s) => s.label) }, description: 'labels of the affected sections' },
    crosswalkAffected: { type: 'boolean', description: 'true if the OpenAPI spec / Official surface (and thus go-crosswalk.ts) is affected' },
    rationale: { type: 'string', description: 'one line: why these sections' },
  },
  required: ['sections', 'rationale'],
}

// ===========================================================================
// Phase 1 — Setup (always runs; Verify needs the dumps + scratch module too).
// ===========================================================================
phase('Setup')
log(`Mode: ${mode}. ${changeDesc ? 'Change: ' + changeDesc : 'No change supplied — verify-only audit.'}`)

const SETUP_PROMPT = `${BASE}

ROLE: SETUP for a docs update+verify run. Mode: ${mode}. Prepare the shared verification substrate the later agents depend on. Run every command from the repo root (your working directory).

(a) REGENERATE the go doc dumps (authoritative API signatures):
  mkdir -p ${DUMPS}
  ${GO} doc -all ./unifi > ${DUMPS}/unifi.txt
  ${GO} doc -all ./unifi/official > ${DUMPS}/unifi_official.txt
  ${GO} doc -all ./unifi/features > ${DUMPS}/unifi_features.txt
  Confirm each file is non-empty.

(b) BOOTSTRAP/REFRESH the scratch Go module at ${SNIP} (used to compile every MDX Go snippet):
  - Ensure ${SNIP} exists. Write ${SNIP}/go.mod containing EXACTLY these directives:
      module gudocs
      go 1.26.0
      require github.com/google/uuid v1.6.0
      replace github.com/filipowm/go-unifi/v2 => <REPO_ROOT_ABS>
    where <REPO_ROOT_ABS> is the absolute path of the repo root (your current working directory; resolve it with pwd — it should be ${REPO}).
  - Create a tiny warm-up package ${SNIP}/_bootstrap/bootstrap.go that imports github.com/filipowm/go-unifi/v2/unifi, github.com/filipowm/go-unifi/v2/unifi/official, github.com/filipowm/go-unifi/v2/unifi/features and github.com/google/uuid with a blank reference (e.g. var _ = uuid.Nil), so go mod tidy retains and PRE-DOWNLOADS these deps for the later (offline) per-section snippet compiles.
  - Run: ${GO} -C ${SNIP} mod tidy   (network IS allowed here, and ONLY here). Confirm it succeeds and uuid + go-unifi/v2 resolve, then sanity-check: ${VET_ALL} should pass on the warm-up package.
  - Leave any existing per-section subdirs (getting, guides, advanced, reference, devmig, crosswalk) in place; later agents recreate their own .go files there.

(c) CROSSWALK — ONLY if the change touches the OpenAPI spec or the Official API surface (decide from the change description below; in verify-only mode SKIP this step):
  CHANGE DESCRIPTION: ${changeDesc || '(none — verify-only)'}
  If it indicates a spec bump or an Official-surface change:
   - Regenerate ${CROSSWALK} by joining the spec operationIds (from codegen/openapi/integration-*.json) to the generated Official methods in unifi/official/*.generated.go. Each generated method carries a comment "// <Name> maps to <METHOD> /<path> on the Official API" — use it to map each operationId to its Go call (group accessor + method + args). Preserve the file's existing TypeScript shape and exports.
   - Verify the Go samples compile: materialize them as .go files under ${SNIP}/crosswalk/ and run ${vetSub('crosswalk')} (exit 0).

Return a SHORT readiness report: dump sizes, whether mod tidy succeeded, and whether the crosswalk was regenerated (with its vet result) or skipped (and why).`

const setupReport = await agent(SETUP_PROMPT, { label: 'setup', phase: 'Setup' })

// ===========================================================================
// Phase 2 — Update (guarded: only when a non-empty change description was supplied).
// ===========================================================================
let updateReports = []
if (changeDesc && String(changeDesc).trim()) {
  phase('Update')

  const triage = await agent(
    `${BASE}

ROLE: UPDATE TRIAGE. A change was made to the library/spec. Decide WHICH docs sections plausibly need editing, so we only fan out updaters where needed.

CHANGE DESCRIPTION: ${changeDesc}

SECTION CATALOG (label -> what it covers):
${SECTIONS.map((s) => `- ${s.label}: ${s.focus}`).join('\n')}

Read the change (and skim the relevant source/dumps if needed). Return the labels of the AFFECTED sections — be inclusive but not exhaustive; if unsure whether a section is affected, include it. Also flag whether the OpenAPI crosswalk/spec is affected, with a one-line rationale.`,
    { label: 'triage', phase: 'Update', schema: SECTIONS_SCHEMA },
  )

  const picked = triage && Array.isArray(triage.sections) ? triage.sections : []
  let toUpdate = SECTIONS.filter((s) => picked.includes(s.label))
  if (!toUpdate.length) {
    log('Triage selected no sections — updating all as a safe default.')
    toUpdate = SECTIONS
  }
  log(`Updating sections: ${toUpdate.map((s) => s.label).join(', ')}${triage && triage.rationale ? ' (' + triage.rationale + ')' : ''}`)

  updateReports = (await parallel(
    toUpdate.map((s) => () => agent(updaterPrompt(s), { label: `update:${s.label}`, phase: 'Update' }).then((r) => ({ section: s.label, report: r }))),
  )).filter(Boolean)
}

// ===========================================================================
// Phase 3 — Verify (parallel report-only reviewers: one per section + whole-site adversarial).
// ===========================================================================
phase('Verify')

const REVIEWERS = SECTIONS.map((s) => ({ label: s.label, prompt: reviewerPrompt(s) }))
REVIEWERS.push({ label: 'adversarial', prompt: adversarialPrompt() })

const reviews = (await parallel(
  REVIEWERS.map((r) => () =>
    agent(r.prompt, { label: `review:${r.label}`, phase: 'Verify', schema: FINDINGS }).then((res) => ({ section: r.label, ...res })),
  ),
)).filter(Boolean)

// ===========================================================================
// Phase 4 — Synthesize (one agent consolidates all findings into a prioritized fix list).
// ===========================================================================
phase('Synthesize')

const bundle = reviews
  .map((r) => `### ${r.section} — ${r.summary || ''}\n` + (r.findings || []).map((f) => `- [${f.severity}/${f.category}] ${f.file}: ${f.issue}\n  FIX: ${f.fix}`).join('\n'))
  .join('\n\n')

const synth = await agent(
  `You are consolidating the Verify-phase review findings for the go-unifi docs site into ONE prioritized, de-duplicated fix list for the maintainer to apply.

RUN MODE: ${mode}. CHANGE: ${changeDesc || '(verify-only — full audit)'}.

Do this:
- Drop false positives (a "finding" that is actually correct as written) and merge duplicates / overlapping items across reviewers.
- Group the kept findings by severity: BLOCKER, MAJOR, MINOR, NIT. Within each, list: the file, the precise problem (with its source evidence), and the exact fix (old -> new where possible).
- End with: a one-paragraph overall quality assessment, the total counts per severity, and a "Rejected (false positives)" section noting any reviewer mistakes and why.
Return the result as Markdown.

RAW FINDINGS BY SECTION:

${bundle}`,
  { label: 'synthesize', phase: 'Synthesize' },
)

return synth
