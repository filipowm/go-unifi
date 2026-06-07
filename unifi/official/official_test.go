package official //nolint:testpackage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encode JSON-marshals v and unmarshals it into respBody, mimicking a transport
// decoding a canned response.
func encode(v, respBody any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, respBody)
}

// fakeDoer is a minimal in-memory Doer. It proves the official package needs
// nothing from the parent unifi package: the transport is a plain structural
// interface satisfied here by a test double.
type fakeDoer struct {
	responses map[string]any // path (sans query) -> value JSON-marshaled into respBody
	calls     []string
	err       error
}

func (f *fakeDoer) Get(_ context.Context, apiPath string, _, respBody any) error {
	f.calls = append(f.calls, apiPath)
	if f.err != nil {
		return f.err
	}
	// Match on the path before any query string so paginated calls resolve.
	key, _, _ := strings.Cut(apiPath, "?")
	v, ok := f.responses[key]
	if !ok {
		return fmt.Errorf("no canned response for %s", apiPath)
	}
	return encode(v, respBody)
}

func (f *fakeDoer) Post(context.Context, string, any, any) error   { return nil }
func (f *fakeDoer) Put(context.Context, string, any, any) error    { return nil }
func (f *fakeDoer) Patch(context.Context, string, any, any) error  { return nil }
func (f *fakeDoer) Delete(context.Context, string, any, any) error { return nil }

const base = "/proxy/network/integration/v1"

func TestGetInfo(t *testing.T) {
	t.Parallel()
	d := &fakeDoer{responses: map[string]any{base + "/info": Info{ApplicationVersion: "10.1.78"}}}
	c := New(d, base, nil)

	info, err := c.Info().Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "10.1.78", info.ApplicationVersion)
	assert.Equal(t, []string{base + "/info"}, d.calls)
}

func TestGateBlocksOperations(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("blocked")
	d := &fakeDoer{}
	c := New(d, base, func(context.Context) error { return sentinel })

	_, err := c.Info().Get(context.Background())
	require.ErrorIs(t, err, sentinel)
	assert.Empty(t, d.calls, "gate must short-circuit before any transport call")

	_, err = c.Sites().ListPage(context.Background(), nil)
	require.ErrorIs(t, err, sentinel)

	// ListAll runs the gate before paging and surfaces failure as the first
	// yielded error — no transport call is made.
	_, err = Collect(c.Sites().ListAll(context.Background()))
	require.ErrorIs(t, err, sentinel)
	assert.Empty(t, d.calls, "gate must short-circuit the iterator before any transport call")
}

// sitePage builds one {offset,limit,count,totalCount,data} envelope.
func sitePage(offset, total int, data []SiteOverview) page[SiteOverview] {
	return page[SiteOverview]{Offset: offset, Limit: maxPageLimit, Count: len(data), TotalCount: total, Data: data}
}

func TestListAllDrainsAcrossPages(t *testing.T) {
	t.Parallel()
	// 250 sites across two pages of <=200 — ListAll must walk both.
	all := make([]SiteOverview, 0, 250)
	for i := range 250 {
		all = append(all, SiteOverview{ID: fmt.Sprintf("uuid-%d", i), InternalReference: fmt.Sprintf("site%d", i), Name: fmt.Sprintf("Site %d", i)})
	}

	calls := 0
	d := &pagingDoer{fn: func(_ string, respBody any) error {
		start := calls * maxPageLimit
		calls++
		if start >= len(all) {
			return encode(sitePage(start, len(all), nil), respBody)
		}
		end := min(start+maxPageLimit, len(all))
		return encode(sitePage(start, len(all), all[start:end]), respBody)
	}}
	c := New(d, base, nil)

	sites, err := Collect(c.Sites().ListAll(context.Background()))
	require.NoError(t, err)
	assert.Len(t, sites, 250)
	// Two fetches: page 1 (200 items), page 2 (50 items, offset==totalCount -> stop).
	assert.Equal(t, 2, calls, "expected exactly two pages")
}

// TestListAllBreakStopsFetching asserts a consumer break aborts the iterator
// before it pages again — draining is abortable, the core of the new contract.
func TestListAllBreakStopsFetching(t *testing.T) {
	t.Parallel()
	all := make([]SiteOverview, 0, 400)
	for i := range 400 {
		all = append(all, SiteOverview{ID: fmt.Sprintf("u%d", i), InternalReference: fmt.Sprintf("r%d", i), Name: "n"})
	}
	calls := 0
	d := &pagingDoer{fn: func(_ string, respBody any) error {
		start := calls * maxPageLimit
		calls++
		end := min(start+maxPageLimit, len(all))
		return encode(sitePage(start, len(all), all[start:end]), respBody)
	}}
	c := New(d, base, nil)

	seen := 0
	for _, err := range c.Sites().ListAll(context.Background()) {
		require.NoError(t, err)
		seen++
		if seen == 5 {
			break
		}
	}
	assert.Equal(t, 5, seen)
	assert.Equal(t, 1, calls, "break must stop the iterator without fetching the next page")
}

// TestListPageFetchesSinglePage asserts the bounded default: one page only, even
// when the server reports far more remain, and nil opts means the first page.
func TestListPageFetchesSinglePage(t *testing.T) {
	t.Parallel()
	calls := 0
	d := &pagingDoer{fn: func(apiPath string, respBody any) error {
		calls++
		assert.Contains(t, apiPath, "offset=0")
		assert.Contains(t, apiPath, "limit=200", "nil opts must request the default page size")
		return encode(sitePage(0, 9999, []SiteOverview{{ID: "u1", InternalReference: "r1", Name: "n"}}), respBody)
	}}
	c := New(d, base, nil)

	page, err := c.Sites().ListPage(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, 9999, page.TotalCount)
	assert.Equal(t, 1, calls, "ListPage must fetch exactly one page, never drain")
}

// TestListPageClampsLimit asserts a non-positive limit and an over-cap limit both
// resolve to maxPageLimit, and an explicit offset is forwarded verbatim.
func TestListPageClampsLimit(t *testing.T) {
	t.Parallel()
	cases := map[string]int{"zero": 0, "negative": -7, "over-cap": maxPageLimit + 500}
	for name, limit := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var url string
			d := &pagingDoer{fn: func(apiPath string, respBody any) error {
				url = apiPath
				return encode(sitePage(10, 1, nil), respBody)
			}}
			c := New(d, base, nil)

			_, err := c.Sites().ListPage(context.Background(), &ListOptions{Offset: 10, Limit: limit})
			require.NoError(t, err)
			assert.Contains(t, url, "limit=200", "limit must clamp to maxPageLimit")
			assert.Contains(t, url, "offset=10", "explicit offset must be forwarded")
		})
	}
}

// TestListEmptyDataYieldsEmptyResult asserts an empty data[] is a benign empty
// result (no nil-deref) in both ListPage and ListAll.
func TestListEmptyDataYieldsEmptyResult(t *testing.T) {
	t.Parallel()
	d := &fakeDoer{responses: map[string]any{base + "/sites": sitePage(0, 0, nil)}}
	c := New(d, base, nil)

	page, err := c.Sites().ListPage(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, page.Items)

	sites, err := Collect(c.Sites().ListAll(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, sites)
}

func TestResolveSiteIDCachesByInternalReference(t *testing.T) {
	t.Parallel()
	d := &fakeDoer{responses: map[string]any{
		base + "/sites": sitePage(0, 2, []SiteOverview{
			{ID: "uuid-default", InternalReference: "default", Name: "Default"},
			{ID: "uuid-other", InternalReference: "other", Name: "Other"},
		}),
	}}
	c := New(d, base, nil)

	id, err := c.Sites().ResolveID(context.Background(), "default")
	require.NoError(t, err)
	assert.Equal(t, "uuid-default", id)

	// Second lookup is served from cache: no further transport call.
	before := len(d.calls)
	id2, err := c.Sites().ResolveID(context.Background(), "other")
	require.NoError(t, err)
	assert.Equal(t, "uuid-other", id2)
	assert.Len(t, d.calls, before, "cached lookup must not hit transport again")

	_, err = c.Sites().ResolveID(context.Background(), "ghost")
	require.Error(t, err)
}

// TestResolveSiteIDNotFoundSentinel asserts that ResolveSiteID wraps ErrSiteNotFound
// so callers can match it with errors.Is.
func TestResolveSiteIDNotFoundSentinel(t *testing.T) {
	t.Parallel()
	d := &fakeDoer{responses: map[string]any{
		base + "/sites": sitePage(0, 1, []SiteOverview{
			{ID: "uuid-default", InternalReference: "default", Name: "Default"},
		}),
	}}
	c := New(d, base, nil)

	_, err := c.Sites().ResolveID(context.Background(), "ghost")
	require.ErrorIs(t, err, ErrSiteNotFound, "not-found must be matchable with errors.Is")
}

// TestGetInfoTransportError asserts the error from the Doer is wrapped and
// propagated by GetInfo.
func TestGetInfoTransportError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("dial tcp: connection refused")
	d := &fakeDoer{err: sentinel}
	c := New(d, base, nil)

	_, err := c.Info().Get(context.Background())
	require.ErrorIs(t, err, sentinel)
}

// TestListTransportError asserts a Doer error is wrapped and propagated
// identically by both ListPage and ListAll (the error envelope, e.g. a mapped
// *ServerError from the transport, flows through unchanged).
func TestListTransportError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("connection reset")
	d := &fakeDoer{err: sentinel}
	c := New(d, base, nil)

	_, err := c.Sites().ListPage(context.Background(), nil)
	require.ErrorIs(t, err, sentinel)

	_, err = Collect(c.Sites().ListAll(context.Background()))
	require.ErrorIs(t, err, sentinel)
}

// TestResolveSiteIDTransportError asserts the error from the Doer is wrapped
// and propagated by ResolveSiteID.
func TestResolveSiteIDTransportError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("timeout")
	d := &fakeDoer{err: sentinel}
	c := New(d, base, nil)

	_, err := c.Sites().ResolveID(context.Background(), "default")
	require.ErrorIs(t, err, sentinel)
}

// TestListAllTerminatesOnEmptyPageWithHighTotalCount asserts the iterator stops
// when data dries up even when the server still reports a large totalCount.
func TestListAllTerminatesOnEmptyPageWithHighTotalCount(t *testing.T) {
	t.Parallel()
	const reportedTotal = 999
	realData := []SiteOverview{
		{ID: "uuid-a", InternalReference: "a", Name: "A"},
		{ID: "uuid-b", InternalReference: "b", Name: "B"},
	}
	calls := 0
	d := &pagingDoer{fn: func(_ string, respBody any) error {
		calls++
		switch calls {
		case 1:
			// First page: return real data but lie about totalCount.
			return encode(sitePage(0, reportedTotal, realData), respBody)
		default:
			// Second page: empty data — the iterator must terminate here.
			return encode(sitePage(len(realData), reportedTotal, nil), respBody)
		}
	}}
	c := New(d, base, nil)

	sites, err := Collect(c.Sites().ListAll(context.Background()))
	require.NoError(t, err)
	assert.Len(t, sites, len(realData))
	assert.Equal(t, 2, calls, "must stop after the empty page, not spin to totalCount")
}

// TestListSeqForwardsFilterAcrossPages asserts the iterator re-sends the filter
// on EVERY page request, not just the first — exercised via listSeq, since the
// public ListXxxAll drains unfiltered.
func TestListSeqForwardsFilterAcrossPages(t *testing.T) {
	t.Parallel()
	// 250 sites across two pages of <=200 so the filter is asserted on more than
	// one request — a single full page would never exercise the "across pages" path.
	all := make([]SiteOverview, 0, 250)
	for i := range 250 {
		all = append(all, SiteOverview{ID: fmt.Sprintf("u%d", i), InternalReference: fmt.Sprintf("r%d", i), Name: "n"})
	}

	var urls []string
	calls := 0
	d := &pagingDoer{fn: func(apiPath string, respBody any) error {
		urls = append(urls, apiPath)
		start := calls * maxPageLimit
		calls++
		end := min(start+maxPageLimit, len(all))
		return encode(sitePage(start, len(all), all[start:end]), respBody)
	}}
	ac := &apiClient{doer: d, basePath: base}

	sites, err := Collect(listSeq[SiteOverview](context.Background(), ac, ac.path("/sites"), "name.eq('n')"))
	require.NoError(t, err)
	require.Len(t, sites, 250)
	require.Len(t, urls, 2, "expected the filter to be forwarded across both pages")
	for _, u := range urls {
		assert.Contains(t, u, "filter=name.eq")
	}
}

// TestListPagePropagatesTransportError asserts ListPage wraps and propagates the
// transport error.
func TestListPagePropagatesTransportError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	d := &pagingDoer{fn: func(string, any) error { return sentinel }}
	c := New(d, base, nil)

	_, err := c.Sites().ListPage(context.Background(), &ListOptions{Limit: 10})
	require.ErrorIs(t, err, sentinel)
}

// pagingDoer drives ListSites pagination through a custom per-call function.
type pagingDoer struct {
	fn func(apiPath string, respBody any) error
}

func (p *pagingDoer) Get(_ context.Context, apiPath string, _, respBody any) error {
	return p.fn(apiPath, respBody)
}
func (p *pagingDoer) Post(context.Context, string, any, any) error   { return nil }
func (p *pagingDoer) Put(context.Context, string, any, any) error    { return nil }
func (p *pagingDoer) Patch(context.Context, string, any, any) error  { return nil }
func (p *pagingDoer) Delete(context.Context, string, any, any) error { return nil }
