---
paths:
  - "**/*_test.go"
---

# Testing conventions

- Assertions: `testify/assert` + `testify/require` (or `tj/assert` for simple cases). Use `assert.JSONEq` to compare marshaled JSON.
- Table-driven tests using a `map[string]struct{...}` of cases; one `t.Run(name, ...)` per case. Call `t.Parallel()` in both the outer test and each subtest.
- When a test needs a real HTTP round-trip, mock the controller with `net/http/httptest.NewServer` and assert on request path/method inside the handler.
- Test files are `*_test.go` and use the external `unifi_test` package where exercising the public API.
- Run the suite with `go test ./...`; a single test with `go test -run TestName ./unifi`.
