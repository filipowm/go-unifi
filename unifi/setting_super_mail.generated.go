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

type SettingSuperMail struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Key string `json:"key"`

	Provider string `json:"provider,omitempty" validate:"omitempty,oneof=smtp cloud disabled"` // smtp|cloud|disabled
}

func (dst *SettingSuperMail) UnmarshalJSON(b []byte) error {
	type Alias SettingSuperMail
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

func (c *client) getSettingSuperMail(ctx context.Context, site string) (*SettingSuperMail, error) {
	var respBody struct {
		Meta Meta               `json:"meta"`
		Data []SettingSuperMail `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/get/setting/super_mail", site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *client) updateSettingSuperMail(ctx context.Context, site string, d *SettingSuperMail) (*SettingSuperMail, error) {
	var respBody struct {
		Meta Meta               `json:"meta"`
		Data []SettingSuperMail `json:"data"`
	}

	d.Key = "super_mail"
	err := c.Put(ctx, fmt.Sprintf("s/%s/set/setting/super_mail", site), d, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	new := respBody.Data[0]

	return &new, nil
}
