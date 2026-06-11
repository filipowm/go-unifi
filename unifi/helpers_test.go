package unifi //nolint: testpackage

const (
	localUrl = "https://127.0.0.1:64431"
	testUrl  = "https://test.url"
)

// TestData is a tiny JSON payload reused by the request round-trip tests.
type TestData struct {
	Data string `json:"data"`
}
