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

type ContentFiltering struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	AllowList  []string                 `json:"allow_list"`
	BlockList  []string                 `json:"block_list"`
	Categories []string                 `json:"categories" validate:"omitempty,dive,oneof=FAMILY ADVERTISEMENT"` // FAMILY|ADVERTISEMENT
	ClientMACs []string                 `json:"client_macs,omitempty" validate:"omitempty,dive,mac"`             // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	Enabled    bool                     `json:"enabled"`
	Name       string                   `json:"name,omitempty"`
	NetworkIDs []string                 `json:"network_ids,omitempty"`
	SafeSearch []string                 `json:"safe_search,omitempty" validate:"omitempty,dive,oneof=GOOGLE YOUTUBE BING"` // GOOGLE|YOUTUBE|BING
	Schedule   ContentFilteringSchedule `json:"schedule,omitempty"`
}

func (dst *ContentFiltering) UnmarshalJSON(b []byte) error {
	type Alias ContentFiltering
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

type ContentFilteringSchedule struct {
	Date           string   `json:"date,omitempty"`                                                                             // ^$|^(20[0-9]{2})-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$
	DateEnd        string   `json:"date_end,omitempty"`                                                                         // ^$|^(20[0-9]{2})-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$
	DateStart      string   `json:"date_start,omitempty"`                                                                       // ^$|^(20[0-9]{2})-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$
	Mode           string   `json:"mode,omitempty" validate:"omitempty,oneof=ALWAYS EVERY_DAY EVERY_WEEK ONE_TIME_ONLY CUSTOM"` // ALWAYS|EVERY_DAY|EVERY_WEEK|ONE_TIME_ONLY|CUSTOM
	RepeatOnDays   []string `json:"repeat_on_days,omitempty" validate:"omitempty,dive,oneof=mon tue wed thu fri sat sun"`       // mon|tue|wed|thu|fri|sat|sun
	TimeAllDay     bool     `json:"time_all_day"`
	TimeRangeEnd   string   `json:"time_range_end,omitempty"`   // ^[0-9][0-9]:[0-9][0-9]$
	TimeRangeStart string   `json:"time_range_start,omitempty"` // ^[0-9][0-9]:[0-9][0-9]$
}

func (dst *ContentFilteringSchedule) UnmarshalJSON(b []byte) error {
	type Alias ContentFilteringSchedule
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

func (c *client) listContentFiltering(ctx context.Context, site string) ([]ContentFiltering, error) {
	var respBody []ContentFiltering

	err := c.Get(ctx, fmt.Sprintf("%s/site/%s/content-filtering", c.apiPaths.ApiV2Path, site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (c *client) getContentFiltering(ctx context.Context, site, id string) (*ContentFiltering, error) {
	var respBody ContentFiltering

	err := c.Get(ctx, fmt.Sprintf("%s/site/%s/content-filtering/%s", c.apiPaths.ApiV2Path, site, id), nil, &respBody)
	if err != nil {
		return nil, err
	}
	if respBody.ID == "" {
		return nil, ErrNotFound
	}
	return &respBody, nil
}

func (c *client) deleteContentFiltering(ctx context.Context, site, id string) error {
	err := c.Delete(ctx, fmt.Sprintf("%s/site/%s/content-filtering/%s", c.apiPaths.ApiV2Path, site, id), struct{}{}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) createContentFiltering(ctx context.Context, site string, d *ContentFiltering) (*ContentFiltering, error) {
	var respBody ContentFiltering

	err := c.Post(ctx, fmt.Sprintf("%s/site/%s/content-filtering", c.apiPaths.ApiV2Path, site), d, &respBody)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *client) updateContentFiltering(ctx context.Context, site string, d *ContentFiltering) (*ContentFiltering, error) {
	var respBody ContentFiltering

	err := c.Put(ctx, fmt.Sprintf("%s/site/%s/content-filtering/%s", c.apiPaths.ApiV2Path, site, d.ID), d, &respBody)
	if err != nil {
		return nil, err
	}
	return &respBody, nil
}
