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

type TrafficRoute struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Description       string                      `json:"description"`
	Domains           []TrafficRouteDomains       `json:"domains"`
	Enabled           bool                        `json:"enabled"`
	IPAddresses       []TrafficRouteIPAddresses   `json:"ip_addresses"`
	IPRanges          []TrafficRouteIPRanges      `json:"ip_ranges"`
	KillSwitchEnabled bool                        `json:"kill_switch_enabled"`
	MatchingTarget    string                      `json:"matching_target,omitempty" validate:"omitempty,oneof=INTERNET DOMAIN IP REGION"` // INTERNET|DOMAIN|IP|REGION
	NetworkID         string                      `json:"network_id,omitempty"`
	NextHop           string                      `json:"next_hop"`
	Regions           []string                    `json:"regions"`
	TargetDevices     []TrafficRouteTargetDevices `json:"target_devices"`
}

func (dst *TrafficRoute) UnmarshalJSON(b []byte) error {
	type Alias TrafficRoute
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

type TrafficRouteDomains struct {
	Domain     string                   `json:"domain,omitempty"`
	PortRanges []TrafficRoutePortRanges `json:"port_ranges"`
	Ports      []int                    `json:"ports"`
}

func (dst *TrafficRouteDomains) UnmarshalJSON(b []byte) error {
	type Alias TrafficRouteDomains
	aux := &struct {
		*Alias

		Ports []emptyStringInt `json:"ports"`
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	dst.Ports = make([]int, len(aux.Ports))
	for i, v := range aux.Ports {
		dst.Ports[i] = int(v)
	}

	return nil
}

type TrafficRouteIPAddresses struct {
	IPOrSubnet string                   `json:"ip_or_subnet,omitempty"`
	IPVersion  string                   `json:"ip_version,omitempty" validate:"omitempty,oneof=v4 v6"` // v4|v6
	PortRanges []TrafficRoutePortRanges `json:"port_ranges"`
	Ports      []int                    `json:"ports"`
}

func (dst *TrafficRouteIPAddresses) UnmarshalJSON(b []byte) error {
	type Alias TrafficRouteIPAddresses
	aux := &struct {
		*Alias

		Ports []emptyStringInt `json:"ports"`
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	dst.Ports = make([]int, len(aux.Ports))
	for i, v := range aux.Ports {
		dst.Ports[i] = int(v)
	}

	return nil
}

type TrafficRouteIPRanges struct {
	IPStart   string `json:"ip_start,omitempty"`
	IPStop    string `json:"ip_stop,omitempty"`
	IPVersion string `json:"ip_version,omitempty" validate:"omitempty,oneof=v4 v6"` // v4|v6
}

func (dst *TrafficRouteIPRanges) UnmarshalJSON(b []byte) error {
	type Alias TrafficRouteIPRanges
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

type TrafficRoutePortRanges struct {
	PortStart int `json:"port_start,omitempty"`
	PortStop  int `json:"port_stop,omitempty"`
}

func (dst *TrafficRoutePortRanges) UnmarshalJSON(b []byte) error {
	type Alias TrafficRoutePortRanges
	aux := &struct {
		*Alias

		PortStart emptyStringInt `json:"port_start"`
		PortStop  emptyStringInt `json:"port_stop"`
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	dst.PortStart = int(aux.PortStart)
	dst.PortStop = int(aux.PortStop)

	return nil
}

type TrafficRouteTargetDevices struct {
	ClientMAC string `json:"client_mac,omitempty" validate:"omitempty,mac"` // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	NetworkID string `json:"network_id,omitempty"`
	Type      string `json:"type,omitempty" validate:"omitempty,oneof=ALL_CLIENTS CLIENT NETWORK"` // ALL_CLIENTS|CLIENT|NETWORK
}

func (dst *TrafficRouteTargetDevices) UnmarshalJSON(b []byte) error {
	type Alias TrafficRouteTargetDevices
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

func (c *client) listTrafficRoute(ctx context.Context, site string) ([]TrafficRoute, error) {
	var respBody []TrafficRoute

	err := c.Get(ctx, fmt.Sprintf("%s/site/%s/trafficroutes", c.apiPaths.ApiV2Path, site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (c *client) getTrafficRoute(ctx context.Context, site, id string) (*TrafficRoute, error) {
	var respBody TrafficRoute

	err := c.Get(ctx, fmt.Sprintf("%s/site/%s/trafficroutes/%s", c.apiPaths.ApiV2Path, site, id), nil, &respBody)
	if err != nil {
		return nil, err
	}
	if respBody.ID == "" {
		return nil, ErrNotFound
	}
	return &respBody, nil
}

func (c *client) deleteTrafficRoute(ctx context.Context, site, id string) error {
	err := c.Delete(ctx, fmt.Sprintf("%s/site/%s/trafficroutes/%s", c.apiPaths.ApiV2Path, site, id), struct{}{}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) createTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	var respBody TrafficRoute

	err := c.Post(ctx, fmt.Sprintf("%s/site/%s/trafficroutes", c.apiPaths.ApiV2Path, site), d, &respBody)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *client) updateTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	var respBody TrafficRoute

	err := c.Put(ctx, fmt.Sprintf("%s/site/%s/trafficroutes/%s", c.apiPaths.ApiV2Path, site, d.ID), d, &respBody)
	if err != nil {
		return nil, err
	}
	return &respBody, nil
}
