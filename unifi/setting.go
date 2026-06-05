package unifi

import (
	"context"
	"encoding/json"
	"fmt"
)

type Setting struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`
	Key    string `json:"key"`
}

// The settingFactories registry lives in setting_registry.go and is populated
// exclusively by the generated per-setting init() functions (registerSetting),
// so it can never drift from the generated setting catalog by hand.

func (s *Setting) newFields() (any, error) {
	factory, ok := settingFactories[s.Key]
	if !ok {
		return nil, fmt.Errorf("unexpected key %q", s.Key)
	}
	return factory(), nil
}

func (c *client) SetSetting(ctx context.Context, site, key string, reqBody any) (any, error) {
	var respBody struct {
		Meta Meta              `json:"meta"`
		Data []json.RawMessage `json:"data"`
	}
	err := c.Put(ctx, fmt.Sprintf("s/%s/set/setting/%s", site, key), reqBody, &respBody)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	var setting *Setting
	for _, d := range respBody.Data {
		err = json.Unmarshal(d, &setting)
		if err != nil {
			return nil, err
		}
		if setting.Key == key {
			raw = d
			break
		}
	}
	if setting == nil || setting.Key != key {
		return nil, ErrNotFound
	}
	fields, err := setting.newFields()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &fields)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

func (c *client) GetSetting(ctx context.Context, site, key string) (*Setting, any, error) {
	var respBody struct {
		Meta Meta              `json:"Meta"`
		Data []json.RawMessage `json:"data"`
	}

	err := c.Get(ctx, fmt.Sprintf("s/%s/get/setting", site), nil, &respBody)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get setting %s: %w", key, err)
	}

	var raw json.RawMessage
	var setting *Setting
	for _, d := range respBody.Data {
		err = json.Unmarshal(d, &setting)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to decode get setting %s: %w", key, err)
		}
		if setting.Key == key {
			raw = d
			break
		}
	}
	if setting == nil || setting.Key != key {
		return nil, nil, ErrNotFound
	}

	fields, err := setting.newFields()
	if err != nil {
		return nil, nil, err
	}

	err = json.Unmarshal(raw, &fields)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decode get setting fields %s: %w", key, err)
	}

	return setting, fields, nil
}
