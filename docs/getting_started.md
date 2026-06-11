# Getting Started with UniFi Go SDK

This guide will help you get started with the UniFi Go SDK client. It covers prerequisites, installation, and basic client initialization.
I highly recommend to use the latest version of UniFi Go SDK, as well as update your UniFi Controller to the latest version to ensure compatibility.

## Prerequisites

- Go 1.26 or later
- UniFi Controller 9.0.114 or newer (new-style/UniFi-OS only; old-style controllers are unsupported)

## Installation

Install the UniFi Go SDK by running:

```bash
go get github.com/filipowm/go-unifi/v2
```

If you need to regenerate the client code from the API specifications, run:

```bash
go generate unifi/codegen.go
```

## Initialization

The client uses API Key authentication exclusively. Username/password authentication was removed in 2.0.0.

**IMPORTANT:** API Key authentication is available only since UniFi Controller version 9.0.114. Old-style (classic) controllers are unsupported.

### Obtaining an API Key

1. Open your Site in UniFi Site Manager
2. Click on Control Plane -> Admins & Users.
3. Select your Admin user.
4. Click Create API Key.
5. Add a name for your API Key.
6. Copy the key and store it securely, as it will only be displayed once.
7. Click Done to ensure the key is hashed and securely stored.

### API Key Authentication

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key",
})
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

### Deferring the startup check (SkipSystemInfo)

By default, `NewClient` performs an eager `GetSystemInformation()` round-trip to verify connectivity and
credentials at construction time. If you need fully-offline client construction (e.g. in tests or when
you know the controller is reachable at call time but not at startup), set `SkipSystemInfo: true`:

```go
c, err := unifi.NewClient(&unifi.ClientConfig{
    URL:            "https://unifi.localdomain",
    APIKey:         "your-api-key",
    SkipSystemInfo: true, // skip eager sysinfo check; errors surface on first API call
})
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

With `SkipSystemInfo: true` a bad API key or unreachable controller will NOT cause `NewClient` to fail —
the error surfaces on the first actual API call instead.

## Generating Client Code

The UniFi Go SDK uses code generation to provide complete API coverage. To regenerate the client based on the latest specifications, run:

```bash
go generate unifi/codegen.go
```

This will update the generated models and REST methods according to the current UniFi Controller API specifications.


## Usage

Once instantiated, the client provides direct access to the generated API methods.
If you use a default site and didn't create any new ones, you can use the `default` site ID.

**Example:**

```go
networks, err := c.ListNetwork(ctx, "default")
if err != nil {
    log.Fatalf("Error listing networks: %v", err)
}

for _, network := range networks {
    fmt.Printf("Network: %s\n", network.Name)
}
```

## Checking if features are supported and enabled

The UniFi Go SDK provides a way to check if a feature is supported and enabled/disabled on the UniFi Controller. 
This can be useful when you want to check if a feature is available before using it. Passed feature names are case-insensitive.

**Example:**

```go
enabled, err := c.IsFeatureEnabled(ctx, "default", "feature-name")
if err != nil {
    log.Fatalf("Error checking feature: %v", err)
}
if enabled {
    // Feature is enabled
} else {
    // Feature is disabled
}
```

Library comes with a set of predefined feature names, which can be found in `github.com/filipowm/go-unifi/v2/unifi/features` module. You can also use custom feature names.

For example, you can check if the `features.ZoneBasedFirewallMigration` is available on the controller (no `unifi.ErrNotFound` raised) and enabled:
```go
f, err := c.GetFeature(ctx, "default", features.ZoneBasedFirewallMigration)
if err != nil {
    if errors.Is(err, unifi.ErrNotFound) {
        log.Printf("Feature %s unavailable (not found)", features.ZoneBasedFirewallMigration)
    } else {
        log.Fatalf("Error getting feature: %v", err)
    }
    return false
}
return f.FeatureExists // `FeatureExists` is a boolean indicating if the feature is enabled
```
