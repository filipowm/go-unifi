package unifi

import "context"

func (c *client) ListContentFiltering(ctx context.Context, site string) ([]ContentFiltering, error) {
	return c.listContentFiltering(ctx, site)
}

func (c *client) DeleteContentFiltering(ctx context.Context, site, id string) error {
	return c.deleteContentFiltering(ctx, site, id)
}

func (c *client) CreateContentFiltering(ctx context.Context, site string, d *ContentFiltering) (*ContentFiltering, error) {
	return c.createContentFiltering(ctx, site, d)
}

func (c *client) UpdateContentFiltering(ctx context.Context, site string, d *ContentFiltering) (*ContentFiltering, error) {
	return c.updateContentFiltering(ctx, site, d)
}
