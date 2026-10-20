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

// MarshalJSON sends nil ports and port_ranges as [] rather than null, for the same reason and with the
// same pointer-receiver scope as TrafficRoute.MarshalJSON.
func (dst *TrafficRouteDomains) MarshalJSON() ([]byte, error) {
	type Alias TrafficRouteDomains
	aux := Alias(*dst)
	aux.PortRanges = emptyIfNil(aux.PortRanges)
	aux.Ports = emptyIfNil(aux.Ports)
	return json.Marshal(&aux)
}

// MarshalJSON sends nil ports and port_ranges as [] rather than null, for the same reason and with the
// same pointer-receiver scope as TrafficRoute.MarshalJSON.
func (dst *TrafficRouteIPAddresses) MarshalJSON() ([]byte, error) {
	type Alias TrafficRouteIPAddresses
	aux := Alias(*dst)
	aux.PortRanges = emptyIfNil(aux.PortRanges)
	aux.Ports = emptyIfNil(aux.Ports)
	return json.Marshal(&aux)
}

// ListTrafficRoute returns every traffic route on the site.
func (c *client) ListTrafficRoute(ctx context.Context, site string) ([]TrafficRoute, error) {
	return c.listTrafficRoute(ctx, site)
}

// GetTrafficRoute returns the route with the given ID, or ErrNotFound if the site has none. It filters
// the list, because GET /trafficroutes/<id> is not a documented endpoint (some controllers answer 405).
func (c *client) GetTrafficRoute(ctx context.Context, site, id string) (*TrafficRoute, error) {
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

// DeleteTrafficRoute deletes the route with the given ID. A route that is already gone is an error:
// the controller answers 404.
func (c *client) DeleteTrafficRoute(ctx context.Context, site, id string) error {
	return c.deleteTrafficRoute(ctx, site, id)
}

// CreateTrafficRoute creates d on the site and returns the route as the controller stored it.
func (c *client) CreateTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	return c.createTrafficRoute(ctx, site, d)
}

// UpdateTrafficRoute replaces the route d.ID with d. The controller treats the PUT body as the complete
// object, so every field d leaves empty is cleared, as is any key TrafficRoute does not model.
func (c *client) UpdateTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	return c.updateTrafficRoute(ctx, site, d)
}
