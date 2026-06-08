package unifi_test

import (
	"context"
	"testing"

	"github.com/filipowm/go-unifi/v2/unifi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientMockUsableAsClient demonstrates the moq-generated ClientMock standing
// in for the public Client interface: a consumer wires a Func, calls through the
// interface type, and inspects the recorded calls. This locks in the
// promise that downstreams can unit-test against go-unifi without a controller.
func TestClientMockUsableAsClient(t *testing.T) {
	t.Parallel()

	want := &unifi.Site{ID: "s1", Name: "default"}
	mock := &unifi.ClientMock{
		GetSiteFunc: func(_ context.Context, id string) (*unifi.Site, error) {
			assert.Equal(t, "s1", id)
			return want, nil
		},
	}

	// Use it through the interface type, as a real consumer would.
	var c unifi.Client = mock
	got, err := c.GetSite(context.Background(), "s1")
	require.NoError(t, err)
	assert.Same(t, want, got)

	// The mock records calls for assertions.
	calls := mock.GetSiteCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "s1", calls[0].ID)
}
