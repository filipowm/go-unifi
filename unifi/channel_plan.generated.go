// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
)

type ChannelPlan struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Date       string                  `json:"date"` // ^$|^(20[0-9]{2}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9])Z?$
	RadioTable []ChannelPlanRadioTable `json:"radio_table,omitempty"`
}

func (dst *ChannelPlan) UnmarshalJSON(b []byte) error {
	type Alias ChannelPlan
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

type ChannelPlanRadioTable struct {
	Channel     string `json:"channel,omitempty"`                                                              // [0-9]|[1][0-4]|16|34|36|38|40|42|44|46|48|52|56|60|64|100|104|108|112|116|120|124|128|132|136|140|144|149|153|157|161|165|183|184|185|187|188|189|192|196|auto
	DeviceMAC   string `json:"device_mac,omitempty" validate:"omitempty,mac"`                                  // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	Name        string `json:"name,omitempty"`                                                                 // [a-z]*[0-9]*
	TxPower     string `json:"tx_power,omitempty"`                                                             // [\d]+|auto
	TxPowerMode string `json:"tx_power_mode,omitempty" validate:"omitempty,oneof=auto medium high low custom"` // auto|medium|high|low|custom
	Width       int    `json:"width,omitempty" validate:"omitempty,oneof=20 40 80 160"`                        // 20|40|80|160
}

func (dst *ChannelPlanRadioTable) UnmarshalJSON(b []byte) error {
	type Alias ChannelPlanRadioTable
	aux := &struct {
		*Alias

		Channel numberOrString `json:"channel"`
		TxPower numberOrString `json:"tx_power"`
		Width   emptyStringInt `json:"width"`
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	dst.Channel = string(aux.Channel)
	dst.TxPower = string(aux.TxPower)
	dst.Width = int(aux.Width)

	return nil
}

func (c *client) listChannelPlan(ctx context.Context, site string) ([]ChannelPlan, error) {
	var respBody struct {
		Meta Meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/rest/channelplan", site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody.Data, nil
}

func (c *client) getChannelPlan(ctx context.Context, site, id string) (*ChannelPlan, error) {
	var respBody struct {
		Meta Meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/rest/channelplan/%s", site, id), nil, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *client) deleteChannelPlan(ctx context.Context, site, id string) error {
	err := c.Delete(ctx, fmt.Sprintf("s/%s/rest/channelplan/%s", site, id), struct{}{}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) createChannelPlan(ctx context.Context, site string, d *ChannelPlan) (*ChannelPlan, error) {
	var respBody struct {
		Meta Meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}

	err := c.Post(ctx, fmt.Sprintf("s/%s/rest/channelplan", site), d, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	newResource := respBody.Data[0]

	return &newResource, nil
}

func (c *client) updateChannelPlan(ctx context.Context, site string, d *ChannelPlan) (*ChannelPlan, error) {
	var respBody struct {
		Meta Meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}

	err := c.Put(ctx, fmt.Sprintf("s/%s/rest/channelplan/%s", site, d.ID), d, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	updatedResource := respBody.Data[0]

	return &updatedResource, nil
}
