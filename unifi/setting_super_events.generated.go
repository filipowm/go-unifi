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

type SettingSuperEvents struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Key string `json:"key"`

	Ignored string `json:"_ignored,omitempty"`
}

func (dst *SettingSuperEvents) UnmarshalJSON(b []byte) error {
	type Alias SettingSuperEvents
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

func (c *client) getSettingSuperEvents(ctx context.Context, site string) (*SettingSuperEvents, error) {
	var respBody struct {
		Meta Meta                 `json:"meta"`
		Data []SettingSuperEvents `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/get/setting/super_events", site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *client) updateSettingSuperEvents(ctx context.Context, site string, d *SettingSuperEvents) (*SettingSuperEvents, error) {
	var respBody struct {
		Meta Meta                 `json:"meta"`
		Data []SettingSuperEvents `json:"data"`
	}

	d.Key = "super_events"
	err := c.Put(ctx, fmt.Sprintf("s/%s/set/setting/super_events", site), d, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	new := respBody.Data[0]

	return &new, nil
}
