# codegen — UniFi API code generation

Generates `../unifi/*.generated.go` from the controller's API definitions. Run via `go generate unifi/codegen.go`.

**Never edit `*.generated.go`.** Fix the source — `codegen/internal/customizations.yml`, the version JSON, or a `*.tmpl` — and regenerate. For behavior, add a sibling `../unifi/<resource>.go`.

## Package layout

```
codegen/                Root orchestration (package main): version resolution, downloading,
                        Official-API pass handoff, version.generated.go + both marker files.
codegen/shared/         Shared utilities: Logger interface, EnsurePath/FindProjectRoot/FindCodegenDir,
                        CopyWithLimit. Imported by both root and internal.
codegen/internal/       Internal-API generation engine. Exposes internal.Generate() as the single
                        entry point. Holds customizations.yml, api.go.tmpl, apiv2.go.tmpl,
                        client.go.tmpl, common.tmpl, download.go, resources.go, customize.go,
                        generator.go, clients.go.
codegen/official/       Separate go.mod module: Official OpenAPI surface generator (standalone).
```

**Root orchestrates, internal generates.** `main.go`'s `generate()` calls `internal.Generate(structuresDir, v2BaseDir, outDir, customizer, logger)` for the Internal-API pass (resources + client interface only — **NOT version.generated.go**), then shells out to `codegen/official` for the Official-API pass. **Root writes version.generated.go** (both `UnifiVersion` and `OfficialAPIVersion` constants) **plus both markers** (`.unifi-version`, `.unifi-version-official`) via `writeVersionArtifacts`.

## Two surfaces, one `go generate`

`generate()` runs two passes; both read **committed snapshots offline** (no network in CI):

1. **Internal** (`internal/download.go`→`internal/resources.go`→`internal/customize.go`→`internal/generator.go`) — controller field JSONs → `<resource>.generated.go` + `client.generated.go`. Steps: JSON→Resource, infer Go types from validation regexes, snake→Camel (`fieldReps`), apply `internal/customizations.yml`, render `api.go.tmpl`/`apiv2.go.tmpl`.
2. **Official** (`codegen/official/`, a **separate Go module** shelled out via `os/exec` from `official_pass.go`) — committed OpenAPI snapshot → the whole `unifi/official/` surface. The separate module keeps `oapi-codegen`/`kin-openapi` out of the root `go.mod`.

**Client interface split** (`internal/client.go.tmpl` + `internal/clients.go`): renders `InternalClient` (resource CRUD) and `Client` (embeds it + transport/lifecycle fns + hand-written `Internal()`/`Official()`), split on `ClientFunction.ResourceName()`. After changing it, regenerate `client.generated.go` **and** `client_mock.generated.go` (offline moq — see `unifi/mock.go`).

**Customizations file** lives at `codegen/internal/customizations.yml` (embedded in the binary). Edit it there to override field types, add query params, exclude resources, or declare extra client functions. The default embed path is `"customizations.yml"` (matched by `NewCodeCustomizer("")`); pass an explicit path for tests.

**Client interface split** (`client.go.tmpl`): `InternalClient` (resource CRUD) + `Client` (transport/lifecycle + hand-written `Internal()`/`Official()`). After changing it, regenerate `client.generated.go` **and** `client_mock.generated.go` (offline moq — see `unifi/mock.go`).

## Frozen legacy snapshot (`codegen/v9.5.21/`)

Committed field-JSON snapshot (+ `.extract-complete` sentinel) so the Internal pass reads it directly — legacy fields are **frozen at 9.5.21** for 2.0.0. `unifi/codegen.go` pins `go:generate` to `9.5.21`, making daily CI a deterministic offline no-op. `.gitignore` keeps `/codegen/v*.*.*/` ignored but unignores `!/codegen/v9.5.21/`.

Any other version (`make generate-resources VERSION=<x>`) targets a different `codegen/v<ver>/` dir and **downloads** — only do this to refresh the snapshot:

1. Remove the old snapshot dir and its gitignore exception.
2. `make generate-resources VERSION=<new-ver>` (downloads + extracts).
3. Add `!/codegen/v<new-ver>/` to `.gitignore`; commit the tree.
4. Update the `go:generate` arg in `unifi/codegen.go` and `.unifi-version`.
5. Regenerate; verify the golden diff is empty or only the intended type changes.

## Two-pipeline version model

The two code-generation pipelines track **independent** version axes:

| Pin | Controls | Resolution |
|---|---|---|
| `.unifi-version` / `go generate` arg | Internal `.deb` → resources (capped at `9.5.21`) | `resolveInternalVersion` |
| `codegen/openapi/integration-<ver>.json` (committed) | Official OpenAPI snapshot (`10.1.78+`) | `resolveOfficialSpecVersion` |

**Internal pipeline — capped at 9.5.21 (classic EOL, fail loud past it)**

The classic UniFi Network Controller reached end-of-life at 9.5.21. UOS packages (>= 10.x) no longer ship the `api/fields/*.json` definitions that the Internal pipeline reads. `resolveInternalVersion` enforces this:

- `latest` (or any resolved version > 9.5.21): clamp to 9.5.21 — `min(resolved, 9.5.21)`.
- explicit version > 9.5.21: **fail loud** with an actionable error pointing to the Official OpenAPI frontend (issue #121) and the `-official-spec-version` flag.
- explicit version ≤ 9.5.21: resolved unchanged (backward compat — `make generate VERSION=<x>` still works).

The cap lives **only** in `resolveInternalVersion`; the shared provider (`Latest`/`ByVersionMarker`) is never modified.

**Official pipeline — tracks UOS ≥ 10.1.78**

The Official API first shipped in 10.1.78. `resolveOfficialSpecVersion`: `--official-spec-version` flag → internal version if ≥ 10.1.78 → else `latest`. Deliberately unaffected by the internal cap so legitimate UOS versions resolve freely.

`codegen/v2/` = hand-maintained V2 API defs (`apiv2.go.tmpl`).

## Official surface internals

oapi-codegen's allOf+discriminator path drops variant structs, so `transform.go`/`naming.go` rewrite the spec into a oneOf union form — deterministic, fail-loud: downconvert 3.1→3.0.3, synthesize a `oneOf` per discriminator, diamond-fix (variant extending 2+ parents), enum dedup, collision-rename. Tri-shape classifier (`resources.go`+`surface.go`) maps ops by `operationId`+method+params: `List*`→`[]…Overview` (auto-paginates), `Get*`→`*…Details`, `Create/Update/Patch*`→`…CreateOrUpdate`.

**Fluent, per-group surface** (`groups.go`+`surface.go`) — the surface is grouped by OpenAPI tag, not flat: each operation's primary tag selects a group (`operationGroup`); docs-only/zero-op tags auto-skip. The generator emits one `<group>.generated.go` per group (the `<Group>Client` interface, its `*apiClient` accessor + impl, the wrapper impls, and a per-group func-field `<Group>ClientMock`) plus a parent `client.generated.go` (the `Client` interface with one accessor per group). `groupName` PascalCases the tag with `groupOverrides` tidying ambiguous ones; convention is **plural for true resource collections** (`DNSPolicies`, `ACLs`, `TrafficMatchingLists`), **singular for feature areas** (`Firewall`, `Hotspot`, `Supporting`, `Info`) — the go-github/k8s/Stripe idiom; an unlisted new tag auto-yields a new group (caught by the golden diff). `methodName` strips the group's resource word(s) so methods read cleanly under their accessor (`createFirewallPolicy` under `Firewall()`→`CreatePolicy`); `stemOverrides` supplies the singular token set for pluralised groups (`DNSPolicies` strips `["DNS","Policy"]` so `createDnsPolicy`→`Create`); a post-strip collision fails loud (`buildGroups`/`assertNoCollision`). The three hand-written methods are **re-homed onto groups** (`Info().Get`, `Sites().List`, `Sites().ResolveID`): the generator emits their interface/mock/accessor but the body lives in `unifi/official/info.go`/`sites.go` on the `infoClient`/`sitesClient` impls, so a group mixes generated wrappers with preserved hand-written members.

Guarded by `unifi/official/models_roundtrip_test.go`, `TestSurfaceMatchesCommitted` (byte-equal, no orphan files), `TestSurfaceDeterministic`.

## Download trust (ARCH-15/16)

`download.go`/`version.go` is the only remote-ingest point (used only when refreshing snapshots). Guards: bounded timeouts, HTTPS + Ubiquiti-host pinning (`validateDownloadURL`), atomic extraction (`.tmp-*` dir + `.extract-complete`), size caps. No `.deb` checksum (the firmware API exposes none) — trust rests on HTTPS + host pinning + field allowlist + size caps.

## Conventions

- **Override a field** — `customizations.yml` (`fieldType`, `omitEmpty`, `customUnmarshalType`, `jsonPath`, `ifFieldType`); new unmarshalers go in `../unifi/json.go`.
- **Query params** — use the `queryParams` map in `customizations.yml`, NOT a `?…` suffix on `resourcePath` (rejected under `UNIFI_CODEGEN_STRICT`). See ARCH-19.
- **`ErrNotFound`** — templates return it only on the single-resource GET path, never on create/update. See ARCH-13.

## CI

`test-codegen` (ci.yaml) runs `go generate unifi/codegen.go`; the daily `generate.yaml` regenerates `latest` and opens a PR.
