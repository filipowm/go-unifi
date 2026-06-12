package official

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/google/uuid"
)

// ErrSiteNotFound is returned by ResolveID when the given legacy site name
// has no matching Official-API UUID in the full site list.
var ErrSiteNotFound = errors.New("site not found")

// SiteOverview is one entry from GET /v1/sites. ID is the Official-API site
// UUID; InternalReference is the legacy site name used by the Internal API
// (so callers keep passing the familiar name); Name is the display name.
type SiteOverview struct {
	ID                uuid.UUID `json:"id"`
	InternalReference string    `json:"internalReference"`
	Name              string    `json:"name"`
}

// ListPage returns one page of local sites; nil opts fetches the first page at
// the default size.
func (c sitesClient) ListPage(ctx context.Context, opts *ListOptions) (Page[SiteOverview], error) {
	if err := c.check(ctx); err != nil {
		return Page[SiteOverview]{}, err
	}
	p, err := listPage[SiteOverview](ctx, c.doer, c.path("/sites"), opts)
	if err != nil {
		return Page[SiteOverview]{}, fmt.Errorf("failed listing sites: %w", err)
	}
	return p, nil
}

// ListAll lazily drains every local site across pages; pass "" filter to drain unfiltered.
func (c sitesClient) ListAll(ctx context.Context, filter string) iter.Seq2[SiteOverview, error] {
	return listSeq[SiteOverview](ctx, c.apiClient, c.path("/sites"), filter)
}

// ResolveID maps a legacy site name (the Internal-API identifier, carried as
// internalReference) to its Official-API site UUID. The full site list is cached
// on first miss so repeated lookups avoid a round-trip.
//
// The returned uuid.UUID is the siteId required by all resource-group methods
// (Networks, Firewall, Devices, etc.). Pass it directly as the siteId argument.
func (c sitesClient) ResolveID(ctx context.Context, name string) (uuid.UUID, error) {
	if id, ok := c.cachedSiteID(name); ok {
		return id, nil
	}
	sites, err := Collect(c.ListAll(ctx, ""))
	if err != nil {
		return uuid.UUID{}, err
	}
	c.mu.Lock()
	if c.siteIDs == nil {
		c.siteIDs = make(map[string]uuid.UUID, len(sites))
	}
	for _, s := range sites {
		c.siteIDs[s.InternalReference] = s.ID
	}
	c.mu.Unlock()
	if id, ok := c.cachedSiteID(name); ok {
		return id, nil
	}
	return uuid.UUID{}, fmt.Errorf("%w: %q", ErrSiteNotFound, name)
}

// cachedSiteID returns the cached UUID for a legacy site name, if present.
func (c *apiClient) cachedSiteID(name string) (uuid.UUID, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.siteIDs[name]
	return id, ok
}
