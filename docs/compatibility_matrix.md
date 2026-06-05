# Compatibility Matrix

This table maps `go-unifi` library releases to the range of UniFi Network Controller versions they are known to work with.

| `go-unifi` version  | Min UniFi Controller | Latest UniFi Controller |
|---------------------|----------------------|-------------------------|
| `1.10.0`            | `5.12.35`            | `9.4.19`                |
| `v1.9.0` – `v1.9.1` | `5.12.35`            | `9.3.45`                |
| `v0.0.1` – `v1.8.1` | `5.12.35`            | `9.0.114`               |

> [!NOTE]
> Only the **min** and **latest** versions listed above are explicitly verified. Versions in
> between (and newer versions released after the latest tested one) are very likely supported as
> well, but this is **not checked**. If you hit an issue on a specific controller version, please
> [open an issue](https://github.com/filipowm/go-unifi/issues).

The library is updated daily to track the latest UniFi Controller releases, so the "Latest" value moves forward over time. 
The controller version a given build targets is recorded in [`.unifi-version`](../.unifi-version).

## Compatibility Changelog

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