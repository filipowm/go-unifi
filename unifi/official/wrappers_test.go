package official //nolint:testpackage

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cannedDoer decodes a canned response for every verb, keyed by path (sans
// query), and records the method/path pairs it served — enough to exercise the
// generated tri-shape wrappers end-to-end.
type cannedDoer struct {
	responses map[string]any
	calls     []string
}

func (d *cannedDoer) Get(_ context.Context, p string, _, r any) error {
	return d.serve("GET", p, r)
}

func (d *cannedDoer) Post(_ context.Context, p string, _, r any) error {
	return d.serve("POST", p, r)
}

func (d *cannedDoer) Put(_ context.Context, p string, _, r any) error {
	return d.serve("PUT", p, r)
}

func (d *cannedDoer) Patch(_ context.Context, p string, _, r any) error {
	return d.serve("PATCH", p, r)
}

func (d *cannedDoer) Delete(_ context.Context, p string, _, r any) error {
	return d.serve("DELETE", p, r)
}

func (d *cannedDoer) serve(method, apiPath string, respBody any) error {
	key, _, _ := strings.Cut(apiPath, "?")
	d.calls = append(d.calls, method+" "+apiPath)
	v, ok := d.responses[key]
	if !ok || respBody == nil {
		return nil
	}
	return encode(v, respBody)
}

// TestGeneratedGetWrapper exercises a single-resource GET wrapper.
func TestGeneratedGetWrapper(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{responses: map[string]any{
		base + "/sites/s1/networks/n1": map[string]any{"name": "lan"},
	}}
	c := New(d, base, nil)

	net, err := c.GetNetworkDetails(context.Background(), "s1", "n1")
	require.NoError(t, err)
	assert.Equal(t, "lan", net.Name)
	assert.Equal(t, []string{"GET " + base + "/sites/s1/networks/n1"}, d.calls)
}

// TestGeneratedListWrapperPaginates proves a list wrapper walks the envelope.
func TestGeneratedListWrapperPaginates(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{responses: map[string]any{
		base + "/sites/s1/networks": map[string]any{
			"data":       []map[string]any{{"name": "a"}, {"name": "b"}},
			"totalCount": 2,
		},
	}}
	c := New(d, base, nil)

	nets, err := c.GetNetworksOverviewPage(context.Background(), "s1")
	require.NoError(t, err)
	require.Len(t, nets, 2)
	assert.Equal(t, "a", nets[0].Name)
}

// TestGeneratedPatchWrapper exercises the PATCH path via the new Doer.Patch seam.
func TestGeneratedPatchWrapper(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{responses: map[string]any{
		base + "/sites/s1/firewall/policies/p1": map[string]any{"name": "policy"},
	}}
	c := New(d, base, nil)

	_, err := c.PatchFirewallPolicy(context.Background(), "s1", "p1", PatchFirewallPolicy{})
	require.NoError(t, err)
	assert.Equal(t, []string{"PATCH " + base + "/sites/s1/firewall/policies/p1"}, d.calls)
}

// TestDeleteVouchersGuardsEmptyFilter asserts the required-filter guard fires
// before any transport call.
func TestDeleteVouchersGuardsEmptyFilter(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{}
	c := New(d, base, nil)

	_, err := c.DeleteVouchers(context.Background(), "s1", "")
	require.Error(t, err)
	assert.Empty(t, d.calls, "empty filter must short-circuit before transport")

	_, err = c.DeleteVouchers(context.Background(), "s1", "expired")
	require.NoError(t, err)
	require.Len(t, d.calls, 1)
	assert.Contains(t, d.calls[0], "filter=expired")
}

// TestClientMockSatisfiesInterface wires a stub through the generated mock.
func TestClientMockSatisfiesInterface(t *testing.T) {
	t.Parallel()
	var c Client = &ClientMock{
		GetInfoFunc: func(context.Context) (*Info, error) { return &Info{ApplicationVersion: "10.1.78"}, nil },
	}
	info, err := c.GetInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "10.1.78", info.ApplicationVersion)
}
