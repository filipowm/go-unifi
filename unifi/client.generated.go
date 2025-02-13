// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

type client interface {

	/* custom method signatures */

	/* client methods generated based on resource generation */
	/* client methods for Account API */

	// GetAccount returns Account resource by its ID
	GetAccount(ctx context.Context, site, id string) (*Account, error)
	// UpdateAccount updates Account resource by its ID
	UpdateAccount(ctx context.Context, site string, d *Account) (*Account, error)
	// ListAccount returns list of Account resources
	ListAccount(ctx context.Context, site string) ([]Account, error)
	// DeleteAccount deletes Account resource by its ID
	DeleteAccount(ctx context.Context, site, id string) error
	// CreateAccount creates new Account resource
	CreateAccount(ctx context.Context, site string, d *Account) (*Account, error)

	/* client methods for BroadcastGroup API */

	// GetBroadcastGroup returns BroadcastGroup resource by its ID
	GetBroadcastGroup(ctx context.Context, site, id string) (*BroadcastGroup, error)
	// UpdateBroadcastGroup updates BroadcastGroup resource by its ID
	UpdateBroadcastGroup(ctx context.Context, site string, d *BroadcastGroup) (*BroadcastGroup, error)
	// ListBroadcastGroup returns list of BroadcastGroup resources
	ListBroadcastGroup(ctx context.Context, site string) ([]BroadcastGroup, error)
	// DeleteBroadcastGroup deletes BroadcastGroup resource by its ID
	DeleteBroadcastGroup(ctx context.Context, site, id string) error
	// CreateBroadcastGroup creates new BroadcastGroup resource
	CreateBroadcastGroup(ctx context.Context, site string, d *BroadcastGroup) (*BroadcastGroup, error)

	/* client methods for ChannelPlan API */

	// GetChannelPlan returns ChannelPlan resource by its ID
	GetChannelPlan(ctx context.Context, site, id string) (*ChannelPlan, error)
	// UpdateChannelPlan updates ChannelPlan resource by its ID
	UpdateChannelPlan(ctx context.Context, site string, d *ChannelPlan) (*ChannelPlan, error)
	// ListChannelPlan returns list of ChannelPlan resources
	ListChannelPlan(ctx context.Context, site string) ([]ChannelPlan, error)
	// DeleteChannelPlan deletes ChannelPlan resource by its ID
	DeleteChannelPlan(ctx context.Context, site, id string) error
	// CreateChannelPlan creates new ChannelPlan resource
	CreateChannelPlan(ctx context.Context, site string, d *ChannelPlan) (*ChannelPlan, error)

	/* client methods for Dashboard API */

	// GetDashboard returns Dashboard resource by its ID
	GetDashboard(ctx context.Context, site, id string) (*Dashboard, error)
	// UpdateDashboard updates Dashboard resource by its ID
	UpdateDashboard(ctx context.Context, site string, d *Dashboard) (*Dashboard, error)
	// ListDashboard returns list of Dashboard resources
	ListDashboard(ctx context.Context, site string) ([]Dashboard, error)
	// DeleteDashboard deletes Dashboard resource by its ID
	DeleteDashboard(ctx context.Context, site, id string) error
	// CreateDashboard creates new Dashboard resource
	CreateDashboard(ctx context.Context, site string, d *Dashboard) (*Dashboard, error)

	/* client methods for Device API */

	// GetDevice returns Device resource by its ID
	GetDevice(ctx context.Context, site, id string) (*Device, error)
	// UpdateDevice updates Device resource by its ID
	UpdateDevice(ctx context.Context, site string, d *Device) (*Device, error)
	// ListDevice returns list of Device resources
	ListDevice(ctx context.Context, site string) ([]Device, error)
	// DeleteDevice deletes Device resource by its ID
	DeleteDevice(ctx context.Context, site, id string) error
	// CreateDevice creates new Device resource
	CreateDevice(ctx context.Context, site string, d *Device) (*Device, error)

	/* client methods for DHCPOption API */

	// GetDHCPOption returns DHCPOption resource by its ID
	GetDHCPOption(ctx context.Context, site, id string) (*DHCPOption, error)
	// UpdateDHCPOption updates DHCPOption resource by its ID
	UpdateDHCPOption(ctx context.Context, site string, d *DHCPOption) (*DHCPOption, error)
	// ListDHCPOption returns list of DHCPOption resources
	ListDHCPOption(ctx context.Context, site string) ([]DHCPOption, error)
	// DeleteDHCPOption deletes DHCPOption resource by its ID
	DeleteDHCPOption(ctx context.Context, site, id string) error
	// CreateDHCPOption creates new DHCPOption resource
	CreateDHCPOption(ctx context.Context, site string, d *DHCPOption) (*DHCPOption, error)

	/* client methods for DpiApp API */

	// GetDpiApp returns DpiApp resource by its ID
	GetDpiApp(ctx context.Context, site, id string) (*DpiApp, error)
	// UpdateDpiApp updates DpiApp resource by its ID
	UpdateDpiApp(ctx context.Context, site string, d *DpiApp) (*DpiApp, error)
	// ListDpiApp returns list of DpiApp resources
	ListDpiApp(ctx context.Context, site string) ([]DpiApp, error)
	// DeleteDpiApp deletes DpiApp resource by its ID
	DeleteDpiApp(ctx context.Context, site, id string) error
	// CreateDpiApp creates new DpiApp resource
	CreateDpiApp(ctx context.Context, site string, d *DpiApp) (*DpiApp, error)

	/* client methods for DpiGroup API */

	// GetDpiGroup returns DpiGroup resource by its ID
	GetDpiGroup(ctx context.Context, site, id string) (*DpiGroup, error)
	// UpdateDpiGroup updates DpiGroup resource by its ID
	UpdateDpiGroup(ctx context.Context, site string, d *DpiGroup) (*DpiGroup, error)
	// ListDpiGroup returns list of DpiGroup resources
	ListDpiGroup(ctx context.Context, site string) ([]DpiGroup, error)
	// DeleteDpiGroup deletes DpiGroup resource by its ID
	DeleteDpiGroup(ctx context.Context, site, id string) error
	// CreateDpiGroup creates new DpiGroup resource
	CreateDpiGroup(ctx context.Context, site string, d *DpiGroup) (*DpiGroup, error)

	/* client methods for DynamicDNS API */

	// GetDynamicDNS returns DynamicDNS resource by its ID
	GetDynamicDNS(ctx context.Context, site, id string) (*DynamicDNS, error)
	// UpdateDynamicDNS updates DynamicDNS resource by its ID
	UpdateDynamicDNS(ctx context.Context, site string, d *DynamicDNS) (*DynamicDNS, error)
	// ListDynamicDNS returns list of DynamicDNS resources
	ListDynamicDNS(ctx context.Context, site string) ([]DynamicDNS, error)
	// DeleteDynamicDNS deletes DynamicDNS resource by its ID
	DeleteDynamicDNS(ctx context.Context, site, id string) error
	// CreateDynamicDNS creates new DynamicDNS resource
	CreateDynamicDNS(ctx context.Context, site string, d *DynamicDNS) (*DynamicDNS, error)

	/* client methods for FirewallGroup API */

	// GetFirewallGroup returns FirewallGroup resource by its ID
	GetFirewallGroup(ctx context.Context, site, id string) (*FirewallGroup, error)
	// UpdateFirewallGroup updates FirewallGroup resource by its ID
	UpdateFirewallGroup(ctx context.Context, site string, d *FirewallGroup) (*FirewallGroup, error)
	// ListFirewallGroup returns list of FirewallGroup resources
	ListFirewallGroup(ctx context.Context, site string) ([]FirewallGroup, error)
	// DeleteFirewallGroup deletes FirewallGroup resource by its ID
	DeleteFirewallGroup(ctx context.Context, site, id string) error
	// CreateFirewallGroup creates new FirewallGroup resource
	CreateFirewallGroup(ctx context.Context, site string, d *FirewallGroup) (*FirewallGroup, error)

	/* client methods for FirewallRule API */

	// GetFirewallRule returns FirewallRule resource by its ID
	GetFirewallRule(ctx context.Context, site, id string) (*FirewallRule, error)
	// UpdateFirewallRule updates FirewallRule resource by its ID
	UpdateFirewallRule(ctx context.Context, site string, d *FirewallRule) (*FirewallRule, error)
	// ListFirewallRule returns list of FirewallRule resources
	ListFirewallRule(ctx context.Context, site string) ([]FirewallRule, error)
	// DeleteFirewallRule deletes FirewallRule resource by its ID
	DeleteFirewallRule(ctx context.Context, site, id string) error
	// CreateFirewallRule creates new FirewallRule resource
	CreateFirewallRule(ctx context.Context, site string, d *FirewallRule) (*FirewallRule, error)

	/* client methods for HeatMap API */

	// GetHeatMap returns HeatMap resource by its ID
	GetHeatMap(ctx context.Context, site, id string) (*HeatMap, error)
	// UpdateHeatMap updates HeatMap resource by its ID
	UpdateHeatMap(ctx context.Context, site string, d *HeatMap) (*HeatMap, error)
	// ListHeatMap returns list of HeatMap resources
	ListHeatMap(ctx context.Context, site string) ([]HeatMap, error)
	// DeleteHeatMap deletes HeatMap resource by its ID
	DeleteHeatMap(ctx context.Context, site, id string) error
	// CreateHeatMap creates new HeatMap resource
	CreateHeatMap(ctx context.Context, site string, d *HeatMap) (*HeatMap, error)

	/* client methods for HeatMapPoint API */

	// GetHeatMapPoint returns HeatMapPoint resource by its ID
	GetHeatMapPoint(ctx context.Context, site, id string) (*HeatMapPoint, error)
	// UpdateHeatMapPoint updates HeatMapPoint resource by its ID
	UpdateHeatMapPoint(ctx context.Context, site string, d *HeatMapPoint) (*HeatMapPoint, error)
	// ListHeatMapPoint returns list of HeatMapPoint resources
	ListHeatMapPoint(ctx context.Context, site string) ([]HeatMapPoint, error)
	// DeleteHeatMapPoint deletes HeatMapPoint resource by its ID
	DeleteHeatMapPoint(ctx context.Context, site, id string) error
	// CreateHeatMapPoint creates new HeatMapPoint resource
	CreateHeatMapPoint(ctx context.Context, site string, d *HeatMapPoint) (*HeatMapPoint, error)

	/* client methods for Hotspot2Conf API */

	// GetHotspot2Conf returns Hotspot2Conf resource by its ID
	GetHotspot2Conf(ctx context.Context, site, id string) (*Hotspot2Conf, error)
	// UpdateHotspot2Conf updates Hotspot2Conf resource by its ID
	UpdateHotspot2Conf(ctx context.Context, site string, d *Hotspot2Conf) (*Hotspot2Conf, error)
	// ListHotspot2Conf returns list of Hotspot2Conf resources
	ListHotspot2Conf(ctx context.Context, site string) ([]Hotspot2Conf, error)
	// DeleteHotspot2Conf deletes Hotspot2Conf resource by its ID
	DeleteHotspot2Conf(ctx context.Context, site, id string) error
	// CreateHotspot2Conf creates new Hotspot2Conf resource
	CreateHotspot2Conf(ctx context.Context, site string, d *Hotspot2Conf) (*Hotspot2Conf, error)

	/* client methods for HotspotOp API */

	// GetHotspotOp returns HotspotOp resource by its ID
	GetHotspotOp(ctx context.Context, site, id string) (*HotspotOp, error)
	// UpdateHotspotOp updates HotspotOp resource by its ID
	UpdateHotspotOp(ctx context.Context, site string, d *HotspotOp) (*HotspotOp, error)
	// ListHotspotOp returns list of HotspotOp resources
	ListHotspotOp(ctx context.Context, site string) ([]HotspotOp, error)
	// DeleteHotspotOp deletes HotspotOp resource by its ID
	DeleteHotspotOp(ctx context.Context, site, id string) error
	// CreateHotspotOp creates new HotspotOp resource
	CreateHotspotOp(ctx context.Context, site string, d *HotspotOp) (*HotspotOp, error)

	/* client methods for HotspotPackage API */

	// GetHotspotPackage returns HotspotPackage resource by its ID
	GetHotspotPackage(ctx context.Context, site, id string) (*HotspotPackage, error)
	// UpdateHotspotPackage updates HotspotPackage resource by its ID
	UpdateHotspotPackage(ctx context.Context, site string, d *HotspotPackage) (*HotspotPackage, error)
	// ListHotspotPackage returns list of HotspotPackage resources
	ListHotspotPackage(ctx context.Context, site string) ([]HotspotPackage, error)
	// DeleteHotspotPackage deletes HotspotPackage resource by its ID
	DeleteHotspotPackage(ctx context.Context, site, id string) error
	// CreateHotspotPackage creates new HotspotPackage resource
	CreateHotspotPackage(ctx context.Context, site string, d *HotspotPackage) (*HotspotPackage, error)

	/* client methods for Map API */

	// GetMap returns Map resource by its ID
	GetMap(ctx context.Context, site, id string) (*Map, error)
	// UpdateMap updates Map resource by its ID
	UpdateMap(ctx context.Context, site string, d *Map) (*Map, error)
	// ListMap returns list of Map resources
	ListMap(ctx context.Context, site string) ([]Map, error)
	// DeleteMap deletes Map resource by its ID
	DeleteMap(ctx context.Context, site, id string) error
	// CreateMap creates new Map resource
	CreateMap(ctx context.Context, site string, d *Map) (*Map, error)

	/* client methods for MediaFile API */

	// GetMediaFile returns MediaFile resource by its ID
	GetMediaFile(ctx context.Context, site, id string) (*MediaFile, error)
	// UpdateMediaFile updates MediaFile resource by its ID
	UpdateMediaFile(ctx context.Context, site string, d *MediaFile) (*MediaFile, error)
	// ListMediaFile returns list of MediaFile resources
	ListMediaFile(ctx context.Context, site string) ([]MediaFile, error)
	// DeleteMediaFile deletes MediaFile resource by its ID
	DeleteMediaFile(ctx context.Context, site, id string) error
	// CreateMediaFile creates new MediaFile resource
	CreateMediaFile(ctx context.Context, site string, d *MediaFile) (*MediaFile, error)

	/* client methods for Network API */

	// GetNetwork returns Network resource by its ID
	GetNetwork(ctx context.Context, site, id string) (*Network, error)
	// UpdateNetwork updates Network resource by its ID
	UpdateNetwork(ctx context.Context, site string, d *Network) (*Network, error)
	// ListNetwork returns list of Network resources
	ListNetwork(ctx context.Context, site string) ([]Network, error)
	// DeleteNetwork deletes Network resource by its ID
	DeleteNetwork(ctx context.Context, site, id string) error
	// CreateNetwork creates new Network resource
	CreateNetwork(ctx context.Context, site string, d *Network) (*Network, error)

	/* client methods for PortProfile API */

	// GetPortProfile returns PortProfile resource by its ID
	GetPortProfile(ctx context.Context, site, id string) (*PortProfile, error)
	// UpdatePortProfile updates PortProfile resource by its ID
	UpdatePortProfile(ctx context.Context, site string, d *PortProfile) (*PortProfile, error)
	// ListPortProfile returns list of PortProfile resources
	ListPortProfile(ctx context.Context, site string) ([]PortProfile, error)
	// DeletePortProfile deletes PortProfile resource by its ID
	DeletePortProfile(ctx context.Context, site, id string) error
	// CreatePortProfile creates new PortProfile resource
	CreatePortProfile(ctx context.Context, site string, d *PortProfile) (*PortProfile, error)

	/* client methods for PortForward API */

	// GetPortForward returns PortForward resource by its ID
	GetPortForward(ctx context.Context, site, id string) (*PortForward, error)
	// UpdatePortForward updates PortForward resource by its ID
	UpdatePortForward(ctx context.Context, site string, d *PortForward) (*PortForward, error)
	// ListPortForward returns list of PortForward resources
	ListPortForward(ctx context.Context, site string) ([]PortForward, error)
	// DeletePortForward deletes PortForward resource by its ID
	DeletePortForward(ctx context.Context, site, id string) error
	// CreatePortForward creates new PortForward resource
	CreatePortForward(ctx context.Context, site string, d *PortForward) (*PortForward, error)

	/* client methods for RADIUSProfile API */

	// GetRADIUSProfile returns RADIUSProfile resource by its ID
	GetRADIUSProfile(ctx context.Context, site, id string) (*RADIUSProfile, error)
	// UpdateRADIUSProfile updates RADIUSProfile resource by its ID
	UpdateRADIUSProfile(ctx context.Context, site string, d *RADIUSProfile) (*RADIUSProfile, error)
	// ListRADIUSProfile returns list of RADIUSProfile resources
	ListRADIUSProfile(ctx context.Context, site string) ([]RADIUSProfile, error)
	// DeleteRADIUSProfile deletes RADIUSProfile resource by its ID
	DeleteRADIUSProfile(ctx context.Context, site, id string) error
	// CreateRADIUSProfile creates new RADIUSProfile resource
	CreateRADIUSProfile(ctx context.Context, site string, d *RADIUSProfile) (*RADIUSProfile, error)

	/* client methods for Routing API */

	// GetRouting returns Routing resource by its ID
	GetRouting(ctx context.Context, site, id string) (*Routing, error)
	// UpdateRouting updates Routing resource by its ID
	UpdateRouting(ctx context.Context, site string, d *Routing) (*Routing, error)
	// ListRouting returns list of Routing resources
	ListRouting(ctx context.Context, site string) ([]Routing, error)
	// DeleteRouting deletes Routing resource by its ID
	DeleteRouting(ctx context.Context, site, id string) error
	// CreateRouting creates new Routing resource
	CreateRouting(ctx context.Context, site string, d *Routing) (*Routing, error)

	/* client methods for ScheduleTask API */

	// GetScheduleTask returns ScheduleTask resource by its ID
	GetScheduleTask(ctx context.Context, site, id string) (*ScheduleTask, error)
	// UpdateScheduleTask updates ScheduleTask resource by its ID
	UpdateScheduleTask(ctx context.Context, site string, d *ScheduleTask) (*ScheduleTask, error)
	// ListScheduleTask returns list of ScheduleTask resources
	ListScheduleTask(ctx context.Context, site string) ([]ScheduleTask, error)
	// DeleteScheduleTask deletes ScheduleTask resource by its ID
	DeleteScheduleTask(ctx context.Context, site, id string) error
	// CreateScheduleTask creates new ScheduleTask resource
	CreateScheduleTask(ctx context.Context, site string, d *ScheduleTask) (*ScheduleTask, error)

	/* client methods for SettingAutoSpeedtest API */

	// GetSettingAutoSpeedtest returns SettingAutoSpeedtest resource
	GetSettingAutoSpeedtest(ctx context.Context, site string) (*SettingAutoSpeedtest, error)
	// UpdateSettingAutoSpeedtest updates SettingAutoSpeedtest resource
	UpdateSettingAutoSpeedtest(ctx context.Context, site string, d *SettingAutoSpeedtest) (*SettingAutoSpeedtest, error)
	/* client methods for SettingBaresip API */

	// GetSettingBaresip returns SettingBaresip resource
	GetSettingBaresip(ctx context.Context, site string) (*SettingBaresip, error)
	// UpdateSettingBaresip updates SettingBaresip resource
	UpdateSettingBaresip(ctx context.Context, site string, d *SettingBaresip) (*SettingBaresip, error)
	/* client methods for SettingBroadcast API */

	// GetSettingBroadcast returns SettingBroadcast resource
	GetSettingBroadcast(ctx context.Context, site string) (*SettingBroadcast, error)
	// UpdateSettingBroadcast updates SettingBroadcast resource
	UpdateSettingBroadcast(ctx context.Context, site string, d *SettingBroadcast) (*SettingBroadcast, error)
	/* client methods for SettingConnectivity API */

	// GetSettingConnectivity returns SettingConnectivity resource
	GetSettingConnectivity(ctx context.Context, site string) (*SettingConnectivity, error)
	// UpdateSettingConnectivity updates SettingConnectivity resource
	UpdateSettingConnectivity(ctx context.Context, site string, d *SettingConnectivity) (*SettingConnectivity, error)
	/* client methods for SettingCountry API */

	// GetSettingCountry returns SettingCountry resource
	GetSettingCountry(ctx context.Context, site string) (*SettingCountry, error)
	// UpdateSettingCountry updates SettingCountry resource
	UpdateSettingCountry(ctx context.Context, site string, d *SettingCountry) (*SettingCountry, error)
	/* client methods for SettingDashboard API */

	// GetSettingDashboard returns SettingDashboard resource
	GetSettingDashboard(ctx context.Context, site string) (*SettingDashboard, error)
	// UpdateSettingDashboard updates SettingDashboard resource
	UpdateSettingDashboard(ctx context.Context, site string, d *SettingDashboard) (*SettingDashboard, error)
	/* client methods for SettingDoh API */

	// GetSettingDoh returns SettingDoh resource
	GetSettingDoh(ctx context.Context, site string) (*SettingDoh, error)
	// UpdateSettingDoh updates SettingDoh resource
	UpdateSettingDoh(ctx context.Context, site string, d *SettingDoh) (*SettingDoh, error)
	/* client methods for SettingDpi API */

	// GetSettingDpi returns SettingDpi resource
	GetSettingDpi(ctx context.Context, site string) (*SettingDpi, error)
	// UpdateSettingDpi updates SettingDpi resource
	UpdateSettingDpi(ctx context.Context, site string, d *SettingDpi) (*SettingDpi, error)
	/* client methods for SettingElementAdopt API */

	// GetSettingElementAdopt returns SettingElementAdopt resource
	GetSettingElementAdopt(ctx context.Context, site string) (*SettingElementAdopt, error)
	// UpdateSettingElementAdopt updates SettingElementAdopt resource
	UpdateSettingElementAdopt(ctx context.Context, site string, d *SettingElementAdopt) (*SettingElementAdopt, error)
	/* client methods for SettingEtherLighting API */

	// GetSettingEtherLighting returns SettingEtherLighting resource
	GetSettingEtherLighting(ctx context.Context, site string) (*SettingEtherLighting, error)
	// UpdateSettingEtherLighting updates SettingEtherLighting resource
	UpdateSettingEtherLighting(ctx context.Context, site string, d *SettingEtherLighting) (*SettingEtherLighting, error)
	/* client methods for SettingEvaluationScore API */

	// GetSettingEvaluationScore returns SettingEvaluationScore resource
	GetSettingEvaluationScore(ctx context.Context, site string) (*SettingEvaluationScore, error)
	// UpdateSettingEvaluationScore updates SettingEvaluationScore resource
	UpdateSettingEvaluationScore(ctx context.Context, site string, d *SettingEvaluationScore) (*SettingEvaluationScore, error)
	/* client methods for SettingGlobalAp API */

	// GetSettingGlobalAp returns SettingGlobalAp resource
	GetSettingGlobalAp(ctx context.Context, site string) (*SettingGlobalAp, error)
	// UpdateSettingGlobalAp updates SettingGlobalAp resource
	UpdateSettingGlobalAp(ctx context.Context, site string, d *SettingGlobalAp) (*SettingGlobalAp, error)
	/* client methods for SettingGlobalNat API */

	// GetSettingGlobalNat returns SettingGlobalNat resource
	GetSettingGlobalNat(ctx context.Context, site string) (*SettingGlobalNat, error)
	// UpdateSettingGlobalNat updates SettingGlobalNat resource
	UpdateSettingGlobalNat(ctx context.Context, site string, d *SettingGlobalNat) (*SettingGlobalNat, error)
	/* client methods for SettingGlobalSwitch API */

	// GetSettingGlobalSwitch returns SettingGlobalSwitch resource
	GetSettingGlobalSwitch(ctx context.Context, site string) (*SettingGlobalSwitch, error)
	// UpdateSettingGlobalSwitch updates SettingGlobalSwitch resource
	UpdateSettingGlobalSwitch(ctx context.Context, site string, d *SettingGlobalSwitch) (*SettingGlobalSwitch, error)
	/* client methods for SettingGuestAccess API */

	// GetSettingGuestAccess returns SettingGuestAccess resource
	GetSettingGuestAccess(ctx context.Context, site string) (*SettingGuestAccess, error)
	// UpdateSettingGuestAccess updates SettingGuestAccess resource
	UpdateSettingGuestAccess(ctx context.Context, site string, d *SettingGuestAccess) (*SettingGuestAccess, error)
	/* client methods for SettingIps API */

	// GetSettingIps returns SettingIps resource
	GetSettingIps(ctx context.Context, site string) (*SettingIps, error)
	// UpdateSettingIps updates SettingIps resource
	UpdateSettingIps(ctx context.Context, site string, d *SettingIps) (*SettingIps, error)
	/* client methods for SettingLcm API */

	// GetSettingLcm returns SettingLcm resource
	GetSettingLcm(ctx context.Context, site string) (*SettingLcm, error)
	// UpdateSettingLcm updates SettingLcm resource
	UpdateSettingLcm(ctx context.Context, site string, d *SettingLcm) (*SettingLcm, error)
	/* client methods for SettingLocale API */

	// GetSettingLocale returns SettingLocale resource
	GetSettingLocale(ctx context.Context, site string) (*SettingLocale, error)
	// UpdateSettingLocale updates SettingLocale resource
	UpdateSettingLocale(ctx context.Context, site string, d *SettingLocale) (*SettingLocale, error)
	/* client methods for SettingMagicSiteToSiteVpn API */

	// GetSettingMagicSiteToSiteVpn returns SettingMagicSiteToSiteVpn resource
	GetSettingMagicSiteToSiteVpn(ctx context.Context, site string) (*SettingMagicSiteToSiteVpn, error)
	// UpdateSettingMagicSiteToSiteVpn updates SettingMagicSiteToSiteVpn resource
	UpdateSettingMagicSiteToSiteVpn(ctx context.Context, site string, d *SettingMagicSiteToSiteVpn) (*SettingMagicSiteToSiteVpn, error)
	/* client methods for SettingMgmt API */

	// GetSettingMgmt returns SettingMgmt resource
	GetSettingMgmt(ctx context.Context, site string) (*SettingMgmt, error)
	// UpdateSettingMgmt updates SettingMgmt resource
	UpdateSettingMgmt(ctx context.Context, site string, d *SettingMgmt) (*SettingMgmt, error)
	/* client methods for SettingNetflow API */

	// GetSettingNetflow returns SettingNetflow resource
	GetSettingNetflow(ctx context.Context, site string) (*SettingNetflow, error)
	// UpdateSettingNetflow updates SettingNetflow resource
	UpdateSettingNetflow(ctx context.Context, site string, d *SettingNetflow) (*SettingNetflow, error)
	/* client methods for SettingNetworkOptimization API */

	// GetSettingNetworkOptimization returns SettingNetworkOptimization resource
	GetSettingNetworkOptimization(ctx context.Context, site string) (*SettingNetworkOptimization, error)
	// UpdateSettingNetworkOptimization updates SettingNetworkOptimization resource
	UpdateSettingNetworkOptimization(ctx context.Context, site string, d *SettingNetworkOptimization) (*SettingNetworkOptimization, error)
	/* client methods for SettingNtp API */

	// GetSettingNtp returns SettingNtp resource
	GetSettingNtp(ctx context.Context, site string) (*SettingNtp, error)
	// UpdateSettingNtp updates SettingNtp resource
	UpdateSettingNtp(ctx context.Context, site string, d *SettingNtp) (*SettingNtp, error)
	/* client methods for SettingPorta API */

	// GetSettingPorta returns SettingPorta resource
	GetSettingPorta(ctx context.Context, site string) (*SettingPorta, error)
	// UpdateSettingPorta updates SettingPorta resource
	UpdateSettingPorta(ctx context.Context, site string, d *SettingPorta) (*SettingPorta, error)
	/* client methods for SettingRadioAi API */

	// GetSettingRadioAi returns SettingRadioAi resource
	GetSettingRadioAi(ctx context.Context, site string) (*SettingRadioAi, error)
	// UpdateSettingRadioAi updates SettingRadioAi resource
	UpdateSettingRadioAi(ctx context.Context, site string, d *SettingRadioAi) (*SettingRadioAi, error)
	/* client methods for SettingRadius API */

	// GetSettingRadius returns SettingRadius resource
	GetSettingRadius(ctx context.Context, site string) (*SettingRadius, error)
	// UpdateSettingRadius updates SettingRadius resource
	UpdateSettingRadius(ctx context.Context, site string, d *SettingRadius) (*SettingRadius, error)
	/* client methods for SettingRsyslogd API */

	// GetSettingRsyslogd returns SettingRsyslogd resource
	GetSettingRsyslogd(ctx context.Context, site string) (*SettingRsyslogd, error)
	// UpdateSettingRsyslogd updates SettingRsyslogd resource
	UpdateSettingRsyslogd(ctx context.Context, site string, d *SettingRsyslogd) (*SettingRsyslogd, error)
	/* client methods for SettingSnmp API */

	// GetSettingSnmp returns SettingSnmp resource
	GetSettingSnmp(ctx context.Context, site string) (*SettingSnmp, error)
	// UpdateSettingSnmp updates SettingSnmp resource
	UpdateSettingSnmp(ctx context.Context, site string, d *SettingSnmp) (*SettingSnmp, error)
	/* client methods for SettingSslInspection API */

	// GetSettingSslInspection returns SettingSslInspection resource
	GetSettingSslInspection(ctx context.Context, site string) (*SettingSslInspection, error)
	// UpdateSettingSslInspection updates SettingSslInspection resource
	UpdateSettingSslInspection(ctx context.Context, site string, d *SettingSslInspection) (*SettingSslInspection, error)
	/* client methods for SettingSuperCloudaccess API */

	// GetSettingSuperCloudaccess returns SettingSuperCloudaccess resource
	GetSettingSuperCloudaccess(ctx context.Context, site string) (*SettingSuperCloudaccess, error)
	// UpdateSettingSuperCloudaccess updates SettingSuperCloudaccess resource
	UpdateSettingSuperCloudaccess(ctx context.Context, site string, d *SettingSuperCloudaccess) (*SettingSuperCloudaccess, error)
	/* client methods for SettingSuperEvents API */

	// GetSettingSuperEvents returns SettingSuperEvents resource
	GetSettingSuperEvents(ctx context.Context, site string) (*SettingSuperEvents, error)
	// UpdateSettingSuperEvents updates SettingSuperEvents resource
	UpdateSettingSuperEvents(ctx context.Context, site string, d *SettingSuperEvents) (*SettingSuperEvents, error)
	/* client methods for SettingSuperFwupdate API */

	// GetSettingSuperFwupdate returns SettingSuperFwupdate resource
	GetSettingSuperFwupdate(ctx context.Context, site string) (*SettingSuperFwupdate, error)
	// UpdateSettingSuperFwupdate updates SettingSuperFwupdate resource
	UpdateSettingSuperFwupdate(ctx context.Context, site string, d *SettingSuperFwupdate) (*SettingSuperFwupdate, error)
	/* client methods for SettingSuperIdentity API */

	// GetSettingSuperIdentity returns SettingSuperIdentity resource
	GetSettingSuperIdentity(ctx context.Context, site string) (*SettingSuperIdentity, error)
	// UpdateSettingSuperIdentity updates SettingSuperIdentity resource
	UpdateSettingSuperIdentity(ctx context.Context, site string, d *SettingSuperIdentity) (*SettingSuperIdentity, error)
	/* client methods for SettingSuperMail API */

	// GetSettingSuperMail returns SettingSuperMail resource
	GetSettingSuperMail(ctx context.Context, site string) (*SettingSuperMail, error)
	// UpdateSettingSuperMail updates SettingSuperMail resource
	UpdateSettingSuperMail(ctx context.Context, site string, d *SettingSuperMail) (*SettingSuperMail, error)
	/* client methods for SettingSuperMgmt API */

	// GetSettingSuperMgmt returns SettingSuperMgmt resource
	GetSettingSuperMgmt(ctx context.Context, site string) (*SettingSuperMgmt, error)
	// UpdateSettingSuperMgmt updates SettingSuperMgmt resource
	UpdateSettingSuperMgmt(ctx context.Context, site string, d *SettingSuperMgmt) (*SettingSuperMgmt, error)
	/* client methods for SettingSuperSdn API */

	// GetSettingSuperSdn returns SettingSuperSdn resource
	GetSettingSuperSdn(ctx context.Context, site string) (*SettingSuperSdn, error)
	// UpdateSettingSuperSdn updates SettingSuperSdn resource
	UpdateSettingSuperSdn(ctx context.Context, site string, d *SettingSuperSdn) (*SettingSuperSdn, error)
	/* client methods for SettingSuperSmtp API */

	// GetSettingSuperSmtp returns SettingSuperSmtp resource
	GetSettingSuperSmtp(ctx context.Context, site string) (*SettingSuperSmtp, error)
	// UpdateSettingSuperSmtp updates SettingSuperSmtp resource
	UpdateSettingSuperSmtp(ctx context.Context, site string, d *SettingSuperSmtp) (*SettingSuperSmtp, error)
	/* client methods for SettingTeleport API */

	// GetSettingTeleport returns SettingTeleport resource
	GetSettingTeleport(ctx context.Context, site string) (*SettingTeleport, error)
	// UpdateSettingTeleport updates SettingTeleport resource
	UpdateSettingTeleport(ctx context.Context, site string, d *SettingTeleport) (*SettingTeleport, error)
	/* client methods for SettingUsg API */

	// GetSettingUsg returns SettingUsg resource
	GetSettingUsg(ctx context.Context, site string) (*SettingUsg, error)
	// UpdateSettingUsg updates SettingUsg resource
	UpdateSettingUsg(ctx context.Context, site string, d *SettingUsg) (*SettingUsg, error)
	/* client methods for SettingUsw API */

	// GetSettingUsw returns SettingUsw resource
	GetSettingUsw(ctx context.Context, site string) (*SettingUsw, error)
	// UpdateSettingUsw updates SettingUsw resource
	UpdateSettingUsw(ctx context.Context, site string, d *SettingUsw) (*SettingUsw, error)
	/* client methods for SpatialRecord API */

	// GetSpatialRecord returns SpatialRecord resource by its ID
	GetSpatialRecord(ctx context.Context, site, id string) (*SpatialRecord, error)
	// UpdateSpatialRecord updates SpatialRecord resource by its ID
	UpdateSpatialRecord(ctx context.Context, site string, d *SpatialRecord) (*SpatialRecord, error)
	// ListSpatialRecord returns list of SpatialRecord resources
	ListSpatialRecord(ctx context.Context, site string) ([]SpatialRecord, error)
	// DeleteSpatialRecord deletes SpatialRecord resource by its ID
	DeleteSpatialRecord(ctx context.Context, site, id string) error
	// CreateSpatialRecord creates new SpatialRecord resource
	CreateSpatialRecord(ctx context.Context, site string, d *SpatialRecord) (*SpatialRecord, error)

	/* client methods for Tag API */

	// GetTag returns Tag resource by its ID
	GetTag(ctx context.Context, site, id string) (*Tag, error)
	// UpdateTag updates Tag resource by its ID
	UpdateTag(ctx context.Context, site string, d *Tag) (*Tag, error)
	// ListTag returns list of Tag resources
	ListTag(ctx context.Context, site string) ([]Tag, error)
	// DeleteTag deletes Tag resource by its ID
	DeleteTag(ctx context.Context, site, id string) error
	// CreateTag creates new Tag resource
	CreateTag(ctx context.Context, site string, d *Tag) (*Tag, error)

	/* client methods for User API */

	// GetUser returns User resource by its ID
	GetUser(ctx context.Context, site, id string) (*User, error)
	// UpdateUser updates User resource by its ID
	UpdateUser(ctx context.Context, site string, d *User) (*User, error)
	// ListUser returns list of User resources
	ListUser(ctx context.Context, site string) ([]User, error)
	// DeleteUser deletes User resource by its ID
	DeleteUser(ctx context.Context, site, id string) error
	// CreateUser creates new User resource
	CreateUser(ctx context.Context, site string, d *User) (*User, error)

	/* client methods for UserGroup API */

	// GetUserGroup returns UserGroup resource by its ID
	GetUserGroup(ctx context.Context, site, id string) (*UserGroup, error)
	// UpdateUserGroup updates UserGroup resource by its ID
	UpdateUserGroup(ctx context.Context, site string, d *UserGroup) (*UserGroup, error)
	// ListUserGroup returns list of UserGroup resources
	ListUserGroup(ctx context.Context, site string) ([]UserGroup, error)
	// DeleteUserGroup deletes UserGroup resource by its ID
	DeleteUserGroup(ctx context.Context, site, id string) error
	// CreateUserGroup creates new UserGroup resource
	CreateUserGroup(ctx context.Context, site string, d *UserGroup) (*UserGroup, error)

	/* client methods for VirtualDevice API */

	// GetVirtualDevice returns VirtualDevice resource by its ID
	GetVirtualDevice(ctx context.Context, site, id string) (*VirtualDevice, error)
	// UpdateVirtualDevice updates VirtualDevice resource by its ID
	UpdateVirtualDevice(ctx context.Context, site string, d *VirtualDevice) (*VirtualDevice, error)
	// ListVirtualDevice returns list of VirtualDevice resources
	ListVirtualDevice(ctx context.Context, site string) ([]VirtualDevice, error)
	// DeleteVirtualDevice deletes VirtualDevice resource by its ID
	DeleteVirtualDevice(ctx context.Context, site, id string) error
	// CreateVirtualDevice creates new VirtualDevice resource
	CreateVirtualDevice(ctx context.Context, site string, d *VirtualDevice) (*VirtualDevice, error)

	/* client methods for WLAN API */

	// GetWLAN returns WLAN resource by its ID
	GetWLAN(ctx context.Context, site, id string) (*WLAN, error)
	// UpdateWLAN updates WLAN resource by its ID
	UpdateWLAN(ctx context.Context, site string, d *WLAN) (*WLAN, error)
	// ListWLAN returns list of WLAN resources
	ListWLAN(ctx context.Context, site string) ([]WLAN, error)
	// DeleteWLAN deletes WLAN resource by its ID
	DeleteWLAN(ctx context.Context, site, id string) error
	// CreateWLAN creates new WLAN resource
	CreateWLAN(ctx context.Context, site string, d *WLAN) (*WLAN, error)

	/* client methods for WLANGroup API */

	// GetWLANGroup returns WLANGroup resource by its ID
	GetWLANGroup(ctx context.Context, site, id string) (*WLANGroup, error)
	// UpdateWLANGroup updates WLANGroup resource by its ID
	UpdateWLANGroup(ctx context.Context, site string, d *WLANGroup) (*WLANGroup, error)
	// ListWLANGroup returns list of WLANGroup resources
	ListWLANGroup(ctx context.Context, site string) ([]WLANGroup, error)
	// DeleteWLANGroup deletes WLANGroup resource by its ID
	DeleteWLANGroup(ctx context.Context, site, id string) error
	// CreateWLANGroup creates new WLANGroup resource
	CreateWLANGroup(ctx context.Context, site string, d *WLANGroup) (*WLANGroup, error)
}
