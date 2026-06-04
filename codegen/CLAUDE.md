# codegen — UniFi API code generation

Generates the `*.generated.go` files in `../unifi/` from the UniFi controller's own API field definitions. Triggered by `go generate unifi/codegen.go`, 
which runs: `go run ../codegen/ -version-base-dir=../codegen/ <version>`.

## Pipeline

1. `download.go` — downloads the controller `.deb` from dl.ui.com, extracts `data.tar.xz` → `ace.jar` → JSON field definitions (`api/fields/*.json`).
   `Setting.json` is split into per-setting files.
2. `resources.go` — parses each JSON into a Resource; infers Go types from the field validation regexes; snake_case → CamelCase (acronyms via `fieldReps`).
3. `customize.go` — applies `customizations.yml` field overrides.
4. `generator.go` — renders `api.go.tmpl` / `apiv2.go.tmpl` → `<resource>.generated.go`; writes `version.generated.go` and the repo `.unifi-version` marker.

## Versioning

- `codegen/v<X.Y.Z>/` holds the JSON field defs per controller version (`v2/` = V2 API resources, rendered with `apiv2.go.tmpl`, different endpoints).
- `.unifi-version` (repo root) and the version arg in `unifi/codegen.go` pin the supported version.

## Workflows

- **Bump controller version**: update the version arg in `unifi/codegen.go` and `.unifi-version`, run `go generate unifi/codegen.go`, then test + commit all
  generated changes.
- **Override a generated field**: edit `customizations.yml` under the resource (`fieldName`, `fieldType`, `omitEmpty`, `customUnmarshalType`, `jsonPath`,
  `ifFieldType`), then regenerate. New unmarshaler types go in `../unifi/json.go`.
- **Fix bad generated output**: NEVER edit the `.generated.go`. Fix it at the source — `customizations.yml`, the version JSON, or the `*.tmpl` template — and
  regenerate. For behavior, add a hand-written wrapper in `../unifi/<resource>.go`.

## CI

`test-codegen` (ci.yaml) runs `go generate unifi/codegen.go`; the daily
`generate.yaml` regenerates for the latest controller version and opens a PR.
