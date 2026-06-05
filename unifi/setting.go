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

// settingFactories maps a setting key to a constructor for its concrete fields type.
//
// Each entry MUST be a closure returning a fresh pointer: the returned value is
// passed to json.Unmarshal, so sharing a single instance across calls would alias
// decoded state between callers.
var settingFactories = map[string]func() any{
	SettingAutoSpeedtestKey:       func() any { return &SettingAutoSpeedtest{} },
	SettingBaresipKey:             func() any { return &SettingBaresip{} },
	SettingBroadcastKey:           func() any { return &SettingBroadcast{} },
	SettingConnectivityKey:        func() any { return &SettingConnectivity{} },
	SettingCountryKey:             func() any { return &SettingCountry{} },
	SettingDashboardKey:           func() any { return &SettingDashboard{} },
	SettingDohKey:                 func() any { return &SettingDoh{} },
	SettingDpiKey:                 func() any { return &SettingDpi{} },
	SettingElementAdoptKey:        func() any { return &SettingElementAdopt{} },
	SettingEtherLightingKey:       func() any { return &SettingEtherLighting{} },
	SettingEvaluationScoreKey:     func() any { return &SettingEvaluationScore{} },
	SettingGlobalApKey:            func() any { return &SettingGlobalAp{} },
	SettingGlobalNatKey:           func() any { return &SettingGlobalNat{} },
	SettingGlobalSwitchKey:        func() any { return &SettingGlobalSwitch{} },
	SettingGuestAccessKey:         func() any { return &SettingGuestAccess{} },
	SettingIpsKey:                 func() any { return &SettingIps{} },
	SettingLcmKey:                 func() any { return &SettingLcm{} },
	SettingLocaleKey:              func() any { return &SettingLocale{} },
	SettingMagicSiteToSiteVpnKey:  func() any { return &SettingMagicSiteToSiteVpn{} },
	SettingMgmtKey:                func() any { return &SettingMgmt{} },
	SettingNetflowKey:             func() any { return &SettingNetflow{} },
	SettingNetworkOptimizationKey: func() any { return &SettingNetworkOptimization{} },
	SettingNtpKey:                 func() any { return &SettingNtp{} },
	SettingPortaKey:               func() any { return &SettingPorta{} },
	SettingRadioAiKey:             func() any { return &SettingRadioAi{} },
	SettingRadiusKey:              func() any { return &SettingRadius{} },
	SettingRsyslogdKey:            func() any { return &SettingRsyslogd{} },
	SettingSnmpKey:                func() any { return &SettingSnmp{} },
	SettingSslInspectionKey:       func() any { return &SettingSslInspection{} },
	SettingSuperCloudaccessKey:    func() any { return &SettingSuperCloudaccess{} },
	SettingSuperEventsKey:         func() any { return &SettingSuperEvents{} },
	SettingSuperFwupdateKey:       func() any { return &SettingSuperFwupdate{} },
	SettingSuperIdentityKey:       func() any { return &SettingSuperIdentity{} },
	SettingSuperMailKey:           func() any { return &SettingSuperMail{} },
	SettingSuperMgmtKey:           func() any { return &SettingSuperMgmt{} },
	SettingSuperSdnKey:            func() any { return &SettingSuperSdn{} },
	SettingSuperSmtpKey:           func() any { return &SettingSuperSmtp{} },
	SettingTeleportKey:            func() any { return &SettingTeleport{} },
	SettingUsgKey:                 func() any { return &SettingUsg{} },
	SettingUswKey:                 func() any { return &SettingUsw{} },
}

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
