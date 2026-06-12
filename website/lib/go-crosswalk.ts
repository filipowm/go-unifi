// OpenAPI operationId -> go-unifi Official SDK call. Generated from the spec +
// unifi/official/*.generated.go; verified to compile. Regenerate when the spec changes.
export const goCrosswalk: Record<string, string> = {
  getWifiBroadcastDetails: `details, err := c.Official().WifiBroadcasts().Get(ctx, siteID, broadcastID)`,
  updateWifiBroadcast: `var body official.WifiBroadcastCreateOrUpdate
updated, err := c.Official().WifiBroadcasts().Update(ctx, siteID, broadcastID, body)`,
  deleteWifiBroadcast: `err := c.Official().WifiBroadcasts().Delete(ctx, siteID, broadcastID, nil)`,
  getTrafficMatchingList: `details, err := c.Official().TrafficMatchingLists().Get(ctx, siteID, listID)`,
  updateTrafficMatchingList: `var body official.TrafficMatchingListCreateOrUpdate
updated, err := c.Official().TrafficMatchingLists().Update(ctx, siteID, listID, body)`,
  deleteTrafficMatchingList: `err := c.Official().TrafficMatchingLists().Delete(ctx, siteID, listID)`,
  getNetworkDetails: `details, err := c.Official().Networks().Get(ctx, siteID, networkID)`,
  updateNetwork: `var body official.NetworkCreateOrUpdate
updated, err := c.Official().Networks().Update(ctx, siteID, networkID, body)`,
  deleteNetwork: `err := c.Official().Networks().Delete(ctx, siteID, networkID, nil)`,
  getFirewallZone: `zone, err := c.Official().Firewall().GetZone(ctx, siteID, zoneID)`,
  updateFirewallZone: `var body official.FirewallZoneCreateOrUpdate
updated, err := c.Official().Firewall().UpdateZone(ctx, siteID, zoneID, body)`,
  deleteFirewallZone: `err := c.Official().Firewall().DeleteZone(ctx, siteID, zoneID)`,
  getFirewallPolicy: `policy, err := c.Official().Firewall().GetPolicy(ctx, siteID, policyID)`,
  updateFirewallPolicy: `var body official.FirewallPolicyCreateOrUpdate
updated, err := c.Official().Firewall().UpdatePolicy(ctx, siteID, policyID, body)`,
  deleteFirewallPolicy: `err := c.Official().Firewall().DeletePolicy(ctx, siteID, policyID)`,
  patchFirewallPolicy: `var body official.PatchFirewallPolicy
patched, err := c.Official().Firewall().PatchPolicy(ctx, siteID, policyID, body)`,
  getFirewallPolicyOrdering: `ordering, err := c.Official().Firewall().GetPolicyOrdering(ctx, siteID, srcZoneID, dstZoneID)`,
  updateFirewallPolicyOrdering: `var body official.FirewallPolicyOrdering
ordering, err := c.Official().Firewall().UpdatePolicyOrdering(ctx, siteID, srcZoneID, dstZoneID, body)`,
  getDnsPolicy: `details, err := c.Official().DNSPolicies().Get(ctx, siteID, dnsPolicyID)`,
  updateDnsPolicy: `var body official.DNSPolicyCreateOrUpdate
updated, err := c.Official().DNSPolicies().Update(ctx, siteID, dnsPolicyID, body)`,
  deleteDnsPolicy: `err := c.Official().DNSPolicies().Delete(ctx, siteID, dnsPolicyID)`,
  getAclRule: `rule, err := c.Official().ACLs().GetRule(ctx, siteID, ruleID)`,
  updateAclRule: `var body official.ACLRuleUpdate
updated, err := c.Official().ACLs().UpdateRule(ctx, siteID, ruleID, body)`,
  deleteAclRule: `err := c.Official().ACLs().DeleteRule(ctx, siteID, ruleID)`,
  getAclRuleOrdering: `ordering, err := c.Official().ACLs().GetRuleOrdering(ctx, siteID)`,
  updateAclRuleOrdering: `var body official.ACLRuleOrdering
ordering, err := c.Official().ACLs().UpdateRuleOrdering(ctx, siteID, body)`,
  getWifiBroadcastPage: `page, err := c.Official().WifiBroadcasts().ListPage(ctx, siteID, nil)`,
  createWifiBroadcast: `var body official.WifiBroadcastCreateOrUpdate
created, err := c.Official().WifiBroadcasts().Create(ctx, siteID, body)`,
  getTrafficMatchingLists: `page, err := c.Official().TrafficMatchingLists().ListPage(ctx, siteID, nil)`,
  createTrafficMatchingList: `var body official.TrafficMatchingListCreateOrUpdate
created, err := c.Official().TrafficMatchingLists().Create(ctx, siteID, body)`,
  getNetworksOverviewPage: `page, err := c.Official().Networks().ListPage(ctx, siteID, nil)`,
  createNetwork: `var body official.NetworkCreateOrUpdate
created, err := c.Official().Networks().Create(ctx, siteID, body)`,
  getVouchers: `page, err := c.Official().Hotspot().ListVouchersPage(ctx, siteID, nil)`,
  createVouchers: `var body official.HotspotVoucherCreationRequest
created, err := c.Official().Hotspot().CreateVouchers(ctx, siteID, body)`,
  deleteVouchers: `result, err := c.Official().Hotspot().DeleteVouchers(ctx, siteID, "")`,
  getFirewallZones: `page, err := c.Official().Firewall().ListZonesPage(ctx, siteID, nil)`,
  createFirewallZone: `var body official.FirewallZoneCreateOrUpdate
created, err := c.Official().Firewall().CreateZone(ctx, siteID, body)`,
  getFirewallPolicies: `page, err := c.Official().Firewall().ListPoliciesPage(ctx, siteID, nil)`,
  createFirewallPolicy: `var body official.FirewallPolicyCreateOrUpdate
created, err := c.Official().Firewall().CreatePolicy(ctx, siteID, body)`,
  getDnsPolicyPage: `page, err := c.Official().DNSPolicies().ListPage(ctx, siteID, nil)`,
  createDnsPolicy: `var body official.DNSPolicyCreateOrUpdate
created, err := c.Official().DNSPolicies().Create(ctx, siteID, body)`,
  getAdoptedDeviceOverviewPage: `page, err := c.Official().Devices().ListAdoptedPage(ctx, siteID, nil)`,
  adoptDevice: `var body official.DeviceAdoptionRequest
adopted, err := c.Official().Devices().Adopt(ctx, siteID, body)`,
  executePortAction: `var body official.PortActionRequest
err := c.Official().Devices().ExecutePortAction(ctx, siteID, deviceID, portIdx, body)`,
  executeAdoptedDeviceAction: `var body official.DeviceActionRequest
err := c.Official().Devices().ExecuteAdoptedAction(ctx, siteID, deviceID, body)`,
  executeConnectedClientAction: `var body official.ClientActionRequest
resp, err := c.Official().Clients().ExecuteConnectedAction(ctx, siteID, clientID, body)`,
  getAclRulePage: `page, err := c.Official().ACLs().ListRulesPage(ctx, siteID, nil)`,
  createAclRule: `var body official.ACLRuleUpdate
created, err := c.Official().ACLs().CreateRule(ctx, siteID, body)`,
  getSiteOverviewPage: `page, err := c.Official().Sites().ListPage(ctx, nil)`,
  getWansOverviewPage: `page, err := c.Official().Supporting().ListWansPage(ctx, siteID, nil)`,
  getSiteToSiteVpnTunnelPage: `page, err := c.Official().Supporting().ListSiteToSiteVpnTunnelsPage(ctx, siteID, nil)`,
  getVpnServerPage: `page, err := c.Official().Supporting().ListVpnServersPage(ctx, siteID, nil)`,
  getRadiusProfileOverviewPage: `page, err := c.Official().Supporting().ListRadiusProfilesPage(ctx, siteID, nil)`,
  getNetworkReferences: `refs, err := c.Official().Networks().GetReferences(ctx, siteID, networkID)`,
  getVoucher: `voucher, err := c.Official().Hotspot().GetVoucher(ctx, siteID, voucherID)`,
  deleteVoucher: `result, err := c.Official().Hotspot().DeleteVoucher(ctx, siteID, voucherID)`,
  getAdoptedDeviceDetails: `device, err := c.Official().Devices().GetAdopted(ctx, siteID, deviceID)`,
  removeDevice: `err := c.Official().Devices().Remove(ctx, siteID, deviceID)`,
  getAdoptedDeviceLatestStatistics: `stats, err := c.Official().Devices().GetAdoptedLatestStatistics(ctx, siteID, deviceID)`,
  getDeviceTagPage: `page, err := c.Official().Supporting().ListDeviceTagsPage(ctx, siteID, nil)`,
  getConnectedClientOverviewPage: `page, err := c.Official().Clients().ListConnectedPage(ctx, siteID, nil)`,
  getConnectedClientDetails: `client, err := c.Official().Clients().GetConnected(ctx, siteID, clientID)`,
  getPendingDevicePage: `page, err := c.Official().Devices().ListPendingPage(ctx, nil)`,
  getInfo: `details, err := c.Official().Info().Get(ctx)`,
  getDpiApplicationCategories: `page, err := c.Official().Supporting().ListDpiApplicationCategoriesPage(ctx, nil)`,
  getDpiApplications: `page, err := c.Official().Supporting().ListDpiApplicationsPage(ctx, nil)`,
  getCountries: `page, err := c.Official().Supporting().ListCountriesPage(ctx, nil)`,
};
