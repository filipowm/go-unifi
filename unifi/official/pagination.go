package official

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"strconv"
)

// maxPageLimit is the largest page size the Official list endpoints accept
// (the spec caps limit at 200; default is 25). It is the default page size and
// the size ListAll uses per fetch to minimize drain round-trips.
const maxPageLimit = 200

// ListOptions bounds a single ListPage call. A nil *ListOptions means the first
// page at the default size; a non-positive Limit defaults to maxPageLimit and a
// larger one is clamped to it. Offset 0 is a valid explicit value.
type ListOptions struct {
	Offset int
	Limit  int
	Filter string // optional server-side filter expression
}

// Page is the public face of the {offset,limit,count,totalCount,data} envelope
// the Official list endpoints return. Items is the decoded data slice.
type Page[T any] struct {
	Items      []T
	Offset     int
	Limit      int
	Count      int
	TotalCount int
}

// page is the raw {offset,limit,count,totalCount,data[]} envelope.
type page[T any] struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       []T `json:"data"`
}

// Collect materializes an iterator into a slice, short-circuiting on the first
// error. It is the explicit opt-in for callers who genuinely want every item in
// memory rather than streaming via ListAll.
func Collect[T any](seq iter.Seq2[T, error]) ([]T, error) {
	var out []T
	for item, err := range seq {
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

// listPage fetches exactly ONE page, resolving nil/partial opts to sane defaults.
// The caller (wrapper) runs the capability gate before this.
func listPage[T any](ctx context.Context, doer Doer, basePath string, opts *ListOptions) (Page[T], error) {
	offset, limit, filter := resolveOptions(opts)
	p, err := fetchPage[T](ctx, doer, basePath, offset, limit, filter)
	if err != nil {
		return Page[T]{}, err
	}
	return Page[T]{Items: p.Data, Offset: p.Offset, Limit: p.Limit, Count: p.Count, TotalCount: p.TotalCount}, nil
}

// listSeq returns a lazy iterator that drains every item across pages. It runs
// the capability gate before the first fetch (surfacing a failed check as the
// first yielded error), forwards filter on every request, stops on an empty page
// or once offset reaches totalCount, and aborts immediately — issuing no further
// request — when the consumer breaks (yield returns false).
func listSeq[T any](ctx context.Context, c *apiClient, basePath, filter string) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		if err := c.check(ctx); err != nil {
			yield(zero, err)
			return
		}
		offset := 0
		for {
			p, err := fetchPage[T](ctx, c.doer, basePath, offset, maxPageLimit, filter)
			if err != nil {
				yield(zero, err)
				return
			}
			for _, item := range p.Data {
				if !yield(item, nil) {
					return
				}
			}
			offset += len(p.Data)
			if len(p.Data) == 0 || offset >= p.TotalCount {
				return
			}
		}
	}
}

// resolveOptions normalizes list options: nil => first page at maxPageLimit; a
// non-positive or oversized Limit clamps to maxPageLimit; a negative Offset to 0.
func resolveOptions(opts *ListOptions) (int, int, string) {
	if opts == nil {
		return 0, maxPageLimit, ""
	}
	limit := opts.Limit
	if limit <= 0 || limit > maxPageLimit {
		limit = maxPageLimit
	}
	return max(opts.Offset, 0), limit, opts.Filter
}

// fetchPage GETs one page at the given offset/limit, forwarding filter when set.
func fetchPage[T any](ctx context.Context, doer Doer, basePath string, offset, limit int, filter string) (page[T], error) {
	var p page[T]
	err := doer.Get(ctx, pageURL(basePath, offset, limit, filter), nil, &p)
	return p, err
}

// pageURL appends the offset/limit (and optional filter) query params to basePath.
func pageURL(basePath string, offset, limit int, filter string) string {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	if filter != "" {
		q.Set("filter", filter)
	}
	return fmt.Sprintf("%s?%s", basePath, q.Encode())
}
