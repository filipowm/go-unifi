package official

import (
	"context"
	"fmt"
)

// Info is the controller application info from GET /v1/info. ApplicationVersion
// feeds the capability gate (the official API exists only from 10.1.78).
type Info struct {
	ApplicationVersion string `json:"applicationVersion"`
}

// Get returns the controller application info.
func (c infoClient) Get(ctx context.Context) (*Info, error) {
	if err := c.check(ctx); err != nil {
		return nil, err
	}
	var info Info
	if err := c.doer.Get(ctx, c.path("/info"), nil, &info); err != nil {
		return nil, fmt.Errorf("failed getting application info: %w", err)
	}
	return &info, nil
}
