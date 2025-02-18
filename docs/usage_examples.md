# Usage Examples

This document demonstrates several common usage scenarios for the UniFi Go SDK client.

## Listing Networks

List all available networks in a given site:

```go
networks, err := c.ListNetwork(ctx, "site-name")
if err != nil {
    log.Fatalf("Error listing networks: %v", err)
}

for _, network := range networks {
    fmt.Printf("Network: %s\n", network.Name)
}
```

## Creating a User

Create a new user assigned to the first available network:

```go
// Assume networks have been retrieved as shown above
user, err := c.CreateUser(ctx, "site-name", &unifi.User{
    Name:      "My Network User",
    MAC:       "00:00:00:00:00:00",
    NetworkID: networks[0].ID,
    IP:        "10.0.21.37",
})
if err != nil {
    log.Fatalf("Error creating user: %v", err)
}

fmt.Printf("Created user: %s\n", user.Name)
```

## Updating a Guest Access setting

Update the guest access setting for a network:

```go
setting := &unifi.SettingGuestAccess{
    PortalCustomizedBoxColor: "#ff0000",
    PortalCustomizedSuccessText: "Welcome to the network!",
    PortalCustomized: true,
}

setting, err = c.UpdateSettingGuestAccess(ctx, "site-name", setting)
if err != nil {
    log.Fatalf("Error updating guest access setting: %v", err)
}
// Use the updated setting
```