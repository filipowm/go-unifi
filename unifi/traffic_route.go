package unifi

import (
	"context"
	"encoding/json"
)

// MarshalJSON sends every nil list as [] rather than null, so a full-replace PUT clears it on the
// controller. Like the other hand-written marshalers in this package it has a pointer receiver
// (matching the generated UnmarshalJSON), so the [] guarantee holds when marshalling a *TrafficRoute,
// which is what every client method does. A TrafficRoute marshalled by value, or held by value in
// another struct, goes through the default encoder and emits null for nil lists.
func (dst *TrafficRoute) MarshalJSON() ([]byte, error) {
	type Alias TrafficRoute
	aux := Alias(*dst)
	aux.Domains = emptyIfNil(aux.Domains)
	aux.IPAddresses = emptyIfNil(aux.IPAddresses)
	aux.IPRanges = emptyIfNil(aux.IPRanges)
	aux.Regions = emptyIfNil(aux.Regions)
	aux.TargetDevices = emptyIfNil(aux.TargetDevices)
	return json.Marshal(&aux)
}

func (dst *TrafficRouteDomains) MarshalJSON() ([]byte, error) {
	type Alias TrafficRouteDomains
	aux := Alias(*dst)
	aux.PortRanges = emptyIfNil(aux.PortRanges)
	aux.Ports = emptyIfNil(aux.Ports)
	return json.Marshal(&aux)
}

func (dst *TrafficRouteIPAddresses) MarshalJSON() ([]byte, error) {
	type Alias TrafficRouteIPAddresses
	aux := Alias(*dst)
	aux.PortRanges = emptyIfNil(aux.PortRanges)
	aux.Ports = emptyIfNil(aux.Ports)
	return json.Marshal(&aux)
}

func (c *client) ListTrafficRoute(ctx context.Context, site string) ([]TrafficRoute, error) {
	return c.listTrafficRoute(ctx, site)
}

func (c *client) GetTrafficRoute(ctx context.Context, site, id string) (*TrafficRoute, error) {
	// GET /trafficroutes/<id> is not a documented endpoint (some controllers answer 405), so filter the list.
	routes, err := c.listTrafficRoute(ctx, site)
	if err != nil {
		return nil, err
	}
	for i := range routes {
		if routes[i].ID == id {
			return &routes[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *client) DeleteTrafficRoute(ctx context.Context, site, id string) error {
	return c.deleteTrafficRoute(ctx, site, id)
}

func (c *client) CreateTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	return c.createTrafficRoute(ctx, site, d)
}

func (c *client) UpdateTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	return c.updateTrafficRoute(ctx, site, d)
}
