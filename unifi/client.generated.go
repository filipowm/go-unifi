// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

type Client interface {

	// BaseURL returns the base URL of the controller.
	BaseURL() string

	// Delete sends a DELETE request to the controller.
	Delete(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error

	// Do sends a request to the controller.
	Do(ctx context.Context, method string, apiPath string, reqBody interface{}, respBody interface{}) error

	// Get sends a GET request to the controller.
	Get(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error

	// Login logs in to the controller. Useful only for user/password authentication.
	Login() error

	// Logout logs out from the controller.
	Logout() error

	// Post sends a POST request to the controller.
	Post(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error

	// Put sends a PUT request to the controller.
	Put(ctx context.Context, apiPath string, reqBody interface{}, respBody interface{}) error

	// ==== client methods for APGroup resource ====

	// CreateAPGroup creates a resource
	CreateAPGroup(ctx context.Context, site string, a *APGroup) (*APGroup, error)

	// DeleteAPGroup deletes a resource
	DeleteAPGroup(ctx context.Context, site string, id string) error

	// GetAPGroup retrieves a resource
	GetAPGroup(ctx context.Context, site string, id string) (*APGroup, error)

	// ListAPGroup lists the resources
	ListAPGroup(ctx context.Context, site string) ([]APGroup, error)

	// UpdateAPGroup updates a resource
	UpdateAPGroup(ctx context.Context, site string, a *APGroup) (*APGroup, error)

	// ==== end of client methods for APGroup resource ====

	// ==== client methods for Account resource ====

	// CreateAccount creates a resource
	CreateAccount(ctx context.Context, site string, a *Account) (*Account, error)

	// DeleteAccount deletes a resource
	DeleteAccount(ctx context.Context, site string, id string) error

	// GetAccount retrieves a resource
	GetAccount(ctx context.Context, site string, id string) (*Account, error)

	// ListAccount lists the resources
	ListAccount(ctx context.Context, site string) ([]Account, error)

	// UpdateAccount updates a resource
	UpdateAccount(ctx context.Context, site string, a *Account) (*Account, error)

	// ==== end of client methods for Account resource ====

	// ==== client methods for DNSRecord resource ====

	// CreateDNSRecord creates a resource
	CreateDNSRecord(ctx context.Context, site string, d *DNSRecord) (*DNSRecord, error)

	// DeleteDNSRecord deletes a resource
	DeleteDNSRecord(ctx context.Context, site string, id string) error

	// GetDNSRecord retrieves a resource
	GetDNSRecord(ctx context.Context, site string, id string) (*DNSRecord, error)

	// ListDNSRecord lists the resources
	ListDNSRecord(ctx context.Context, site string) ([]DNSRecord, error)

	// UpdateDNSRecord updates a resource
	UpdateDNSRecord(ctx context.Context, site string, d *DNSRecord) (*DNSRecord, error)

	// ==== end of client methods for DNSRecord resource ====

	// ==== client methods for Device resource ====

	// AdoptDevice adopts a device by MAC address.
	AdoptDevice(ctx context.Context, site string, mac string) error

	// CreateDevice creates a resource
	CreateDevice(ctx context.Context, site string, d *Device) (*Device, error)

	// DeleteDevice deletes a resource
	DeleteDevice(ctx context.Context, site string, id string) error

	// ForgetDevice forgets a device by MAC address.
	ForgetDevice(ctx context.Context, site string, mac string) error

	// GetDevice retrieves a resource
	GetDevice(ctx context.Context, site string, id string) (*Device, error)

	GetDeviceByMAC(ctx context.Context, site string, mac string) (*Device, error)

	// ListDevice lists the resources
	ListDevice(ctx context.Context, site string) ([]Device, error)

	// UpdateDevice updates a resource
	UpdateDevice(ctx context.Context, site string, d *Device) (*Device, error)

	// ==== end of client methods for Device resource ====

	// ==== client methods for DynamicDNS resource ====

	// CreateDynamicDNS creates a resource
	CreateDynamicDNS(ctx context.Context, site string, d *DynamicDNS) (*DynamicDNS, error)

	// DeleteDynamicDNS deletes a resource
	DeleteDynamicDNS(ctx context.Context, site string, id string) error

	// GetDynamicDNS retrieves a resource
	GetDynamicDNS(ctx context.Context, site string, id string) (*DynamicDNS, error)

	// ListDynamicDNS lists the resources
	ListDynamicDNS(ctx context.Context, site string) ([]DynamicDNS, error)

	// UpdateDynamicDNS updates a resource
	UpdateDynamicDNS(ctx context.Context, site string, d *DynamicDNS) (*DynamicDNS, error)

	// ==== end of client methods for DynamicDNS resource ====

	// ==== client methods for FirewallGroup resource ====

	// CreateFirewallGroup creates a resource
	CreateFirewallGroup(ctx context.Context, site string, f *FirewallGroup) (*FirewallGroup, error)

	// DeleteFirewallGroup deletes a resource
	DeleteFirewallGroup(ctx context.Context, site string, id string) error

	// GetFirewallGroup retrieves a resource
	GetFirewallGroup(ctx context.Context, site string, id string) (*FirewallGroup, error)

	// ListFirewallGroup lists the resources
	ListFirewallGroup(ctx context.Context, site string) ([]FirewallGroup, error)

	// UpdateFirewallGroup updates a resource
	UpdateFirewallGroup(ctx context.Context, site string, f *FirewallGroup) (*FirewallGroup, error)

	// ==== end of client methods for FirewallGroup resource ====

	// ==== client methods for FirewallRule resource ====

	// CreateFirewallRule creates a resource
	CreateFirewallRule(ctx context.Context, site string, f *FirewallRule) (*FirewallRule, error)

	// DeleteFirewallRule deletes a resource
	DeleteFirewallRule(ctx context.Context, site string, id string) error

	// GetFirewallRule retrieves a resource
	GetFirewallRule(ctx context.Context, site string, id string) (*FirewallRule, error)

	// ListFirewallRule lists the resources
	ListFirewallRule(ctx context.Context, site string) ([]FirewallRule, error)

	ReorderFirewallRules(ctx context.Context, site string, ruleset string, reorder []FirewallRuleIndexUpdate) error

	// UpdateFirewallRule updates a resource
	UpdateFirewallRule(ctx context.Context, site string, f *FirewallRule) (*FirewallRule, error)

	// ==== end of client methods for FirewallRule resource ====

	// ==== client methods for Network resource ====

	// CreateNetwork creates a resource
	CreateNetwork(ctx context.Context, site string, n *Network) (*Network, error)

	// DeleteNetwork deletes a resource
	DeleteNetwork(ctx context.Context, site string, id string) error

	// GetNetwork retrieves a resource
	GetNetwork(ctx context.Context, site string, id string) (*Network, error)

	// ListNetwork lists the resources
	ListNetwork(ctx context.Context, site string) ([]Network, error)

	// UpdateNetwork updates a resource
	UpdateNetwork(ctx context.Context, site string, n *Network) (*Network, error)

	// ==== end of client methods for Network resource ====

	// ==== client methods for PortForward resource ====

	// CreatePortForward creates a resource
	CreatePortForward(ctx context.Context, site string, p *PortForward) (*PortForward, error)

	// DeletePortForward deletes a resource
	DeletePortForward(ctx context.Context, site string, id string) error

	// GetPortForward retrieves a resource
	GetPortForward(ctx context.Context, site string, id string) (*PortForward, error)

	// ListPortForward lists the resources
	ListPortForward(ctx context.Context, site string) ([]PortForward, error)

	// UpdatePortForward updates a resource
	UpdatePortForward(ctx context.Context, site string, p *PortForward) (*PortForward, error)

	// ==== end of client methods for PortForward resource ====

	// ==== client methods for PortProfile resource ====

	// CreatePortProfile creates a resource
	CreatePortProfile(ctx context.Context, site string, p *PortProfile) (*PortProfile, error)

	// DeletePortProfile deletes a resource
	DeletePortProfile(ctx context.Context, site string, id string) error

	// GetPortProfile retrieves a resource
	GetPortProfile(ctx context.Context, site string, id string) (*PortProfile, error)

	// ListPortProfile lists the resources
	ListPortProfile(ctx context.Context, site string) ([]PortProfile, error)

	// UpdatePortProfile updates a resource
	UpdatePortProfile(ctx context.Context, site string, p *PortProfile) (*PortProfile, error)

	// ==== end of client methods for PortProfile resource ====

	// ==== client methods for RADIUSProfile resource ====

	// CreateRADIUSProfile creates a resource
	CreateRADIUSProfile(ctx context.Context, site string, r *RADIUSProfile) (*RADIUSProfile, error)

	// DeleteRADIUSProfile deletes a resource
	DeleteRADIUSProfile(ctx context.Context, site string, id string) error

	// GetRADIUSProfile retrieves a resource
	GetRADIUSProfile(ctx context.Context, site string, id string) (*RADIUSProfile, error)

	// ListRADIUSProfile lists the resources
	ListRADIUSProfile(ctx context.Context, site string) ([]RADIUSProfile, error)

	// UpdateRADIUSProfile updates a resource
	UpdateRADIUSProfile(ctx context.Context, site string, r *RADIUSProfile) (*RADIUSProfile, error)

	// ==== end of client methods for RADIUSProfile resource ====

	// ==== client methods for Routing resource ====

	// CreateRouting creates a resource
	CreateRouting(ctx context.Context, site string, r *Routing) (*Routing, error)

	// DeleteRouting deletes a resource
	DeleteRouting(ctx context.Context, site string, id string) error

	// GetRouting retrieves a resource
	GetRouting(ctx context.Context, site string, id string) (*Routing, error)

	// ListRouting lists the resources
	ListRouting(ctx context.Context, site string) ([]Routing, error)

	// UpdateRouting updates a resource
	UpdateRouting(ctx context.Context, site string, r *Routing) (*Routing, error)

	// ==== end of client methods for Routing resource ====

	GetSetting(ctx context.Context, site string, key string) (*Setting, interface{}, error)

	CreateSite(ctx context.Context, description string) ([]Site, error)

	DeleteSite(ctx context.Context, id string) ([]Site, error)

	GetSite(ctx context.Context, id string) (*Site, error)

	ListSites(ctx context.Context) ([]Site, error)

	UpdateSite(ctx context.Context, name string, description string) ([]Site, error)

	GetSystemInfo(ctx context.Context, id string) (*SysInfo, error)

	GetSystemInformation() (*SysInfo, error)

	// ==== client methods for User resource ====

	BlockUserByMAC(ctx context.Context, site string, mac string) error

	// CreateUser creates a resource
	CreateUser(ctx context.Context, site string, u *User) (*User, error)

	// DeleteUser deletes a resource
	DeleteUser(ctx context.Context, site string, id string) error

	DeleteUserByMAC(ctx context.Context, site string, mac string) error

	// GetUser retrieves a resource
	GetUser(ctx context.Context, site string, id string) (*User, error)

	GetUserByMAC(ctx context.Context, site string, mac string) (*User, error)

	KickUserByMAC(ctx context.Context, site string, mac string) error

	// ListUser lists the resources
	ListUser(ctx context.Context, site string) ([]User, error)

	OverrideUserFingerprint(ctx context.Context, site string, mac string, devIdOverride int) error

	UnblockUserByMAC(ctx context.Context, site string, mac string) error

	// UpdateUser updates a resource
	UpdateUser(ctx context.Context, site string, u *User) (*User, error)

	// ==== client methods for UserGroup resource ====

	// CreateUserGroup creates a resource
	CreateUserGroup(ctx context.Context, site string, u *UserGroup) (*UserGroup, error)

	// DeleteUserGroup deletes a resource
	DeleteUserGroup(ctx context.Context, site string, id string) error

	// GetUserGroup retrieves a resource
	GetUserGroup(ctx context.Context, site string, id string) (*UserGroup, error)

	// ListUserGroup lists the resources
	ListUserGroup(ctx context.Context, site string) ([]UserGroup, error)

	// UpdateUserGroup updates a resource
	UpdateUserGroup(ctx context.Context, site string, u *UserGroup) (*UserGroup, error)

	// ==== end of client methods for UserGroup resource ====

	// ==== end of client methods for User resource ====

	// ==== client methods for WLAN resource ====

	// CreateWLAN creates a resource
	CreateWLAN(ctx context.Context, site string, w *WLAN) (*WLAN, error)

	// DeleteWLAN deletes a resource
	DeleteWLAN(ctx context.Context, site string, id string) error

	// GetWLAN retrieves a resource
	GetWLAN(ctx context.Context, site string, id string) (*WLAN, error)

	// ListWLAN lists the resources
	ListWLAN(ctx context.Context, site string) ([]WLAN, error)

	// UpdateWLAN updates a resource
	UpdateWLAN(ctx context.Context, site string, w *WLAN) (*WLAN, error)

	// ==== client methods for WLANGroup resource ====

	// CreateWLANGroup creates a resource
	CreateWLANGroup(ctx context.Context, site string, w *WLANGroup) (*WLANGroup, error)

	// DeleteWLANGroup deletes a resource
	DeleteWLANGroup(ctx context.Context, site string, id string) error

	// GetWLANGroup retrieves a resource
	GetWLANGroup(ctx context.Context, site string, id string) (*WLANGroup, error)

	// ListWLANGroup lists the resources
	ListWLANGroup(ctx context.Context, site string) ([]WLANGroup, error)

	// UpdateWLANGroup updates a resource
	UpdateWLANGroup(ctx context.Context, site string, w *WLANGroup) (*WLANGroup, error)

	// ==== end of client methods for WLANGroup resource ====

	// ==== end of client methods for WLAN resource ====

}
