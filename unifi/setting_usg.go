package unifi

import (
	"context"
)

func (c *client) GetSettingUsg(ctx context.Context, site string) (*SettingUsg, error) {
	return c.getSettingUsg(ctx, site)
}

func (c *client) UpdateSettingUsg(ctx context.Context, site string, d *SettingUsg) (*SettingUsg, error) {
	return c.updateSettingUsg(ctx, site, d)
}
