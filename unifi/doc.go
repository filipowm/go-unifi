// Package unifi provides a Go client for the UniFi Network Controller API.
//
// # Authentication
//
// Every client is created with [NewClient] using an API key (requires UniFi Network 9.0.108+
// on new-style/UniFi-OS controllers). Old-style (classic) controllers are unsupported and
// return [ErrOldStyleUnsupported] at construction time.
//
//	c, err := unifi.NewClient(&unifi.ClientConfig{
//	    URL:    "https://unifi.example.com",
//	    APIKey: "your-api-key",
//	})
//
// # Internal vs Official API
//
// The client exposes two surfaces:
//
//   - Internal — the legacy Network API (all generated resource methods, e.g. GetNetwork,
//     ListUser). Reachable directly on the client or via [Client.Internal].
//   - Official — the official UniFi OpenAPI (integration/v1), reached via [Client.Official].
//     Requires controller ≥ 10.1.78 with API-key auth. Operations return
//     [ErrOfficialAPIUnavailable] on unsupported controllers.
//
// In 2.0.0 the Internal surface remains the default: existing code calling resource methods
// directly on the client is unaffected. 3.0.0 is expected to flip the default to Official.
//
// # Concurrency
//
// A *client (and anything obtained from it) is safe for concurrent use by multiple goroutines.
package unifi
