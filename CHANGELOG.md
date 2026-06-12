# Changelog

All notable changes to this project are documented in this file. Entries are derived from
[GitHub Releases](https://github.com/filipowm/go-unifi/releases) notes, generated from conventional commits;
internal and maintenance changes are not listed.

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
