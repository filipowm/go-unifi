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

const SettingSuperIdentityKey = "super_identity"

// Self-register this setting's fields factory so the settingFactories registry
// in setting_registry.go stays a 1:1 reflection of the generated catalog and
// can never drift from it by hand.
func init() { //nolint:gochecknoinits
	registerSetting(SettingSuperIdentityKey, func() any { return &SettingSuperIdentity{} })
}

type SettingSuperIdentity struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Key string `json:"key"`

	Hostname string `json:"hostname,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (dst *SettingSuperIdentity) UnmarshalJSON(b []byte) error {
	type Alias SettingSuperIdentity
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

// GetSettingSuperIdentity Experimental! This function is not yet stable and may change in the future.
func (c *client) GetSettingSuperIdentity(ctx context.Context, site string) (*SettingSuperIdentity, error) {
	s, f, err := c.GetSetting(ctx, site, SettingSuperIdentityKey)
	if err != nil {
		return nil, err
	}
	if s.Key != SettingSuperIdentityKey {
		return nil, fmt.Errorf("unexpected setting key received. Requested: %q, received: %q", SettingSuperIdentityKey, s.Key)
	}
	resource, ok := f.(*SettingSuperIdentity)
	if !ok {
		return nil, fmt.Errorf("unexpected type for setting value. expected: *SettingSuperIdentity, received: %T", f)
	}
	return resource, nil
}

// UpdateSettingSuperIdentity Experimental! This function is not yet stable and may change in the future.
func (c *client) UpdateSettingSuperIdentity(ctx context.Context, site string, s *SettingSuperIdentity) (*SettingSuperIdentity, error) {
	s.Key = SettingSuperIdentityKey
	result, err := c.SetSetting(ctx, site, SettingSuperIdentityKey, s)
	if err != nil {
		return nil, err
	}
	updatedResource, ok := result.(*SettingSuperIdentity)
	if !ok {
		return nil, fmt.Errorf("unexpected type for setting value. expected: *SettingSuperIdentity, received: %T", result)
	}
	return updatedResource, nil
}
