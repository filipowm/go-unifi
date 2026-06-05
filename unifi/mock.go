package unifi

// ClientMock (in the generated client_mock.generated.go sibling) is a
// moq-generated test double for the public Client interface. Regenerate it after
// any change to the Client interface:
//
//	go generate ./unifi/...   # or: go run github.com/matryer/moq@latest -out client_mock.generated.go . Client
//
// The .generated.go suffix keeps the mock out of the project's coverage report
// (the Makefile excludes *.generated.go), matching the repo's generated-file
// convention.
//
//go:generate go run github.com/matryer/moq@latest -out client_mock.generated.go . Client

// Compile-time assurance that the generated mock satisfies the public Client
// interface. The generated file carries the same assertion, but duplicating it in
// a hand-written sibling means a stale or accidentally-deleted
// client_mock.generated.go is caught at build time from non-generated code too.
var _ Client = (*ClientMock)(nil)
