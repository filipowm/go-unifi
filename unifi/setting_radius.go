package unifi

import (
	"context"
)

func (c *client) GetSettingRadius(ctx context.Context, site string) (*SettingRadius, error) {
	return c.getSettingRadius(ctx, site)
}

func (c *client) UpdateSettingRadius(ctx context.Context, site string, d *SettingRadius) (*SettingRadius, error) {
	return c.updateSettingRadius(ctx, site, d)
}
