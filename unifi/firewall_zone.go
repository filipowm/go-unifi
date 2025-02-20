package unifi

import "context"

func (c *client) ListFirewallZone(ctx context.Context, site string) ([]FirewallZone, error) {
	return c.listFirewallZone(ctx, site)
}

func (c *client) GetFirewallZone(ctx context.Context, site, id string) (*FirewallZone, error) {
	return c.getFirewallZone(ctx, site, id)
}

func (c *client) DeleteFirewallZone(ctx context.Context, site, id string) error {
	return c.deleteFirewallZone(ctx, site, id)
}

func (c *client) CreateFirewallZone(ctx context.Context, site string, d *FirewallZone) (*FirewallZone, error) {
	return c.createFirewallZone(ctx, site, d)
}

func (c *client) UpdateFirewallZone(ctx context.Context, site string, d *FirewallZone) (*FirewallZone, error) {
	return c.updateFirewallZone(ctx, site, d)
}
