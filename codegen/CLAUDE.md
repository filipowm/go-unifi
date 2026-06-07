# codegen — UniFi API code generation

Generates `../unifi/*.generated.go` from the controller's own API definitions. Run via `go generate unifi/codegen.go` (→ `go run ../codegen/ -version-base-dir=../codegen/ <version>`).

## Pipeline

1. `download.go` — fetch controller `.deb` from dl.ui.com; extract `data.tar.xz`→`ace.jar`→ field JSONs (`api/fields/*.json`; `Setting.json` split per-setting). Also fetches `unifi-uos_sysvinit.deb`, extracts the Official OpenAPI 3.1 spec (`integration.json`) and commits a pinned snapshot `codegen/openapi/integration-<ver>.json` (controllers < 10.1.78 lack it → skipped with a warning).
2. `resources.go` — JSON → Resource; infer Go types from validation regexes; snake_case→CamelCase (`fieldReps`).
3. `customize.go` — apply `customizations.yml` overrides.
4. `generator.go` — render `api.go.tmpl`/`apiv2.go.tmpl` → `<resource>.generated.go`; write `version.generated.go` + `.unifi-version`.

**Client interface split** (`client.go.tmpl` + `clients.go`): renders `InternalClient` (resource CRUD) and `Client` (embeds it + transport/lifecycle fns + hand-written `Internal()`/`Official()`), split on `ClientFunction.ResourceName()`. After changing it, regenerate `client.generated.go` **and** `client_mock.generated.go` (offline moq — see `unifi/mock.go`).

## Official-API frontend — OpenAPI surface (`codegen/official/`)

Separate Go module (`…/codegen/official`) hosting the OpenAPI toolchain (`oapi-codegen/v2` + `kin-openapi`), isolated so those deps stay out of root `go.mod` (root gains only `oapi-codegen/runtime`, imported by the generated models). Reads the **committed** snapshot **offline** → writes the whole Official surface into `unifi/official/` (all `DO NOT EDIT`): `models.generated.go`, `wrappers.generated.go`, `client.generated.go` (the `Client` interface), `client_mock.generated.go` (a func-field mock).

```sh
cd codegen/official && go run .   # -openapi-dir / -out-dir override the defaults
```

**Folded into `go generate` (second pass).** `generate()` (root `main.go`) shells out to this module via `os/exec` after the Internal pass (`official_pass.go`), so one `go generate unifi/codegen.go` emits both surfaces. We shell out rather than import to keep the oapi-codegen graph out of the root module.

**Tri-shape wrappers** (`resources.go` + `surface.go`). Each operation is classified from its `operationId` + HTTP method + parameters (NOT path regexes): `List*`→`[]…Overview` (auto-paginating the offset/limit envelope, max 200), `Get*`→`*…Details`, `Create/Update/Patch*(…CreateOrUpdate)`, plus ordering GET/PUT, action POSTs, references, `statistics/latest`, and bulk `deleteVouchers` (its `filter` is REQUIRED — guarded against empty). Wrapper types resolve through the same `finalName` map as the models, so they match emitted type names exactly. `getInfo`/`getSiteOverviewPage` are skipped — hand-written in `info.go`/`sites.go`. The `PATCH` verb is carried by `official.Doer.Patch` (and `unifi.client.Patch`, exposed on the public `Client` via `customizations.yml`).

**The oneOf transform** (`transform.go`, `naming.go`) — oapi-codegen's allOf+discriminator path silently drops every variant struct, so the spec is rewritten into its oneOf path (per-variant union types). Deterministic, fail-loud:

- **Downconvert 3.1→3.0.3** — bump the `openapi` string only, after asserting zero 3.1-only constructs across the whole doc (lossless).
- **oneOf synthesis** — each discriminator parent gets a `oneOf` over all variants, mined from allOf back-refs AND `discriminator.mapping`; graph is cycle-checked.
- **Diamond fix** — a variant extending 2+ parents keeps the first parent ref and inlines the rest's fields (oapi-codegen can't merge two discriminators).
- **Naming + collisions** — strip `Integration` prefix / `Dto` suffix, `Create or update X`→`XCreateOrUpdate`; collisions recomputed on the post-transform set must be unique (one override: `IP Address selector`→`SpecificIPAddressSelector`).
- **Enum dedup** — `sharedEnums` hoists a value-set shared across a tri-shape family into one type (`enumRef` preserves sibling metadata via a single-member `allOf`).
- **mapping assert** — every `discriminator.mapping` key must be the UPPER_SNAKE wire value (so `ValueByDiscriminator()` decodes real payloads).

**Generated shape** — each parent → union struct (discriminator field + unexported `union json.RawMessage`, `AsXxx/FromXxx/MergeXxx`, `Discriminator()`, `ValueByDiscriminator()`, `Marshal/UnmarshalJSON`). Variants with fields are full structs; **empty variants are type aliases** of the parent (branch on `Discriminator()`). Round-trip survival is tested in `unifi/official/models_roundtrip_test.go`; `TestModelsMatchCommitted` byte-guards every other family. `unifi/json.go` leaf unmarshalers are unaffected.

**Hand-written collisions** — `Site overview` is excluded (refs resolve to hand-written `SiteOverview`); `Application info` → `type ApplicationInfo = Info`.

**Determinism gate** — `TestSurfaceMatchesCommitted` byte-guards every generated surface file against its committed copy; `TestSurfaceDeterministic` proves re-generation is byte-identical.

## Download trust model (ARCH-15 / ARCH-16)

`download.go`/`version.go` is the only remote-ingest point. Guards: bounded `context.Context` timeouts; HTTPS + Ubiquiti-host pinning (`validateDownloadURL`; loopback exempt for the httptest seam); atomic extraction (sibling `*.tmp-*` dir + `.extract-complete` sentinel; the single-file spec snapshot uses temp-file rename + `maxOpenAPISpecSize` cap). **Not** verified: no `.deb` checksum (the firmware API exposes none) — provenance rests on HTTPS + host pinning + the field allowlist + size caps.

## Two-version model (internal vs Official spec)

The internal resource-gen version and the Official-spec version are intentionally decoupled:

| Pin | Controls | Example |
|---|---|---|
| `.unifi-version` / `go generate` arg | internal `.deb` → generated resources | `9.5.21` |
| `codegen/openapi/integration-<ver>.json` (committed) | Official OpenAPI snapshot | `10.1.78` |

The Official API first shipped in 10.1.78; below that, `generate()` fetches the spec from `latest`. Pin exactly with `--official-spec-version=<ver>` (how `integration-10.1.78.json` was produced while `.unifi-version` stays `9.5.21`). Resolution (`resolveOfficialSpecVersion`): explicit flag → internal if ≥ 10.1.78 → else `latest`.

`codegen/v<X.Y.Z>/` is a `.gitignore`d download cache (`v2/` = V2 API, `apiv2.go.tmpl`); the `openapi/` snapshot is committed.

## Workflows

- **Bump version** — edit the arg in `unifi/codegen.go` + `.unifi-version`, `go generate unifi/codegen.go`, test, commit all generated changes.
- **Override a field** — edit `customizations.yml` (`fieldType`, `omitEmpty`, `customUnmarshalType`, `jsonPath`, `ifFieldType`), regenerate; new unmarshalers go in `../unifi/json.go`.
- **Add query params** — use the `queryParams` map under the resource in `customizations.yml`, NOT a `?…` suffix on `resourcePath` (rejected under `UNIFI_CODEGEN_STRICT`). See ARCH-19.
- **Fix bad output** — NEVER edit `*.generated.go`; fix the source (`customizations.yml`, version JSON, `*.tmpl`) and regenerate. For behavior, add a sibling `../unifi/<resource>.go`.

## Conventions

- **`ErrNotFound` only on get/list-single, never create/update** — templates return it solely on the single-resource GET path; a create/update with an unexpected shape returns a descriptive `fmt.Errorf` instead. See ARCH-13. (Hand-written wrappers may still surface `ErrNotFound` for their own lookup semantics.)

## CI

`test-codegen` (ci.yaml) runs `go generate unifi/codegen.go`; the daily `generate.yaml` regenerates for `latest` and opens a PR.
