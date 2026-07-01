# codegen — UniFi API code generation

Generates `../unifi/*.generated.go`. Run `go generate unifi/codegen.go`.

**Never edit `*.generated.go`.** Fix the source — `codegen/internal/customizations.yml`, a committed snapshot, or a `*.tmpl` — and regenerate. For behavior, add a sibling `../unifi/<resource>.go`.

## Two passes (both offline from committed snapshots)

`generate()` runs two passes; root (`main.go`) orchestrates and writes `version.generated.go` + both markers (`.unifi-version`, `.unifi-version-official`).

- **Internal** (`internal/`, entry point `internal.Generate(...)`) — controller field JSONs → `<resource>.generated.go` + `client.generated.go`. `v2/` defs render via `apiv2.go.tmpl`.
- **Official** (`codegen/official/`, a separate go.mod shelled out via `os/exec`) — committed OpenAPI snapshot → the whole `unifi/official/` surface. Separate module keeps `oapi-codegen`/`kin-openapi` out of the root `go.mod`.

After changing client templates, regenerate `client.generated.go` **and** `client_mock.generated.go` (offline moq — see `unifi/mock.go`).

## Version model — two independent axes

| Pin | Controls | Resolver |
|---|---|---|
| `.unifi-version` / `-legacy-version` arg | Internal resources, **capped at 9.5.21** (classic EOL — fail loud past it) | `resolveInternalVersion` |
| committed `codegen/openapi/integration-<ver>.json` / **positional** `go:generate` arg | Official OpenAPI snapshot (≥ 10.1.78) | `resolveOfficialSpecVersion` |

The positional codegen argument is the **Official spec version**; `-legacy-version` pins the Internal version (`make generate-resources` maps `VERSION`→official, `LEGACY_VERSION`→internal). Multiple `integration-<ver>.json` snapshots may coexist under `codegen/openapi/`: the Go surface is generated from the pinned version (`ResolveSnapshot`), while the docs website renders the newest committed snapshot.

Legacy fields are frozen at `codegen/v9.5.21/` (committed snapshot + `.gitignore` unignore), so daily CI is a deterministic offline no-op. To refresh: remove the old snapshot dir + its gitignore exception, `make generate-resources LEGACY_VERSION=<x>`, re-add `!/codegen/v<x>/`, bump `unifi/codegen.go` + `.unifi-version`, regenerate, verify the golden diff.

## Official surface

Generation details (spec rewrite, per-tag grouping, pagination, validation-tag injection) live next to the code that owns them — read the source, not this file:

- `official/transform.go`, `naming.go` — allOf/discriminator → oneOf rewrite.
- `official/resources.go`, `surface.go`, `groups.go` — per-tag fluent grouping + shape classification.
- `official/validation.go` — go-playground `validate` tags from spec constraints (validated under the parent client's `ValidationMode`; non-breaking by default).
- `unifi/official/pagination.go` — hand-written `ListOptions`/`Page[T]`/drain helpers.

Guarded by `TestSurfaceMatchesCommitted`, `TestSurfaceDeterministic`, `models_roundtrip_test.go`.

## Conventions

- **Override a field** — `customizations.yml` (`fieldType`, `omitEmpty`, `customUnmarshalType`, `jsonPath`, `ifFieldType`); new unmarshalers in `../unifi/json.go`.
- **Query params** — `queryParams` map in `customizations.yml`, not a `?…` suffix on `resourcePath` (rejected under `UNIFI_CODEGEN_STRICT`). See ARCH-19.
- **`ErrNotFound`** — only on single-resource GET, never create/update. See ARCH-13.
- **Snapshot downloads** (`download.go`) — the only remote-ingest point; HTTPS + Ubiquiti-host pinning, atomic extract, size caps. No `.deb` checksum (none published). ARCH-15/16.

## CI

`test-codegen` (ci.yaml) runs `go generate unifi/codegen.go`; daily `generate.yaml` regenerates `latest` and opens a PR.
