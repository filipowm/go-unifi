---
paths:
  - "**/*_test.go"
---

# Testing conventions

- Assertions: `testify/assert` + `testify/require` (or `tj/assert` for simple cases). Use `assert.JSONEq` to compare marshaled JSON.
- Table-driven tests using a `map[string]struct{...}` of cases; one `t.Run(name, ...)` per case. Call `t.Parallel()` in both the outer test and each subtest.
- When a test needs a real HTTP round-trip, mock the controller with `net/http/httptest.NewServer` and assert on request path/method inside the handler.
- Prefer the shared fixtures over hand-rolling setup: route through the `controllerServer` fixture and helpers like `sysinfoRoute` / `clientWith` instead of spinning up a bespoke `httptest.NewServer` + client per test. Reusing them keeps request routing, version pre-warm, and counters consistent across the suite.
- Strive to create and reuse fixtures. When you find yourself duplicating server/client setup, route handlers, or assertion counters across tests, extract a shared fixture or helper rather than copy-pasting — and reach for an existing fixture before writing new boilerplate.
- Test files are `*_test.go` and use the external `unifi_test` package where exercising the public API.
- Run the suite with `go test ./...`; a single test with `go test -run TestName ./unifi`.
