package official

import (
	"context"
	"fmt"
)

// maxPageLimit is the largest page size the Official list endpoints accept
// (the spec caps limit at 200; default is 25). We request the max to minimize
// round-trips during auto-pagination.
const maxPageLimit = 200

// page is the {offset,limit,count,totalCount,data[]} envelope returned by the
// paginated Official list endpoints.
type page[T any] struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       []T `json:"data"`
}

// listAll fetches every page of a paginated list endpoint and appends the
// decoded items to out. It walks offset/limit until an empty page or the
// reported totalCount is reached; an empty page always terminates so a
// misreported totalCount can never spin forever.
func listAll[T any](ctx context.Context, doer Doer, basePath string, out *[]T) error {
	offset := 0
	for {
		var p page[T]
		url := fmt.Sprintf("%s?offset=%d&limit=%d", basePath, offset, maxPageLimit)
		if err := doer.Get(ctx, url, nil, &p); err != nil {
			return err
		}
		*out = append(*out, p.Data...)
		offset += len(p.Data)
		if len(p.Data) == 0 || offset >= p.TotalCount {
			return nil
		}
	}
}
