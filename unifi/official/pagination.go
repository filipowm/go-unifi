package official

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// maxPageLimit is the largest page size the Official list endpoints accept
// (the spec caps limit at 200; default is 25). We request the max to minimize
// round-trips during auto-pagination.
const maxPageLimit = 200

// ListOption configures a single list request. With no options a list method
// auto-paginates (drains every page); WithOffset/WithLimit switch it to a single
// bounded page, and WithFilter narrows the result server-side.
type ListOption func(*listParams)

// listParams is the resolved option set. offset/limit are pointers so "explicit
// pagination requested" is distinguishable from a zero default.
type listParams struct {
	offset *int
	limit  *int
	filter string
}

// WithOffset requests a single bounded page starting at offset (disables
// auto-pagination).
func WithOffset(offset int) ListOption { return func(p *listParams) { p.offset = &offset } }

// WithLimit requests a single bounded page of at most limit items (disables
// auto-pagination).
func WithLimit(limit int) ListOption { return func(p *listParams) { p.limit = &limit } }

// WithFilter applies a server-side filter expression; it composes with both the
// auto-paginating default and explicit pagination.
func WithFilter(filter string) ListOption { return func(p *listParams) { p.filter = filter } }

// bounded reports whether the caller asked for explicit pagination (a single
// bounded page) rather than the drain-all default.
func (p listParams) bounded() bool { return p.offset != nil || p.limit != nil }

// page is the {offset,limit,count,totalCount,data[]} envelope returned by the
// paginated Official list endpoints.
type page[T any] struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       []T `json:"data"`
}

// listAll fetches a paginated list endpoint into out. With no options it drains
// every page (terminating on an empty page or when accumulated == totalCount);
// WithOffset/WithLimit fetch exactly one bounded page instead. A filter, when
// set, is forwarded on every request. Error mapping is identical in both modes:
// the Doer maps meta.rc==error to *ServerError and yields an empty data[].
func listAll[T any](ctx context.Context, doer Doer, basePath string, out *[]T, opts ...ListOption) error {
	var lp listParams
	for _, o := range opts {
		o(&lp)
	}

	if lp.bounded() {
		offset, limit := derefOr(lp.offset, 0), derefOr(lp.limit, maxPageLimit)
		p, err := fetchPage[T](ctx, doer, basePath, offset, limit, lp.filter)
		if err != nil {
			return err
		}
		*out = append(*out, p.Data...)
		return nil
	}

	offset := 0
	for {
		p, err := fetchPage[T](ctx, doer, basePath, offset, maxPageLimit, lp.filter)
		if err != nil {
			return err
		}
		*out = append(*out, p.Data...)
		offset += len(p.Data)
		// Terminate on an empty page (definitive end-of-list) or when accumulated
		// equals totalCount. If the server underreports totalCount, the equal-count
		// check never fires and the empty-page terminator catches it.
		if len(p.Data) == 0 || offset == p.TotalCount {
			return nil
		}
	}
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

// derefOr returns *p, or def when p is nil.
func derefOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
