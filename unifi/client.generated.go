// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

type client interface {

	// BaseURL returns the base URL of the controller.
	BaseURL() string

	// Login logs in to the controller. Useful only for user/password authentication.
	Login() error

	// Logout logs out from the controller.
	Logout() error

	// client methods for Account resource

	// CreateAccount creates a resource
	CreateAccount(ctx context.Context, site string, a *Account) (*Account, error)

	// DeleteAccount deletes a resource
	DeleteAccount(ctx context.Context, site string, id string) error

	// GetAccount retrieves a resource
	GetAccount(ctx context.Context, site string, id string) (*Account, error)

	// ListAccount lists the resources
	ListAccount(ctx context.Context, site string) ([]*Account, error)

	// UpdateAccount updates a resource
	UpdateAccount(ctx context.Context, site string, a *Account) (*Account, error)

	// client methods for BroadcastGroup resource

	// CreateBroadcastGroup creates a resource
	CreateBroadcastGroup(ctx context.Context, site string, b *BroadcastGroup) (*BroadcastGroup, error)

	// DeleteBroadcastGroup deletes a resource
	DeleteBroadcastGroup(ctx context.Context, site string, id string) error

	// GetBroadcastGroup retrieves a resource
	GetBroadcastGroup(ctx context.Context, site string, id string) (*BroadcastGroup, error)

	// ListBroadcastGroup lists the resources
	ListBroadcastGroup(ctx context.Context, site string) ([]*BroadcastGroup, error)

	// UpdateBroadcastGroup updates a resource
	UpdateBroadcastGroup(ctx context.Context, site string, b *BroadcastGroup) (*BroadcastGroup, error)

	// client methods for ChannelPlan resource

	// CreateChannelPlan creates a resource
	CreateChannelPlan(ctx context.Context, site string, c *ChannelPlan) (*ChannelPlan, error)

	// DeleteChannelPlan deletes a resource
	DeleteChannelPlan(ctx context.Context, site string, id string) error

	// GetChannelPlan retrieves a resource
	GetChannelPlan(ctx context.Context, site string, id string) (*ChannelPlan, error)

	// ListChannelPlan lists the resources
	ListChannelPlan(ctx context.Context, site string) ([]*ChannelPlan, error)

	// UpdateChannelPlan updates a resource
	UpdateChannelPlan(ctx context.Context, site string, c *ChannelPlan) (*ChannelPlan, error)

	// client methods for DHCPOption resource

	// CreateDHCPOption creates a resource
	CreateDHCPOption(ctx context.Context, site string, d *DHCPOption) (*DHCPOption, error)

	// DeleteDHCPOption deletes a resource
	DeleteDHCPOption(ctx context.Context, site string, id string) error

	// GetDHCPOption retrieves a resource
	GetDHCPOption(ctx context.Context, site string, id string) (*DHCPOption, error)

	// ListDHCPOption lists the resources
	ListDHCPOption(ctx context.Context, site string) ([]*DHCPOption, error)

	// UpdateDHCPOption updates a resource
	UpdateDHCPOption(ctx context.Context, site string, d *DHCPOption) (*DHCPOption, error)

	// client methods for Dashboard resource

	// CreateDashboard creates a resource
	CreateDashboard(ctx context.Context, site string, d *Dashboard) (*Dashboard, error)

	// DeleteDashboard deletes a resource
	DeleteDashboard(ctx context.Context, site string, id string) error

	// GetDashboard retrieves a resource
	GetDashboard(ctx context.Context, site string, id string) (*Dashboard, error)

	// ListDashboard lists the resources
	ListDashboard(ctx context.Context, site string) ([]*Dashboard, error)

	// UpdateDashboard updates a resource
	UpdateDashboard(ctx context.Context, site string, d *Dashboard) (*Dashboard, error)

	// client methods for Device resource

	// AdoptDevice adopts a device by MAC address.
	AdoptDevice(ctx context.Context, site string, mac string) error

	// CreateDevice creates a resource
	CreateDevice(ctx context.Context, site string, d *Device) (*Device, error)

	// DeleteDevice deletes a resource
	DeleteDevice(ctx context.Context, site string, id string) error

	// GetDevice retrieves a resource
	GetDevice(ctx context.Context, site string, id string) (*Device, error)

	// ListDevice lists the resources
	ListDevice(ctx context.Context, site string) ([]*Device, error)

	// UpdateDevice updates a resource
	UpdateDevice(ctx context.Context, site string, d *Device) (*Device, error)

	// client methods for DpiApp resource

	// CreateDpiApp creates a resource
	CreateDpiApp(ctx context.Context, site string, d *DpiApp) (*DpiApp, error)

	// DeleteDpiApp deletes a resource
	DeleteDpiApp(ctx context.Context, site string, id string) error

	// GetDpiApp retrieves a resource
	GetDpiApp(ctx context.Context, site string, id string) (*DpiApp, error)

	// ListDpiApp lists the resources
	ListDpiApp(ctx context.Context, site string) ([]*DpiApp, error)

	// UpdateDpiApp updates a resource
	UpdateDpiApp(ctx context.Context, site string, d *DpiApp) (*DpiApp, error)

	// client methods for DpiGroup resource

	// CreateDpiGroup creates a resource
	CreateDpiGroup(ctx context.Context, site string, d *DpiGroup) (*DpiGroup, error)

	// DeleteDpiGroup deletes a resource
	DeleteDpiGroup(ctx context.Context, site string, id string) error

	// GetDpiGroup retrieves a resource
	GetDpiGroup(ctx context.Context, site string, id string) (*DpiGroup, error)

	// ListDpiGroup lists the resources
	ListDpiGroup(ctx context.Context, site string) ([]*DpiGroup, error)

	// UpdateDpiGroup updates a resource
	UpdateDpiGroup(ctx context.Context, site string, d *DpiGroup) (*DpiGroup, error)

	// client methods for DynamicDNS resource

	// CreateDynamicDNS creates a resource
	CreateDynamicDNS(ctx context.Context, site string, d *DynamicDNS) (*DynamicDNS, error)

	// DeleteDynamicDNS deletes a resource
	DeleteDynamicDNS(ctx context.Context, site string, id string) error

	// GetDynamicDNS retrieves a resource
	GetDynamicDNS(ctx context.Context, site string, id string) (*DynamicDNS, error)

	// ListDynamicDNS lists the resources
	ListDynamicDNS(ctx context.Context, site string) ([]*DynamicDNS, error)

	// UpdateDynamicDNS updates a resource
	UpdateDynamicDNS(ctx context.Context, site string, d *DynamicDNS) (*DynamicDNS, error)

	// client methods for FirewallGroup resource

	// CreateFirewallGroup creates a resource
	CreateFirewallGroup(ctx context.Context, site string, f *FirewallGroup) (*FirewallGroup, error)

	// DeleteFirewallGroup deletes a resource
	DeleteFirewallGroup(ctx context.Context, site string, id string) error

	// GetFirewallGroup retrieves a resource
	GetFirewallGroup(ctx context.Context, site string, id string) (*FirewallGroup, error)

	// ListFirewallGroup lists the resources
	ListFirewallGroup(ctx context.Context, site string) ([]*FirewallGroup, error)

	// UpdateFirewallGroup updates a resource
	UpdateFirewallGroup(ctx context.Context, site string, f *FirewallGroup) (*FirewallGroup, error)

	// client methods for FirewallRule resource

	// CreateFirewallRule creates a resource
	CreateFirewallRule(ctx context.Context, site string, f *FirewallRule) (*FirewallRule, error)

	// DeleteFirewallRule deletes a resource
	DeleteFirewallRule(ctx context.Context, site string, id string) error

	// GetFirewallRule retrieves a resource
	GetFirewallRule(ctx context.Context, site string, id string) (*FirewallRule, error)

	// ListFirewallRule lists the resources
	ListFirewallRule(ctx context.Context, site string) ([]*FirewallRule, error)

	// UpdateFirewallRule updates a resource
	UpdateFirewallRule(ctx context.Context, site string, f *FirewallRule) (*FirewallRule, error)

	// client methods for HeatMap resource

	// CreateHeatMap creates a resource
	CreateHeatMap(ctx context.Context, site string, h *HeatMap) (*HeatMap, error)

	// DeleteHeatMap deletes a resource
	DeleteHeatMap(ctx context.Context, site string, id string) error

	// GetHeatMap retrieves a resource
	GetHeatMap(ctx context.Context, site string, id string) (*HeatMap, error)

	// ListHeatMap lists the resources
	ListHeatMap(ctx context.Context, site string) ([]*HeatMap, error)

	// UpdateHeatMap updates a resource
	UpdateHeatMap(ctx context.Context, site string, h *HeatMap) (*HeatMap, error)

	// client methods for HeatMapPoint resource

	// CreateHeatMapPoint creates a resource
	CreateHeatMapPoint(ctx context.Context, site string, h *HeatMapPoint) (*HeatMapPoint, error)

	// DeleteHeatMapPoint deletes a resource
	DeleteHeatMapPoint(ctx context.Context, site string, id string) error

	// GetHeatMapPoint retrieves a resource
	GetHeatMapPoint(ctx context.Context, site string, id string) (*HeatMapPoint, error)

	// ListHeatMapPoint lists the resources
	ListHeatMapPoint(ctx context.Context, site string) ([]*HeatMapPoint, error)

	// UpdateHeatMapPoint updates a resource
	UpdateHeatMapPoint(ctx context.Context, site string, h *HeatMapPoint) (*HeatMapPoint, error)

	// client methods for Hotspot2Conf resource

	// CreateHotspot2Conf creates a resource
	CreateHotspot2Conf(ctx context.Context, site string, h *Hotspot2Conf) (*Hotspot2Conf, error)

	// DeleteHotspot2Conf deletes a resource
	DeleteHotspot2Conf(ctx context.Context, site string, id string) error

	// GetHotspot2Conf retrieves a resource
	GetHotspot2Conf(ctx context.Context, site string, id string) (*Hotspot2Conf, error)

	// ListHotspot2Conf lists the resources
	ListHotspot2Conf(ctx context.Context, site string) ([]*Hotspot2Conf, error)

	// UpdateHotspot2Conf updates a resource
	UpdateHotspot2Conf(ctx context.Context, site string, h *Hotspot2Conf) (*Hotspot2Conf, error)

	// client methods for HotspotOp resource

	// CreateHotspotOp creates a resource
	CreateHotspotOp(ctx context.Context, site string, h *HotspotOp) (*HotspotOp, error)

	// DeleteHotspotOp deletes a resource
	DeleteHotspotOp(ctx context.Context, site string, id string) error

	// GetHotspotOp retrieves a resource
	GetHotspotOp(ctx context.Context, site string, id string) (*HotspotOp, error)

	// ListHotspotOp lists the resources
	ListHotspotOp(ctx context.Context, site string) ([]*HotspotOp, error)

	// UpdateHotspotOp updates a resource
	UpdateHotspotOp(ctx context.Context, site string, h *HotspotOp) (*HotspotOp, error)

	// client methods for HotspotPackage resource

	// CreateHotspotPackage creates a resource
	CreateHotspotPackage(ctx context.Context, site string, h *HotspotPackage) (*HotspotPackage, error)

	// DeleteHotspotPackage deletes a resource
	DeleteHotspotPackage(ctx context.Context, site string, id string) error

	// GetHotspotPackage retrieves a resource
	GetHotspotPackage(ctx context.Context, site string, id string) (*HotspotPackage, error)

	// ListHotspotPackage lists the resources
	ListHotspotPackage(ctx context.Context, site string) ([]*HotspotPackage, error)

	// UpdateHotspotPackage updates a resource
	UpdateHotspotPackage(ctx context.Context, site string, h *HotspotPackage) (*HotspotPackage, error)

	// client methods for Map resource

	// CreateMap creates a resource
	CreateMap(ctx context.Context, site string, m *Map) (*Map, error)

	// DeleteMap deletes a resource
	DeleteMap(ctx context.Context, site string, id string) error

	// GetMap retrieves a resource
	GetMap(ctx context.Context, site string, id string) (*Map, error)

	// ListMap lists the resources
	ListMap(ctx context.Context, site string) ([]*Map, error)

	// UpdateMap updates a resource
	UpdateMap(ctx context.Context, site string, m *Map) (*Map, error)

	// client methods for MediaFile resource

	// CreateMediaFile creates a resource
	CreateMediaFile(ctx context.Context, site string, m *MediaFile) (*MediaFile, error)

	// DeleteMediaFile deletes a resource
	DeleteMediaFile(ctx context.Context, site string, id string) error

	// GetMediaFile retrieves a resource
	GetMediaFile(ctx context.Context, site string, id string) (*MediaFile, error)

	// ListMediaFile lists the resources
	ListMediaFile(ctx context.Context, site string) ([]*MediaFile, error)

	// UpdateMediaFile updates a resource
	UpdateMediaFile(ctx context.Context, site string, m *MediaFile) (*MediaFile, error)

	// client methods for Network resource

	// CreateNetwork creates a resource
	CreateNetwork(ctx context.Context, site string, n *Network) (*Network, error)

	// DeleteNetwork deletes a resource
	DeleteNetwork(ctx context.Context, site string, id string) error

	// GetNetwork retrieves a resource
	GetNetwork(ctx context.Context, site string, id string) (*Network, error)

	// ListNetwork lists the resources
	ListNetwork(ctx context.Context, site string) ([]*Network, error)

	// UpdateNetwork updates a resource
	UpdateNetwork(ctx context.Context, site string, n *Network) (*Network, error)

	// client methods for PortForward resource

	// CreatePortForward creates a resource
	CreatePortForward(ctx context.Context, site string, p *PortForward) (*PortForward, error)

	// DeletePortForward deletes a resource
	DeletePortForward(ctx context.Context, site string, id string) error

	// GetPortForward retrieves a resource
	GetPortForward(ctx context.Context, site string, id string) (*PortForward, error)

	// ListPortForward lists the resources
	ListPortForward(ctx context.Context, site string) ([]*PortForward, error)

	// UpdatePortForward updates a resource
	UpdatePortForward(ctx context.Context, site string, p *PortForward) (*PortForward, error)

	// client methods for PortProfile resource

	// CreatePortProfile creates a resource
	CreatePortProfile(ctx context.Context, site string, p *PortProfile) (*PortProfile, error)

	// DeletePortProfile deletes a resource
	DeletePortProfile(ctx context.Context, site string, id string) error

	// GetPortProfile retrieves a resource
	GetPortProfile(ctx context.Context, site string, id string) (*PortProfile, error)

	// ListPortProfile lists the resources
	ListPortProfile(ctx context.Context, site string) ([]*PortProfile, error)

	// UpdatePortProfile updates a resource
	UpdatePortProfile(ctx context.Context, site string, p *PortProfile) (*PortProfile, error)

	// client methods for RADIUSProfile resource

	// CreateRADIUSProfile creates a resource
	CreateRADIUSProfile(ctx context.Context, site string, r *RADIUSProfile) (*RADIUSProfile, error)

	// DeleteRADIUSProfile deletes a resource
	DeleteRADIUSProfile(ctx context.Context, site string, id string) error

	// GetRADIUSProfile retrieves a resource
	GetRADIUSProfile(ctx context.Context, site string, id string) (*RADIUSProfile, error)

	// ListRADIUSProfile lists the resources
	ListRADIUSProfile(ctx context.Context, site string) ([]*RADIUSProfile, error)

	// UpdateRADIUSProfile updates a resource
	UpdateRADIUSProfile(ctx context.Context, site string, r *RADIUSProfile) (*RADIUSProfile, error)

	// client methods for Routing resource

	// CreateRouting creates a resource
	CreateRouting(ctx context.Context, site string, r *Routing) (*Routing, error)

	// DeleteRouting deletes a resource
	DeleteRouting(ctx context.Context, site string, id string) error

	// GetRouting retrieves a resource
	GetRouting(ctx context.Context, site string, id string) (*Routing, error)

	// ListRouting lists the resources
	ListRouting(ctx context.Context, site string) ([]*Routing, error)

	// UpdateRouting updates a resource
	UpdateRouting(ctx context.Context, site string, r *Routing) (*Routing, error)

	// client methods for ScheduleTask resource

	// CreateScheduleTask creates a resource
	CreateScheduleTask(ctx context.Context, site string, s *ScheduleTask) (*ScheduleTask, error)

	// DeleteScheduleTask deletes a resource
	DeleteScheduleTask(ctx context.Context, site string, id string) error

	// GetScheduleTask retrieves a resource
	GetScheduleTask(ctx context.Context, site string, id string) (*ScheduleTask, error)

	// ListScheduleTask lists the resources
	ListScheduleTask(ctx context.Context, site string) ([]*ScheduleTask, error)

	// UpdateScheduleTask updates a resource
	UpdateScheduleTask(ctx context.Context, site string, s *ScheduleTask) (*ScheduleTask, error)

	// client methods for SpatialRecord resource

	// CreateSpatialRecord creates a resource
	CreateSpatialRecord(ctx context.Context, site string, s *SpatialRecord) (*SpatialRecord, error)

	// DeleteSpatialRecord deletes a resource
	DeleteSpatialRecord(ctx context.Context, site string, id string) error

	// GetSpatialRecord retrieves a resource
	GetSpatialRecord(ctx context.Context, site string, id string) (*SpatialRecord, error)

	// ListSpatialRecord lists the resources
	ListSpatialRecord(ctx context.Context, site string) ([]*SpatialRecord, error)

	// UpdateSpatialRecord updates a resource
	UpdateSpatialRecord(ctx context.Context, site string, s *SpatialRecord) (*SpatialRecord, error)

	// client methods for Tag resource

	// CreateTag creates a resource
	CreateTag(ctx context.Context, site string, t *Tag) (*Tag, error)

	// DeleteTag deletes a resource
	DeleteTag(ctx context.Context, site string, id string) error

	// GetTag retrieves a resource
	GetTag(ctx context.Context, site string, id string) (*Tag, error)

	// ListTag lists the resources
	ListTag(ctx context.Context, site string) ([]*Tag, error)

	// UpdateTag updates a resource
	UpdateTag(ctx context.Context, site string, t *Tag) (*Tag, error)

	// client methods for User resource

	// CreateUser creates a resource
	CreateUser(ctx context.Context, site string, u *User) (*User, error)

	// DeleteUser deletes a resource
	DeleteUser(ctx context.Context, site string, id string) error

	// GetUser retrieves a resource
	GetUser(ctx context.Context, site string, id string) (*User, error)

	// ListUser lists the resources
	ListUser(ctx context.Context, site string) ([]*User, error)

	// UpdateUser updates a resource
	UpdateUser(ctx context.Context, site string, u *User) (*User, error)

	// client methods for UserGroup resource

	// CreateUserGroup creates a resource
	CreateUserGroup(ctx context.Context, site string, u *UserGroup) (*UserGroup, error)

	// DeleteUserGroup deletes a resource
	DeleteUserGroup(ctx context.Context, site string, id string) error

	// GetUserGroup retrieves a resource
	GetUserGroup(ctx context.Context, site string, id string) (*UserGroup, error)

	// ListUserGroup lists the resources
	ListUserGroup(ctx context.Context, site string) ([]*UserGroup, error)

	// UpdateUserGroup updates a resource
	UpdateUserGroup(ctx context.Context, site string, u *UserGroup) (*UserGroup, error)

	// client methods for VirtualDevice resource

	// CreateVirtualDevice creates a resource
	CreateVirtualDevice(ctx context.Context, site string, v *VirtualDevice) (*VirtualDevice, error)

	// DeleteVirtualDevice deletes a resource
	DeleteVirtualDevice(ctx context.Context, site string, id string) error

	// GetVirtualDevice retrieves a resource
	GetVirtualDevice(ctx context.Context, site string, id string) (*VirtualDevice, error)

	// ListVirtualDevice lists the resources
	ListVirtualDevice(ctx context.Context, site string) ([]*VirtualDevice, error)

	// UpdateVirtualDevice updates a resource
	UpdateVirtualDevice(ctx context.Context, site string, v *VirtualDevice) (*VirtualDevice, error)

	// client methods for WLAN resource

	// CreateWLAN creates a resource
	CreateWLAN(ctx context.Context, site string, w *WLAN) (*WLAN, error)

	// DeleteWLAN deletes a resource
	DeleteWLAN(ctx context.Context, site string, id string) error

	// GetWLAN retrieves a resource
	GetWLAN(ctx context.Context, site string, id string) (*WLAN, error)

	// ListWLAN lists the resources
	ListWLAN(ctx context.Context, site string) ([]*WLAN, error)

	// UpdateWLAN updates a resource
	UpdateWLAN(ctx context.Context, site string, w *WLAN) (*WLAN, error)

	// client methods for WLANGroup resource

	// CreateWLANGroup creates a resource
	CreateWLANGroup(ctx context.Context, site string, w *WLANGroup) (*WLANGroup, error)

	// DeleteWLANGroup deletes a resource
	DeleteWLANGroup(ctx context.Context, site string, id string) error

	// GetWLANGroup retrieves a resource
	GetWLANGroup(ctx context.Context, site string, id string) (*WLANGroup, error)

	// ListWLANGroup lists the resources
	ListWLANGroup(ctx context.Context, site string) ([]*WLANGroup, error)

	// UpdateWLANGroup updates a resource
	UpdateWLANGroup(ctx context.Context, site string, w *WLANGroup) (*WLANGroup, error)
}
