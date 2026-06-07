# codegen — UniFi API code generation

Generates `../unifi/*.generated.go` from the controller's API definitions. Run via `go generate unifi/codegen.go`.

**Never edit `*.generated.go`.** Fix the source — `customizations.yml`, the version JSON, or a `*.tmpl` — and regenerate. For behavior, add a sibling `../unifi/<resource>.go`.

## Two surfaces, one `go generate`

`generate()` (`main.go`) runs two passes; both read **committed snapshots offline** (no network in CI):

1. **Internal** (`download.go`→`resources.go`→`customize.go`→`generator.go`) — controller field JSONs → `<resource>.generated.go` + `version.generated.go` + `.unifi-version`. Steps: JSON→Resource, infer Go types from validation regexes, snake→Camel (`fieldReps`), apply `customizations.yml`, render `api.go.tmpl`/`apiv2.go.tmpl`.
2. **Official** (`codegen/official/`, a **separate Go module** shelled out via `os/exec` from `official_pass.go`) — committed OpenAPI snapshot → the whole `unifi/official/` surface. The separate module keeps `oapi-codegen`/`kin-openapi` out of the root `go.mod`.

**Client interface split** (`client.go.tmpl`): `InternalClient` (resource CRUD) + `Client` (transport/lifecycle + hand-written `Internal()`/`Official()`). After changing it, regenerate `client.generated.go` **and** `client_mock.generated.go` (offline moq — see `unifi/mock.go`).

## Frozen legacy snapshot (`codegen/v9.5.21/`)

Committed field-JSON snapshot (+ `.extract-complete` sentinel) so the Internal pass reads it directly — legacy fields are **frozen at 9.5.21** for 2.0.0. `unifi/codegen.go` pins `go:generate` to `9.5.21`, making daily CI a deterministic offline no-op. `.gitignore` keeps `/codegen/v*.*.*/` ignored but unignores `!/codegen/v9.5.21/`.

Any other version (`make generate-resources VERSION=<x>`) targets a different `codegen/v<ver>/` dir and **downloads** — only do this to refresh the snapshot:

1. Remove the old snapshot dir and its gitignore exception.
2. `make generate-resources VERSION=<new-ver>` (downloads + extracts).
3. Add `!/codegen/v<new-ver>/` to `.gitignore`; commit the tree.
4. Update the `go:generate` arg in `unifi/codegen.go` and `.unifi-version`.
5. Regenerate; verify the golden diff is empty or only the intended type changes.

## Two-version model

Internal resource version and Official-spec version are intentionally decoupled:

| Pin | Controls |
|---|---|
| `.unifi-version` / `go generate` arg | internal `.deb` → resources (`9.5.21`) |
| `codegen/openapi/integration-<ver>.json` (committed) | Official OpenAPI snapshot (`10.1.78`) |

The Official API first shipped in 10.1.78. Resolution (`resolveOfficialSpecVersion`): `--official-spec-version` flag → internal version if ≥ 10.1.78 → else `latest`. `codegen/v2/` = hand-maintained V2 API defs (`apiv2.go.tmpl`).

## Official surface internals

oapi-codegen's allOf+discriminator path drops variant structs, so `transform.go`/`naming.go` rewrite the spec into a oneOf union form — deterministic, fail-loud: downconvert 3.1→3.0.3, synthesize a `oneOf` per discriminator, diamond-fix (variant extending 2+ parents), enum dedup, collision-rename. Tri-shape classifier (`resources.go`+`surface.go`) maps ops by `operationId`+method+params: `List*`→`[]…Overview` (auto-paginates), `Get*`→`*…Details`, `Create/Update/Patch*`→`…CreateOrUpdate`. Guarded by `unifi/official/models_roundtrip_test.go`, `TestSurfaceMatchesCommitted`, `TestSurfaceDeterministic`.

## Download trust (ARCH-15/16)

`download.go`/`version.go` is the only remote-ingest point (used only when refreshing snapshots). Guards: bounded timeouts, HTTPS + Ubiquiti-host pinning (`validateDownloadURL`), atomic extraction (`.tmp-*` dir + `.extract-complete`), size caps. No `.deb` checksum (the firmware API exposes none) — trust rests on HTTPS + host pinning + field allowlist + size caps.

## Conventions

- **Override a field** — `customizations.yml` (`fieldType`, `omitEmpty`, `customUnmarshalType`, `jsonPath`, `ifFieldType`); new unmarshalers go in `../unifi/json.go`.
- **Query params** — use the `queryParams` map in `customizations.yml`, NOT a `?…` suffix on `resourcePath` (rejected under `UNIFI_CODEGEN_STRICT`). See ARCH-19.
- **`ErrNotFound`** — templates return it only on the single-resource GET path, never on create/update. See ARCH-13.

## CI

`test-codegen` (ci.yaml) runs `go generate unifi/codegen.go`; the daily `generate.yaml` regenerates `latest` and opens a PR.
