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

type SettingConnectivity struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Key string `json:"key"`

	EnableIsolatedWLAN bool   `json:"enable_isolated_wlan"`
	Enabled            bool   `json:"enabled"`
	UplinkHost         string `json:"uplink_host,omitempty"`
	UplinkType         string `json:"uplink_type,omitempty"`
	XMeshEssid         string `json:"x_mesh_essid,omitempty"`
	XMeshPsk           string `json:"x_mesh_psk,omitempty"`
}

func (dst *SettingConnectivity) UnmarshalJSON(b []byte) error {
	type Alias SettingConnectivity
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

func (c *Client) getSettingConnectivity(ctx context.Context, site string) (*SettingConnectivity, error) {
	var respBody struct {
		Meta Meta                  `json:"meta"`
		Data []SettingConnectivity `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/get/setting/connectivity", site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *Client) updateSettingConnectivity(ctx context.Context, site string, d *SettingConnectivity) (*SettingConnectivity, error) {
	var respBody struct {
		Meta Meta                  `json:"meta"`
		Data []SettingConnectivity `json:"data"`
	}

	d.Key = "connectivity"
	err := c.Put(ctx, fmt.Sprintf("s/%s/set/setting/connectivity", site), d, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	new := respBody.Data[0]

	return &new, nil
}
