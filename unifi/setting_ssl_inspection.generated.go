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

const SettingSslInspectionKey = "ssl_inspection"

// Self-register this setting's fields factory so the settingFactories registry
// in setting_registry.go stays a 1:1 reflection of the generated catalog and
// can never drift from it by hand.
func init() { //nolint:gochecknoinits
	registerSetting(SettingSslInspectionKey, func() any { return &SettingSslInspection{} })
}

type SettingSslInspection struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Key string `json:"key"`

	State string `json:"state,omitempty" validate:"omitempty,oneof=off simple advanced"` // off|simple|advanced
}

func (dst *SettingSslInspection) UnmarshalJSON(b []byte) error {
	type Alias SettingSslInspection
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

// GetSettingSslInspection Experimental! This function is not yet stable and may change in the future.
func (c *client) GetSettingSslInspection(ctx context.Context, site string) (*SettingSslInspection, error) {
	s, f, err := c.GetSetting(ctx, site, SettingSslInspectionKey)
	if err != nil {
		return nil, err
	}
	if s.Key != SettingSslInspectionKey {
		return nil, fmt.Errorf("unexpected setting key received. Requested: %q, received: %q", SettingSslInspectionKey, s.Key)
	}
	resource, ok := f.(*SettingSslInspection)
	if !ok {
		return nil, fmt.Errorf("unexpected type for setting value. expected: *SettingSslInspection, received: %T", f)
	}
	return resource, nil
}

// UpdateSettingSslInspection Experimental! This function is not yet stable and may change in the future.
func (c *client) UpdateSettingSslInspection(ctx context.Context, site string, s *SettingSslInspection) (*SettingSslInspection, error) {
	s.Key = SettingSslInspectionKey
	result, err := c.SetSetting(ctx, site, SettingSslInspectionKey, s)
	if err != nil {
		return nil, err
	}
	updatedResource, ok := result.(*SettingSslInspection)
	if !ok {
		return nil, fmt.Errorf("unexpected type for setting value. expected: *SettingSslInspection, received: %T", result)
	}
	return updatedResource, nil
}
