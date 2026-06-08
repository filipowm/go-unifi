# Migrating from `paultyng/go-unifi`

This guide will help you migrate from `paultyng/go-unifi` to `filipowm/go-unifi`. The main differences are in how the client is initialized, while all client methods remain the same, with additional methods covering of the UniFi Controller API.

## Client Initialization Changes

### paultyng/go-unifi Style

In the upstream library, client initialization typically looks like this:

```go
client := unifi.Client{}
client.SetBaseURL("https://unifi.localdomain")

// Optional: Configure TLS
if skipTLSVerify {
    httpClient := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        },
    }
    jar, _ := cookiejar.New(nil)
    httpClient.Jar = jar
    client.SetHTTPClient(httpClient)
}

// Login is required
err := client.Login(ctx, username, password)
if err != nil {
    return nil, err
}
```

### filipowm/go-unifi Style

The new library provides a more structured and configurable approach. In 2.0.0, only API Key
authentication is supported (available from UniFi Controller 9.0.114+):

```go
client, err := unifi.NewClient(&unifi.ClientConfig{
    URL:    "https://unifi.localdomain",
    APIKey: "your-api-key",
})

if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

## Key Differences

1. **Client Creation**:
   - Old: Manual client creation and configuration using setters. Client used is a struct.
   - New: Builder pattern with `NewClient` function and `ClientConfig` struct. Client used is an [interface](../unifi/client.generated.go).

2. **Authentication**:
   - Old: Only username/password authentication with explicit `Login()` call
   - New: API Key authentication only (username/password removed in 2.0.0)
   - New: Login is handled automatically during client creation

3. **HTTP Client Configuration**:
   - Old: Manual HTTP client configuration with `SetHTTPClient` and manual creation of `http.Client` and `cookiejar.Jar`
   - New: Built-in configuration options through `ClientConfig`:
     - `HttpTransportCustomizer` for transport-level customization
     - `HttpRoundTripperProvider` for complete HTTP client control

4. **Removed unifi.APIError**:
   - Old: `unifi.APIError` struct for API errors
   - New: Standard `unifi.ServerError` struct for API errors

5. **Additional Features in filipowm/go-unifi**:
   - Validation modes (Soft, Hard, Disabled)
   - Request/Response interceptors
   - Custom error handling
   - Comprehensive configuration options

## Migration Steps

1. Replace the import from `github.com/paultyng/go-unifi` to `github.com/filipowm/go-unifi/v2`
2. Replace manual client creation with `NewClient` and appropriate `ClientConfig` (API key required)
3. TLS verification is now **on by default** (secure by default). To skip verification (e.g. self-signed
   certs), set `SkipVerifySSL` to `true` (the zero value `false` verifies), and
   disabling verification logs a warning:
   ```go
   client, err := unifi.NewClient(&unifi.ClientConfig{
       URL:           "https://unifi.localdomain",
       APIKey:        "your-api-key",
       SkipVerifySSL: true,
   })
   ```
4. Remove explicit `Login()` calls as they are now handled automatically
5. Replace usage of `unifi.APIError` with `unifi.ServerError`

The rest of your code using the client methods should continue to work as before, as the API methods remain the same.

For details on configuring client, check [client configuration](./configuration.md).
