// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
)

// just to fix compile issues with the import
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
)

const SettingMagicSiteToSiteVpnKey = "magic_site_to_site_vpn"

type SettingMagicSiteToSiteVpn struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Key string `json:"key"`

	Enabled bool `json:"enabled"`
}

func (dst *SettingMagicSiteToSiteVpn) UnmarshalJSON(b []byte) error {
	type Alias SettingMagicSiteToSiteVpn
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

// Update SettingMagicSiteToSiteVpn Experimental! This function is not yet stable and may change in the future.
func (c *client) GetSettingMagicSiteToSiteVpn(ctx context.Context, site string) (*SettingMagicSiteToSiteVpn, error) {
	s, f, err := c.GetSetting(ctx, site, SettingMagicSiteToSiteVpnKey)
	if err != nil {
		return nil, err
	}
	if s.Key != SettingMagicSiteToSiteVpnKey {
		return nil, fmt.Errorf("unexpected setting key received. Requested: %q, received: %q", SettingMagicSiteToSiteVpnKey, s.Key)
	}
	return f.(*SettingMagicSiteToSiteVpn), nil
}

// Update SettingMagicSiteToSiteVpn Experimental! This function is not yet stable and may change in the future.
func (c *client) UpdateSettingMagicSiteToSiteVpn(ctx context.Context, site string, s *SettingMagicSiteToSiteVpn) (*SettingMagicSiteToSiteVpn, error) {
	result, err := c.SetSetting(ctx, site, SettingMagicSiteToSiteVpnKey, s)
	if err != nil {
		return nil, err
	}
	return result.(*SettingMagicSiteToSiteVpn), nil
}
