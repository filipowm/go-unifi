// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"io"
)

type Client interface {
	Logger

	// BaseURL returns the base URL of the controller.
	BaseURL() string

	// Delete sends a DELETE request to the controller.
	Delete(ctx context.Context, apiPath string, reqBody any, respBody any) error

	// Do sends a request to the controller.
	Do(ctx context.Context, method string, apiPath string, reqBody any, respBody any) error

	// Get sends a GET request to the controller.
	Get(ctx context.Context, apiPath string, reqBody any, respBody any) error

	// Login logs in to the controller. Useful only for user/password authentication.
	Login() error

	// Logout logs out from the controller.
	Logout() error

	// Post sends a POST request to the controller.
	Post(ctx context.Context, apiPath string, reqBody any, respBody any) error

	// Put sends a PUT request to the controller.
	Put(ctx context.Context, apiPath string, reqBody any, respBody any) error

	// Version returns the version of the UniFi Controller API.
	Version() string

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

	// ==== client methods for ContentFiltering resource ====

	// CreateContentFiltering creates a resource
	CreateContentFiltering(ctx context.Context, site string, c *ContentFiltering) (*ContentFiltering, error)

	// DeleteContentFiltering deletes a resource
	DeleteContentFiltering(ctx context.Context, site string, id string) error

	// ListContentFiltering lists the resources
	ListContentFiltering(ctx context.Context, site string) ([]ContentFiltering, error)

	// UpdateContentFiltering updates a resource
	UpdateContentFiltering(ctx context.Context, site string, c *ContentFiltering) (*ContentFiltering, error)

	// ==== end of client methods for ContentFiltering resource ====

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

	// GetFeature returns a specific feature by it's name. Name is case-insensitive.
	GetFeature(ctx context.Context, site string, name string) (*DescribedFeature, error)

	// IsFeatureEnabled returns if a specific feature is enabled by it's name. Name is case-insensitive.
	IsFeatureEnabled(ctx context.Context, site string, name string) (bool, error)

	// ListFeatures returns all features of the UniFi controller.
	ListFeatures(ctx context.Context, site string) ([]DescribedFeature, error)

	// AdoptDevice adopts a device by MAC address.
	AdoptDevice(ctx context.Context, site string, mac string) error

	// ForgetDevice forgets a device by MAC address.
	ForgetDevice(ctx context.Context, site string, mac string) error

	GetDeviceByMAC(ctx context.Context, site string, mac string) (*Device, error)

	ReorderFirewallRules(ctx context.Context, site string, ruleset string, reorder []FirewallRuleIndexUpdate) error

	// ==== client methods for FirewallZone resource ====

	// CreateFirewallZone creates a resource
	CreateFirewallZone(ctx context.Context, site string, f *FirewallZone) (*FirewallZone, error)

	// DeleteFirewallZone deletes a resource
	DeleteFirewallZone(ctx context.Context, site string, id string) error

	// GetFirewallZone retrieves a resource
	GetFirewallZone(ctx context.Context, site string, id string) (*FirewallZone, error)

	// ListFirewallZone lists the resources
	ListFirewallZone(ctx context.Context, site string) ([]FirewallZone, error)

	// UpdateFirewallZone updates a resource
	UpdateFirewallZone(ctx context.Context, site string, f *FirewallZone) (*FirewallZone, error)

	ListFirewallZoneMatrix(ctx context.Context, site string) ([]FirewallZoneMatrix, error)

	// ==== client methods for FirewallZonePolicy resource ====

	// CreateFirewallZonePolicy creates a resource
	CreateFirewallZonePolicy(ctx context.Context, site string, f *FirewallZonePolicy) (*FirewallZonePolicy, error)

	// DeleteFirewallZonePolicy deletes a resource
	DeleteFirewallZonePolicy(ctx context.Context, site string, id string) error

	// GetFirewallZonePolicy retrieves a resource
	GetFirewallZonePolicy(ctx context.Context, site string, id string) (*FirewallZonePolicy, error)

	// ListFirewallZonePolicy lists the resources
	ListFirewallZonePolicy(ctx context.Context, site string) ([]FirewallZonePolicy, error)

	ReorderFirewallPolicies(ctx context.Context, site string, d *FirewallPolicyOrderUpdate) ([]FirewallZonePolicy, error)

	// UpdateFirewallZonePolicy updates a resource
	UpdateFirewallZonePolicy(ctx context.Context, site string, f *FirewallZonePolicy) (*FirewallZonePolicy, error)

	// ==== end of client methods for FirewallZonePolicy resource ====

	// ==== end of client methods for FirewallZone resource ====

	// DeletePortalFile deletes a Hotspot Portal file from the controller.
	DeletePortalFile(ctx context.Context, site string, id string) error

	// GetPortalFile returns a specific Hotspot Portal file by it's ID.
	GetPortalFile(ctx context.Context, site string, id string) (*PortalFile, error)

	// ListPortalFiles lists all Hotspot Portal files on the controller.
	ListPortalFiles(ctx context.Context, site string) ([]PortalFile, error)

	// UploadPortalFile uploads a Hotspot Portal file to the controller.
	UploadPortalFile(ctx context.Context, site string, filepath string) (*PortalFile, error)

	// UploadPortalFileFromReader uploads a Hotspot Portal file using io.Reader to the controller.
	UploadPortalFileFromReader(ctx context.Context, site string, reader io.Reader, filename string) (*PortalFile, error)

	GetSetting(ctx context.Context, site string, key string) (*Setting, any, error)

	CreateSite(ctx context.Context, description string) ([]Site, error)

	DeleteSite(ctx context.Context, id string) ([]Site, error)

	GetSite(ctx context.Context, id string) (*Site, error)

	ListSites(ctx context.Context) ([]Site, error)

	UpdateSite(ctx context.Context, name string, description string) ([]Site, error)

	GetSystemInfo(ctx context.Context, id string) (*SysInfo, error)

	GetSystemInformation() (*SysInfo, error)

	BlockUserByMAC(ctx context.Context, site string, mac string) error

	DeleteUserByMAC(ctx context.Context, site string, mac string) error

	GetUserByMAC(ctx context.Context, site string, mac string) (*User, error)

	KickUserByMAC(ctx context.Context, site string, mac string) error

	OverrideUserFingerprint(ctx context.Context, site string, mac string, devIdOverride int) error

	UnblockUserByMAC(ctx context.Context, site string, mac string) error
}
