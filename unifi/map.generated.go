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

type Map struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Lat        string  `json:"lat,omitempty"`                                                         // ^([-]?[\d]+[.]?[\d]*([eE][-+]?[\d]+)?)$
	Lng        string  `json:"lng,omitempty"`                                                         // ^([-]?[\d]+[.]?[\d]*([eE][-+]?[\d]+)?)$
	MapTypeID  string  `json:"mapTypeId" validate:"omitempty,oneof=satellite roadmap hybrid terrain"` // satellite|roadmap|hybrid|terrain
	Name       string  `json:"name,omitempty"`
	OffsetLeft float64 `json:"offset_left,omitempty"`
	OffsetTop  float64 `json:"offset_top,omitempty"`
	Opacity    float64 `json:"opacity,omitempty"` // ^(0(\.[\d]{1,2})?|1)$|^$
	Selected   bool    `json:"selected"`
	Tilt       int     `json:"tilt,omitempty"`
	Type       string  `json:"type,omitempty" validate:"omitempty,oneof=designerMap imageMap googleMap"` // designerMap|imageMap|googleMap
	Unit       string  `json:"unit,omitempty" validate:"omitempty,oneof=m f"`                            // m|f
	Upp        float64 `json:"upp,omitempty"`
	Zoom       int     `json:"zoom,omitempty"`
}

func (dst *Map) UnmarshalJSON(b []byte) error {
	type Alias Map
	aux := &struct {
		Tilt emptyStringInt `json:"tilt"`
		Zoom emptyStringInt `json:"zoom"`

		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	dst.Tilt = int(aux.Tilt)
	dst.Zoom = int(aux.Zoom)

	return nil
}

func (c *client) listMap(ctx context.Context, site string) ([]Map, error) {
	var respBody struct {
		Meta Meta  `json:"meta"`
		Data []Map `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/rest/map", site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody.Data, nil
}

func (c *client) getMap(ctx context.Context, site, id string) (*Map, error) {
	var respBody struct {
		Meta Meta  `json:"meta"`
		Data []Map `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/rest/map/%s", site, id), nil, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *client) deleteMap(ctx context.Context, site, id string) error {
	err := c.Delete(ctx, fmt.Sprintf("s/%s/rest/map/%s", site, id), struct{}{}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) createMap(ctx context.Context, site string, d *Map) (*Map, error) {
	var respBody struct {
		Meta Meta  `json:"meta"`
		Data []Map `json:"data"`
	}

	err := c.Post(ctx, fmt.Sprintf("s/%s/rest/map", site), d, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	new := respBody.Data[0]

	return &new, nil
}

func (c *client) updateMap(ctx context.Context, site string, d *Map) (*Map, error) {
	var respBody struct {
		Meta Meta  `json:"meta"`
		Data []Map `json:"data"`
	}

	err := c.Put(ctx, fmt.Sprintf("s/%s/rest/map/%s", site, d.ID), d, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, ErrNotFound
	}

	new := respBody.Data[0]

	return &new, nil
}
