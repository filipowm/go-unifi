# Compatibility Matrix

This table maps `go-unifi` library releases to the range of UniFi Network Controller versions they are known to work with.

| `go-unifi` version  | Min UniFi Controller | Internal API version | Official API version |
|---------------------|----------------------|----------------------|----------------------|
| `2.0.0`             | `9.0.114`            | `9.5.21` (frozen)    | `10.1.78` (tracks spec) |
| `1.11.0`            | `5.12.35`            | `9.5.21`             | —                    |
| `1.10.0`            | `5.12.35`            | `9.4.19`             | —                    |
| `v1.9.0` – `v1.9.1` | `5.12.35`            | `9.3.45`             | —                    |
| `v0.0.1` – `v1.8.1` | `5.12.35`            | `9.0.114`            | —                    |

> [!NOTE]
> Only the **min** and **latest** versions listed above are explicitly verified. Versions in
> between (and newer versions released after the latest tested one) are very likely supported as
> well, but this is **not checked**. If you hit an issue on a specific controller version, please
> [open an issue](https://github.com/filipowm/go-unifi/issues).

**2.0.0 notes:**
- The **minimum** controller version (9.0.114) is set by API-key authentication — the only supported auth.
  Old-style (classic) controllers are unsupported (`ErrOldStyleUnsupported`).
- The **Internal API** is frozen at `9.5.21` for 2.0.0. The daily CI run is a deterministic no-op for the
  internal half — the resolver in `codegen/version.go` clamps to `9.5.21`, so `.unifi-version` does not
  move forward during the 2.0.0 lifecycle.
- The **Official OpenAPI** surface (`c.Official()`) requires controller ≥ `10.1.78`. The spec version
  (`.unifi-version-official`) tracks the latest committed snapshot and may update when the Official spec changes.

Only the Official API spec may update daily — the `.unifi-version-official` marker tracks the latest committed
OpenAPI snapshot. The Internal (legacy) resources are frozen at `9.5.21` and do not change during the 2.0.0 lifecycle.
Two plain-text version markers are written at the repo root by `go generate`:

- [`.unifi-version`](../.unifi-version) — the Internal (legacy) API controller version
- [`.unifi-version-official`](../.unifi-version-official) — the Official OpenAPI (`integration/v1`) spec version (requires controller ≥ `10.1.78`)

## Compatibility Changelog

### 2.0.0 — Breaking changes

See [docs/2.0.0/breaking_changes.md](2.0.0/breaking_changes.md) for the full list of 10 breaking changes.
Key changes: API-key-only auth, TLS verify-by-default, Go 1.26+, `VerifySSL` → `SkipVerifySSL`, removed
`Login`/`Logout`, `NewBareClient` replaced by `SkipSystemInfo`.

### 1.11.0 / UniFi Controller 9.5.21

**Breaking changes**

`ChannelPlan` — trimmed down to `Date` and `RadioTable` (no replacement identified):
- removed fields `ApBlacklistedChannels`, `ConfSource`, `Coupling`, `Fitness`, `Note`, `Radio`, `Satisfaction`, `SatisfactionTable`, `SiteBlacklistedChannels`
- removed types `ChannelPlanApBlacklistedChannels`, `ChannelPlanCoupling`, `ChannelPlanSatisfactionTable`, `ChannelPlanSiteBlacklistedChannels`
- removed field `ChannelPlanRadioTable.BackupChannel` 

`DeviceRadioTable`:
- removed fields `BackupChannel` and `ChannelOptimizationEnabled`

**Additions**

- `Network`: `WANDHCPv6PDSizeAuto` (auto IPv6 prefix-delegation size); `WANType` accepts `dslite-over-pppoe`.
- `SettingRadioAi`: `HighPriorityDevices` (high-priority device MACs); `Radios` accepts `6e` (Wi-Fi 6E band).

### 1.10.0 / UniFi Controller 9.4.19

**Breaking changes**

`SettingIps` — DNS filtering & ad-blocking moved out into the new `ContentFiltering` resource:
- removed fields `DNSFiltering`, `DNSFilters`, `AdBlockingEnabled`, `AdBlockingConfigurations` (and types `SettingIpsDNSFilters`, `SettingIpsAdBlockingConfigurations`)
- added `ContentFilteringBlockingPageEnabled`

`Device`:
- `Mbb` field renamed to `MbbOverrides` (type `DeviceMbb` → `DeviceMbbOverrides`)
- `DeviceSim.Iccid` removed

`ContentFiltering` (new resource, replaces the removed IPS fields)

**Additions**

- New `ContentFiltering` resource with full CRUD (`List/Get/Create/Update/DeleteContentFiltering`).
- `Device`: `NutServer`, `DeviceCurrentApn.PDpType`, and `DeviceSim` SIM-data fields
  (`DataSoftLimitDisplayUnit`, `DataWarningThreshold`, `ResetDate`, `ResetPolicy`, `UseCustomApn`).
- `Network`: IPv6 DHCP support (`WANDHCPv6Cos`, `WANDHCPv6Options`) and MAP-E `WANType` values.
- `WLANMdnsProxyCustom.ServicesMode` accepts `none`; new dashboard widgets.
