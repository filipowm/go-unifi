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

	net, err := c.Networks().Get(context.Background(), "s1", "n1")
	require.NoError(t, err)
	assert.Equal(t, "lan", net.Name)
	assert.Equal(t, []string{"GET " + base + "/sites/s1/networks/n1"}, d.calls)
}

// TestGeneratedListAllWrapperDrains proves the ListXxxAll iterator walks the
// envelope and Collect materializes it.
func TestGeneratedListAllWrapperDrains(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{responses: map[string]any{
		base + "/sites/s1/networks": map[string]any{
			"data":       []map[string]any{{"name": "a"}, {"name": "b"}},
			"totalCount": 2,
		},
	}}
	c := New(d, base, nil)

	nets, err := Collect(c.Networks().ListAll(context.Background(), "s1"))
	require.NoError(t, err)
	require.Len(t, nets, 2)
	assert.Equal(t, "a", nets[0].Name)
}

// TestGeneratedListPageWrapperBounded proves ListXxxPage fetches exactly one page
// (no drain-all probe even when more remain) and plumbs offset/limit/filter.
func TestGeneratedListPageWrapperBounded(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{responses: map[string]any{
		base + "/sites/s1/networks": map[string]any{
			"data":       []map[string]any{{"name": "a"}, {"name": "b"}},
			"totalCount": 99, // far more remain, yet a single page must not paginate.
		},
	}}
	c := New(d, base, nil)

	page, err := c.Networks().ListPage(context.Background(), "s1", &ListOptions{Offset: 0, Limit: 2, Filter: "name.eq('a')"})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	assert.Equal(t, 99, page.TotalCount)
	require.Len(t, d.calls, 1, "a single page must fetch exactly one request")
	assert.Contains(t, d.calls[0], "limit=2")
	assert.Contains(t, d.calls[0], "offset=0")
	assert.Contains(t, d.calls[0], "filter=name.eq")
}

// TestGeneratedPatchWrapper exercises the PATCH path via the new Doer.Patch seam.
func TestGeneratedPatchWrapper(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{responses: map[string]any{
		base + "/sites/s1/firewall/policies/p1": map[string]any{"name": "policy"},
	}}
	c := New(d, base, nil)

	_, err := c.Firewall().PatchPolicy(context.Background(), "s1", "p1", PatchFirewallPolicy{})
	require.NoError(t, err)
	assert.Equal(t, []string{"PATCH " + base + "/sites/s1/firewall/policies/p1"}, d.calls)
}

// TestDeleteVouchersGuardsEmptyFilter asserts the required-filter guard fires
// before any transport call.
func TestDeleteVouchersGuardsEmptyFilter(t *testing.T) {
	t.Parallel()
	d := &cannedDoer{}
	c := New(d, base, nil)

	_, err := c.Hotspot().DeleteVouchers(context.Background(), "s1", "")
	require.Error(t, err)
	assert.Empty(t, d.calls, "empty filter must short-circuit before transport")

	_, err = c.Hotspot().DeleteVouchers(context.Background(), "s1", "expired")
	require.NoError(t, err)
	require.Len(t, d.calls, 1)
	assert.Contains(t, d.calls[0], "filter=expired")
}

// TestClientMockSatisfiesInterface wires a stub through the parent mock and its
// per-group mock: c.Info() returns the group mock, whose Get is stubbed.
func TestClientMockSatisfiesInterface(t *testing.T) {
	t.Parallel()
	info := &InfoClientMock{
		GetFunc: func(context.Context) (*Info, error) { return &Info{ApplicationVersion: "10.1.78"}, nil },
	}
	var c Client = &ClientMock{InfoFunc: func() InfoClient { return info }}
	got, err := c.Info().Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "10.1.78", got.ApplicationVersion)
}

// TestGroupMockStandalone exercises a per-group mock directly, without the parent
// Client — each group mock is independently usable in tests.
func TestGroupMockStandalone(t *testing.T) {
	t.Parallel()
	var fw FirewallClient = &FirewallClientMock{
		GetPolicyFunc: func(_ context.Context, _, _ string) (*FirewallPolicy, error) {
			return &FirewallPolicy{Name: "allow-all"}, nil
		},
	}
	p, err := fw.GetPolicy(context.Background(), "s1", "p1")
	require.NoError(t, err)
	assert.Equal(t, "allow-all", p.Name)
}
