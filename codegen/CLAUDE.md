# codegen — UniFi API code generation

Generates the `*.generated.go` files in `../unifi/` from the UniFi controller's own API field definitions. Triggered by `go generate unifi/codegen.go`, 
which runs: `go run ../codegen/ -version-base-dir=../codegen/ <version>`.

## Pipeline

1. `download.go` — downloads the controller `.deb` from dl.ui.com, extracts `data.tar.xz` → `ace.jar` → JSON field definitions (`api/fields/*.json`).
   `Setting.json` is split into per-setting files.
   1a. **Official OpenAPI spec source** (`download.go` `DownloadAndExtractOfficialSpec` + `version.go` `OfficialSpecURL`) — alongside the internal `.deb`, fetches the
   UniFi OS Server package `unifi-uos_sysvinit.deb` (same dl.ui.com path, different filename, keyed by the same version), extracts `integration.json` (the
   Official-API OpenAPI 3.1 spec) from `./usr/lib/unifi/webapps/ROOT/api-docs/integration.json` in its `data.tar.xz`, and commits a **byte-for-byte pinned
   snapshot** at `codegen/openapi/integration-<ver>.json`. The versioned filename is the pin (mirrors `.unifi-version`); the committed snapshot makes generation
   deterministic and surfaces the spec delta in PR diffs. Packages predating the Official API (< 10.1.78, no `integration.json`) are skipped with a warning so the
   internal pipeline never regresses; downstream OpenAPI codegen stages (#121) consume the committed snapshot, not a live fetch.
2. `resources.go` — parses each JSON into a Resource; infers Go types from the field validation regexes; snake_case → CamelCase (acronyms via `fieldReps`).
3. `customize.go` — applies `customizations.yml` field overrides.
4. `generator.go` — renders `api.go.tmpl` / `apiv2.go.tmpl` → `<resource>.generated.go`; writes `version.generated.go` and the repo `.unifi-version` marker.

### Client interface split (`client.go.tmpl` + `clients.go`)

`client.go.tmpl` renders **two** interfaces into `client.generated.go`: an embedded `InternalClient` (all
resource CRUD — every function carrying a resource name) and the top-level `Client` (which embeds
`InternalClient`, then lists the transport/lifecycle functions — those with an *empty* resource name — and
the hand-written `Internal()`/`Official()` accessors). The split is driven by `ClientInfo.ResourceFunctions`
/ `ClientInfo.TransportFunctions` in `clients.go`, keyed on `ClientFunction.ResourceName()`. The template
always imports `unifi/official` for the `Official()` accessor return type. After changing the split,
regenerate `client.generated.go` **and** `client_mock.generated.go` (offline moq — see `unifi/mock.go`).

## Official-API frontend — OpenAPI models (`codegen/official/`)

`codegen/official/` is a **separate Go module** (`github.com/filipowm/go-unifi/codegen/official`) that hosts the
OpenAPI toolchain (`oapi-codegen/v2` + `kin-openapi`). It is isolated so those heavy deps never enter the root
`go.mod`; the only root addition is `github.com/oapi-codegen/runtime` (imported by the generated models). The
existing `.deb` pipeline stays in the root module and `go run ../codegen/` is unaffected.

It reads the **committed** snapshot (`codegen/openapi/integration-<ver>.json`) — fully **offline**, no
controller download — and writes `unifi/official/models.generated.go` (package `official`, `DO NOT EDIT`).
Regenerate with:

```sh
cd codegen/official && go run .            # defaults: --openapi-dir ../openapi --out ../../unifi/official/models.generated.go
```

### Why a transform (the oneOf bridge)

oapi-codegen's allOf+discriminator path **silently drops every variant struct**. The transform (`transform.go`,
`naming.go`) rewrites the spec into oapi-codegen's **oneOf** path, which emits per-variant union types with full
machinery. Steps (all on the raw JSON, deterministic, fail-loud):

1. **Downconvert 3.1 → 3.0.3** — patches only the `openapi` string after asserting the spec uses **zero**
   3.1-only constructs (type arrays, `prefixItems`, `const`, `unevaluated*`, …); the bump is lossless.
2. **UPPER_SNAKE mapping assert** — every `discriminator.mapping` key must be the wire enum value, so the
   generated `ValueByDiscriminator()` switch decodes real payloads.
3. **Enum dedup** — a curated table (`sharedEnums`) hoists a value-set shared across a tri-shape family into one
   type (e.g. `ACLRuleAction`), collapsing oapi-codegen's per-schema duplicates. Cross-family identical sets
   (e.g. ALLOW/BLOCK on Wi-Fi, the protocol-name set across IPv4/IPv6) are left distinct on purpose.
4. **Diamond fix** — a variant whose `allOf` extends 2+ discriminator parents would make oapi-codegen fail
   ("cannot merge two discriminators"). Keep the first parent ref; inline the rest's properties minus their
   discriminator (lossless — those bases contribute fields, not a second union).
5. **oneOf synthesis** — each discriminator parent gets a `oneOf` over ALL variants, mined from BOTH allOf
   back-references AND the `discriminator.mapping` (some variants appear only in the mapping). Member list is
   sorted; the synthesized graph is cycle-checked.
6. **Naming + collision recompute** — strip `Integration` prefix / `Dto` suffix, flip `Create or update X` →
   `XCreateOrUpdate`, then oapi-codegen's own camel-casing (so our names match its output). Collisions are
   recomputed on the POST-transform set and must resolve to unique names or generation fails. One explicit
   override: `IP Address selector` (a variant) → `SpecificIPAddressSelector`, freeing `IPAddressSelector` for
   its parent union.
7. **Header normalize** — collapse oapi-codegen's `// Package …` doc to a bare `DO NOT EDIT` banner so the
   hand-written package doc in `official.go` stays the single package godoc.

### Generated public API shape

Each polymorphic parent becomes a union struct: the discriminator field + an unexported `union json.RawMessage`,
plus `AsXxx/FromXxx/MergeXxx`, `Discriminator()`, `ValueByDiscriminator()` (switching on the UPPER_SNAKE wire
values) and generated `MarshalJSON/UnmarshalJSON`. Variants with own fields are full structs (also carrying the
union machinery merged from the parent); **empty variants are Go type aliases of the parent** — branch on
`Discriminator()` to tell them apart. No silent field loss on any family (management / firewall / Wi-Fi security).
The leaf-value unmarshalers in `unifi/json.go` are unaffected — the union codec applies only to discriminated
parents.

### Hand-written collisions

`package official` already hand-writes `SiteOverview` and `Info`. The transform defers to them: `Site overview`
is excluded from generation (refs resolve to the hand-written type) and `Application info` becomes
`type ApplicationInfo = Info`.

> Stage 3 folds this frontend into `generator.go`'s second pass and adds the tri-shape wrappers + `official.Client`.
> Until then it is a standalone runnable.

## Download trust model (ARCH-15 / ARCH-16)

The download pipeline (`download.go`, `version.go`) is the only point where codegen
ingests remote, code-influencing data. Guards in place:

- **Cancellation/timeouts:** `DownloadAndExtract` / `downloadJar` take a `context.Context`;
  `generate()` passes a bounded one. A nil/timeout-less injected client gets a default
  timeout (`defaultDownloadTimeout`), and the firmware-latest call uses `firmwareApiTimeout`.
  A hung dl.ui.com / fw-update.ubnt.com now fails cleanly instead of stalling the CI job.
- **Host/scheme pinning:** `validateDownloadURL` requires `https` on a Ubiquiti host
  (`ui.com` / `ubnt.com`, host-or-`*.suffix`) before any fetch; loopback hosts are exempt
  for the offline httptest seam. `Latest()` also re-validates `channel==release` /
  `product==unifi-controller` locally rather than trusting server-side filtering.
- **Atomic extraction:** extraction runs in a sibling `*.tmp-*` dir that is `os.Rename`d
  into place only after a fully-successful extract, with a `.extract-complete` sentinel
  written last. A version dir lacking the sentinel (a crashed prior run) is treated as
  incomplete and re-extracted — a partial tree is never silently accepted. The Official-spec
  snapshot is a single file, so it uses a temp-file-`os.Rename` publish (no sentinel — one
  rename is itself atomic) and the same `copyWithLimit` cap (`maxOpenAPISpecSize`); the deb
  fetch shares the internal `withDebDataTar` helper, so host pinning + timeouts apply identically.
- **NOT yet pinned:** there is no checksum/signature verification of the `.deb` — the
  firmware API exposes no checksum to verify against. Provenance rests on HTTPS + host
  pinning + the `api/fields/*.json` allowlist + size caps (`copyWithLimit`). To harden
  further, pin known-good SHA256s per supported version alongside `.unifi-version`.

## Versioning

- `codegen/v<X.Y.Z>/` holds the JSON field defs per controller version (`v2/` = V2 API resources, rendered with `apiv2.go.tmpl`, different endpoints). It is a
  download cache (`.gitignore`d), unlike `codegen/openapi/integration-<ver>.json` which is a **committed** snapshot.
- `.unifi-version` (repo root) and the version arg in `unifi/codegen.go` pin the supported **internal** version.

### Two-version model (internal vs Official-API spec)

The internal resource-gen version and the Official-API spec version are **intentionally decoupled** and may differ:

| Pin | Controls | Example |
|---|---|---|
| `.unifi-version` / `go generate` arg | Internal `.deb` download (field JSONs → generated resources) | `9.5.21` |
| `codegen/openapi/integration-<ver>.json` (committed) | Official OpenAPI spec snapshot consumed by downstream OpenAPI stages (#121) | `10.1.78` |

**Why they diverge**: the Official API (`integration.json`) first appeared in controller 10.1.78. When the internal version pin is below that threshold, `generate()` fetches the Official spec from the **latest** release instead of the internal version. This keeps the committed snapshot current without rewriting all internal resources.

**Reproducing a specific snapshot**: run with `--official-spec-version=<ver>` to pin the Official spec to an exact version regardless of the internal pin:
```sh
go run ./codegen/ -version-base-dir=./codegen/ -output-dir=./unifi --official-spec-version=10.1.78 9.5.21
# → internal resources from 9.5.21 + Official spec from 10.1.78
```
This is how the committed `integration-10.1.78.json` was produced while `.unifi-version` remains `9.5.21`.

**Auto-select logic** (`resolveOfficialSpecVersion` in `version.go`):
- explicit `--official-spec-version` → use that version
- internal >= 10.1.78 → reuse internal version (spec is present in that package)
- internal < 10.1.78 → resolve `latest` (determinism rests on an explicit pin for reproducible CI snapshots)

## Workflows

- **Bump controller version**: update the version arg in `unifi/codegen.go` and `.unifi-version`, run `go generate unifi/codegen.go`, then test + commit all
  generated changes.
- **Override a generated field**: edit `customizations.yml` under the resource (`fieldName`, `fieldType`, `omitEmpty`, `customUnmarshalType`, `jsonPath`,
  `ifFieldType`), then regenerate. New unmarshaler types go in `../unifi/json.go`.
- **Add query params to a resource's URLs**: use the `queryParams` map under the resource in `customizations.yml`
  (e.g. `queryParams: { includeSystemFeatures: "true" }`), NOT a `?foo=bar` suffix on `resourcePath`. The templates render the query string AFTER the `/%s`
  id segment on get/update/delete URLs (and after the bare path on list/create), so the id never lands behind the query string. A raw `?` in `resourcePath`
  is a generation footgun (`described-features?q=1/%s` is never a valid URL) and is rejected under `UNIFI_CODEGEN_STRICT` / warned otherwise. See ARCH-19.
- **Fix bad generated output**: NEVER edit the `.generated.go`. Fix it at the source — `customizations.yml`, the version JSON, or the `*.tmpl` template — and
  regenerate. For behavior, add a hand-written wrapper in `../unifi/<resource>.go`.

## Generated-code conventions

- **`ErrNotFound` is ONLY for get/list-single, never create/update.** The v1 (`api.go.tmpl`) and v2 (`apiv2.go.tmpl`) templates return `ErrNotFound` solely on
  the single-resource GET path (data array length != 1 / empty struct id). A create or update that comes back with an unexpected response shape returns a
  descriptive `fmt.Errorf("unexpected response: expected 1 <Resource>, got %d", ...)` instead — returning a "not found" sentinel from a successful write is
  semantically wrong and misleads callers doing `errors.Is(err, ErrNotFound)`. See ARCH-13. (Hand-written wrappers like `CreateUser`, which post to a nested
  `group/user` endpoint, may still surface `ErrNotFound` for their own inner-lookup semantics — that is wrapper business logic, not the template contract.)

## CI

`test-codegen` (ci.yaml) runs `go generate unifi/codegen.go`; the daily
`generate.yaml` regenerates for the latest controller version and opens a PR.
