# Changelog

All notable changes to this project are documented in this file. Entries are derived from
[GitHub Releases](https://github.com/filipowm/go-unifi/releases) notes, generated from conventional commits;
internal and maintenance changes are not listed.

## [v2.0.1](https://github.com/filipowm/go-unifi/releases/tag/v2.0.1) (2026-06-27)

### 🔧 Bug Fixes

* fix(network): default absent `enabled` to true so import doesn't disable networks (#177)

## [v2.0.0](https://github.com/filipowm/go-unifi/releases/tag/v2.0.0) (2026-06-13)

### 🚨 Breaking Changes

* feat!: Wave 2 P2 hardening — codegen robustness, ctx-aware interface, centralized soft-error handling
* feat(codegen)!: generate drift-proof settings registry, expose SetSetting, drop DpiApp/DpiGroup (ARCH-08)
* feat(unifi)!: secure-by-default TLS, fine-grained concurrency, offline apiStyle seam (ARCH-06, ARCH-04, TEST-09)

### ✨ New Features

* feat(client): reverse VerifySSL to SkipVerifySSL
* feat(codegen): Official-API OpenAPI models frontend (#121)
* feat(codegen): add Official OpenAPI spec source + committed integration.json snapshot (#121)
* feat(codegen): emit Official API surface and fold into go generate (#121)
* feat(codegen): track Internal and Official API versions separately (#136)
* feat(codegen/official): emit go-playground validate tags from OpenAPI constraints (#163)
* feat(codegen/official): replace oapi-codegen placeholder godoc on model types (#154)
* feat(logging): decouple Logger from Client, make Logger configurable, replace logrus with slog (#167)
* feat(official): add Official-API runtime seam + info/sites vertical (#119)
* feat(official): bounded ListPage + lazy ListAll iterator with explicit filter (#143)
* feat(official): fluent, per-resource-group API surface (#134)
* feat(official): type siteId and all UUID path params as uuid.UUID; ResolveID returns uuid.UUID

### 🔧 Bug Fixes

* fix(auth): drop spurious InternalClientMock from generated mock (#125)
* fix(ci): quote unquoted YAML string values to satisfy yamllint quoted-strings rule
* fix(client): derive User-Agent from module version via debug.ReadBuildInfo, fall back to go-unifi/2
* fix(client): emit Warn when HttpRoundTripperProvider bypasses TLS config management
* fix(client): lower multipart upload size cap to 10 MiB
* fix(client): reject http:// URLs — API key would be sent over plaintext; only https:// accepted
* fix(client): return nil client on construction errors — prevents nil-panic on apiPaths dereference
* fix(client): warn when Timeout is unset — requests can hang indefinitely with a hostile controller
* fix(client): warn when user interceptor is silently dropped due to same-concrete-type dedup
* fix(codegen): address PR #131 review comments (#121)
* fix(codegen): cap internal-API generation at 9.5.21 (classic EOL); fail loud on newer versions (#129)
* fix(codegen): decouple official spec version, test skip path, assert snapshot in TestGenerateLatest
* fix(codegen): dedup intra-family enum twins + harden Official models frontend (#121)
* fix(codegen): emit compiling Go for numeric optional query params
* fix(codegen): make docFor shape-aware so list methods show auto-pagination in godoc (#121)
* fix(codegen): remove stray brace breaking codegen/official build (#121)
* fix(codegen): repin Official spec to 10.1.78 + address review findings
* fix(codegen): url-escape path args in official wrappers (#121)
* fix(codegen): write .unifi-version relative to output dir, not cwd
* fix(codegen/official): emit optional query params (force) on Delete methods via *DeleteOptions
* fix(codegen/official): pluralise resource qualifier in List* method names (ListRulesAll, etc.)
* fix(json): handle JSON null in emptyStringInt.UnmarshalJSON — decode to 0 like empty string
* fix(official): add ErrSiteNotFound sentinel + cover transport/gate/pagination edge cases (#119)
* fix(official): rename SiteOverview.Id -> ID; cover probe-cache and 404-gate paths
* fix(requests): cap multipart upload buffer at 512 MiB to prevent OOM on large uploads
* fix(unifi): decode JSON null as empty in numberOrString (ARCH-07)
* fix(unifi): make booleanishString decoding permissive (ARCH-02)
* fix(unifi): prevent Version() self-deadlock under UseLocking (ARCH-01)
* fix(unifi): register missing mdns/roaming_assistant/traffic_flow setting factories (ARCH-03)
* fix(unifi): robust error handling — Unwrap, 404->ErrNotFound, preserve status on empty bodies (ARCH-22, ARCH-05)
* fix(validation): wire CustomValidators through ClientConfig so exported API is actually usable
* fix: escape % in codegen QuerySuffix and surface root cause in empty ValidationError

## [v1.11.1](https://github.com/filipowm/go-unifi/releases/tag/v1.11.1) (2026-06-11)

### ✨ New Features

* feat: Add GetTrafficFlows to supported API calls (#116)

### 🔧 Bug Fixes

* fix(device): make QOSProfile a pointer so omitempty drops empty qos_profile (#149)

## [v1.11.0](https://github.com/filipowm/go-unifi/releases/tag/v1.11.0) (2026-06-05)

### ✨ New Features

* feat: update to the controller version to 9.5.21

## [v1.10.0](https://github.com/filipowm/go-unifi/releases/tag/v1.10.0) (2026-06-05)

### ✨ New Features

* feat(client): add support for excluding client CRUD actions in resource generation
* feat(content-filtering): add ContentFiltering resource with CRUD operations
* feat: update to the controller version to 9.4.19

## [v1.9.1](https://github.com/filipowm/go-unifi/releases/tag/v1.9.1) (2026-06-04)

### 🔧 Bug Fixes

* fix: omitEmpty for FirewallRule/WLAN/PortProfile reference fields (#111)

## [v1.9.0](https://github.com/filipowm/go-unifi/releases/tag/v1.9.0) (2026-06-04)

### ✨ New Features

* feat(firewall): add network_ids support to FirewallZonePolicyDestination
* feat: add ReorderFirewallPolicies to client
* feat: update to the controller version 9.2.87
* feat: update to the controller version 9.3.45

### 🔧 Bug Fixes

* fix(test): adjust error message assertion for invalid version required after bumping go-version
* fix: always serialize FirewallGroup group_members, even when empty
* fix: properly parse app-ids

### Other

* #84 Support Validation on Array Types (#88)
* Fix issues 76, 77, and IPSec network lifetime

## [v1.8.1](https://github.com/filipowm/go-unifi/releases/tag/v1.8.1) (2025-04-02)

### 🔧 Bug Fixes

* fix: updated FirewallZonePolicy.json to handle port list and ranges (#68)

## [v1.8.0](https://github.com/filipowm/go-unifi/releases/tag/v1.8.0) (2025-03-21)

### ✨ New Features

* feat: support auto allow return traffic for firewall zone policy with create_allow_respond (#60)

### 🔧 Bug Fixes

* fix: do not omit empty portal_customized_bg_image_filename and portal_customized_logo_filename (#59)

## [v1.7.1](https://github.com/filipowm/go-unifi/releases/tag/v1.7.1) (2025-03-20)

### 🔧 Bug Fixes

* fix: perform client-side filtering on GET firewall zone, because API for getting single zone by ID does not exist (#58)

## [v1.7.0](https://github.com/filipowm/go-unifi/releases/tag/v1.7.0) (2025-03-20)

### ✨ New Features

* feat: support list of MAC addresses for Firewall Zone Policy (#56)

### 🔧 Bug Fixes

* fix: remove match_mac from firewall zone destination which is not supported (#57)

## [v1.6.2](https://github.com/filipowm/go-unifi/releases/tag/v1.6.2) (2025-03-17)

### 🔧 Bug Fixes

* fix: adjust firewall zone policy resource date attributes (#54)

## [v1.6.1](https://github.com/filipowm/go-unifi/releases/tag/v1.6.1) (2025-03-17)

### 🔧 Bug Fixes

* fix: adjust guest access settings (#53)

## [v1.6.0](https://github.com/filipowm/go-unifi/releases/tag/v1.6.0) (2025-03-16)

### ✨ New Features

* feat: support Remember Me for prolonging session validity on user/pass authentication (#52)

## [v1.5.4](https://github.com/filipowm/go-unifi/releases/tag/v1.5.4) (2025-03-16)

### 🔧 Bug Fixes

* fix: use omitEmpty only on hotspot for SettingIps (#51)

## [v1.5.3](https://github.com/filipowm/go-unifi/releases/tag/v1.5.3) (2025-03-14)

### 🔧 Bug Fixes

* fix: allow empty fields in SettingMgmt (#50)

## [v1.5.2](https://github.com/filipowm/go-unifi/releases/tag/v1.5.2) (2025-03-11)

### 🔧 Bug Fixes

* fix: revert allowed empty fields for NTP servers

## [v1.5.1](https://github.com/filipowm/go-unifi/releases/tag/v1.5.1) (2025-03-05)

### 🔧 Bug Fixes

* fix: allow more empty fields on rsyslogd, NTP, IPS and USG settings (#45)

## [v1.5.0](https://github.com/filipowm/go-unifi/releases/tag/v1.5.0) (2025-03-03)

### ✨ New Features

* feat: add support for uploading Hotspot Captive Portal files (like background image, logo) (#42)
* feat: support checking supported and enabled controller features (#41)

## [v1.4.1](https://github.com/filipowm/go-unifi/releases/tag/v1.4.1) (2025-03-02)

### 🔧 Bug Fixes

* fix: add missing ip_group_id to firewall zone policy to support firewall groups of address-group type (ipv4) (#39)

## [v1.4.0](https://github.com/filipowm/go-unifi/releases/tag/v1.4.0) (2025-02-23)

### ✨ New Features

* feat: add logging and support for custom logger (#36)

### 🔧 Bug Fixes

* fix: explicitly set Setting key when updating a settings (#37)

## [v1.3.1](https://github.com/filipowm/go-unifi/releases/tag/v1.3.1) (2025-02-21)

### 🔧 Bug Fixes

* fix: add missing field mapping for settings (#35)
* fix: passing setting response body as pointer to Post method (#34)

## [v1.3.0](https://github.com/filipowm/go-unifi/releases/tag/v1.3.0) (2025-02-20)

### ✨ New Features

* feat: add Version method to client to provide system version information (#32)
* feat: support Zone-Based Firewalls (#33)

## [v1.2.0](https://github.com/filipowm/go-unifi/releases/tag/v1.2.0) (2025-02-19)

### ✨ New Features

* feat: allow creating own http.RoundTripper for http.Client with `HttpRoundTripperProvider` when customizing pre-configured http.Transport with `HttpTransportCustomizer` is not sufficient  (#31)

## [v1.1.0](https://github.com/filipowm/go-unifi/releases/tag/v1.1.0) (2025-02-19)

### ✨ New Features

* feat: rename HttpCustomizer to HttpTransportCustomizer and make it return http.Transport that is later used (#30)

## [v1.0.0](https://github.com/filipowm/go-unifi/releases/tag/v1.0.0) (2025-02-18)

### ✨ New Features

* feat(experimental): add support for reading and updating all settings (#25)
* feat(experimental): add support for reading and updating all settings (#26)
* feat: add API v2 support by adding APGroup and DNSRecord resource handling with generated code (#23)
* feat: add client customization option (#20)
* feat: add code generation for Unifi client interface (#11)
* feat: expose all available actions on all resources through Client (#27)
* feat: generate fields validation and use it when sending requests to API (#7)
* feat: make error handling more verbose and collect more error information from API errors, support API V2 error format (#8)
* feat: remove deprecated SettingProviderCapabilities (#24)
* feat: simplified generated resources code customizations with yaml file config (#17)
* feat: use Client interface instead of client struct when interacting with UniFi SDK (#21)
* feat: use sysinfo API for getting system information with fallback to old API (#10)

## [v0.0.4](https://github.com/filipowm/go-unifi/releases/tag/v0.0.4) (2025-02-21)

### Other

* downgrade go version

## [v0.0.2](https://github.com/filipowm/go-unifi/releases/tag/v0.0.2) (2025-02-09)

### ✨ New Features

* feat: add validation of ClientConfig fields for improved data integrity (#5)
* feat: new, more customizable client supporting API Key and user/password authentication

### 🔧 Bug Fixes

* fix: renamed generator template to use ErrNotFound instead of NotFoundError

## [v0.0.1](https://github.com/filipowm/go-unifi/releases/tag/v0.0.1) (2025-02-07)

_Initial release, forked from [paultyng/go-unifi](https://github.com/paultyng/go-unifi)._
