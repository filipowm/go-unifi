package unifi

import (
	"context"
	"fmt"
)

// TrafficFlowsRequest represents the request payload for fetching traffic flows.
type TrafficFlowsRequest struct {
	Risk                 []string `json:"risk"`
	Action               []string `json:"action"`
	Direction            []string `json:"direction"`
	Protocol             []string `json:"protocol"`
	Policy               []string `json:"policy"`
	PolicyType           []string `json:"policy_type"`
	Service              []string `json:"service"`
	SourceHost           []string `json:"source_host"`
	SourceMAC            []string `json:"source_mac"`
	SourceIP             []string `json:"source_ip"`
	SourcePort           []int    `json:"source_port"`
	SourceNetworkID      []string `json:"source_network_id"`
	SourceDomain         []string `json:"source_domain"`
	SourceZoneID         []string `json:"source_zone_id"`
	SourceRegion         []string `json:"source_region"`
	DestinationHost      []string `json:"destination_host"`
	DestinationMAC       []string `json:"destination_mac"`
	DestinationIP        []string `json:"destination_ip"`
	DestinationPort      []int    `json:"destination_port"`
	DestinationNetworkID []string `json:"destination_network_id"`
	DestinationDomain    []string `json:"destination_domain"`
	DestinationZoneID    []string `json:"destination_zone_id"`
	DestinationRegion    []string `json:"destination_region"`
	InNetworkID          []string `json:"in_network_id"`
	OutNetworkID         []string `json:"out_network_id"`
	NextAiQuery          []string `json:"next_ai_query"`
	ExceptFor            []string `json:"except_for"`
	TimestampFrom        int64    `json:"timestampFrom"`
	TimestampTo          int64    `json:"timestampTo"`
	PageNumber           int      `json:"pageNumber"`
	SearchText           string   `json:"search_text"`
	PageSize             int      `json:"pageSize"`
	SkipCount            bool     `json:"skip_count"`
}

// TrafficFlowsResponse represents the paginated response for traffic flows.
type TrafficFlowsResponse struct {
	Data              []TrafficFlow `json:"data"`
	HasNext           bool          `json:"has_next"`
	OrMore            bool          `json:"or_more"`
	PageNumber        int           `json:"page_number"`
	TotalElementCount int           `json:"total_element_count"`
	TotalPageCount    int           `json:"total_page_count"`
}

// TrafficFlow represents a single traffic flow entry.
type TrafficFlow struct {
	Action      string                 `json:"action"`
	Count       int                    `json:"count"`
	Destination TrafficFlowTarget      `json:"destination"`
	Direction   string                 `json:"direction"`
	ID          string                 `json:"id"`
	NextAi      []string               `json:"next_ai"`
	Policies    []TrafficFlowPolicy    `json:"policies"`
	Protocol    string                 `json:"protocol"`
	Risk        string                 `json:"risk"`
	Service     string                 `json:"service"`
	Source      TrafficFlowTarget      `json:"source"`
	Time        int64                  `json:"time"`
	TrafficData TrafficFlowTrafficData `json:"traffic_data"`
}

// TrafficFlowTarget represents the source or destination of a traffic flow.
type TrafficFlowTarget struct {
	ClientFingerprint *TrafficFlowClientFingerprint `json:"client_fingerprint,omitempty"`
	ClientName        string                        `json:"client_name,omitempty"`
	ClientOui         string                        `json:"client_oui,omitempty"`
	Domains           []string                      `json:"domains,omitempty"`
	HostName          string                        `json:"host_name,omitempty"`
	ID                string                        `json:"id,omitempty"`
	IP                string                        `json:"ip,omitempty"`
	MAC               string                        `json:"mac,omitempty"`
	NetworkID         string                        `json:"network_id,omitempty"`
	NetworkName       string                        `json:"network_name,omitempty"`
	Port              int                           `json:"port,omitempty"`
	Subnet            string                        `json:"subnet,omitempty"`
	ZoneID            string                        `json:"zone_id,omitempty"`
	ZoneName          string                        `json:"zone_name,omitempty"`
	Region            string                        `json:"region,omitempty"`
}

// TrafficFlowClientFingerprint represents the fingerprint of a client involved in a traffic flow.
type TrafficFlowClientFingerprint struct {
	ComputedDevID  int  `json:"computed_dev_id"`
	ComputedEngine int  `json:"computed_engine"`
	Confidence     int  `json:"confidence"`
	DevCat         int  `json:"dev_cat"`
	DevFamily      int  `json:"dev_family"`
	DevID          int  `json:"dev_id"`
	DevIDOverride  int  `json:"dev_id_override"`
	DevVendor      int  `json:"dev_vendor"`
	HasOverride    bool `json:"has_override"`
	OSClass        int  `json:"os_class"`
	OSName         int  `json:"os_name"`
}

// TrafficFlowPolicy represents a policy applied to a traffic flow.
type TrafficFlowPolicy struct {
	ID           string `json:"id"`
	InternalType string `json:"internal_type"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

// TrafficFlowTrafficData represents the traffic statistics of a traffic flow.
type TrafficFlowTrafficData struct {
	BytesRx   int64 `json:"bytes_rx"`
	PacketsRx int64 `json:"packets_rx"`
}

// GetTrafficFlows fetches traffic flows using the provided request payload.
func (c *client) GetTrafficFlows(ctx context.Context, site string, req *TrafficFlowsRequest) (*TrafficFlowsResponse, error) {
	var respBody TrafficFlowsResponse

	err := c.Post(ctx, fmt.Sprintf("%s/site/%s/traffic-flows", c.apiPaths.ApiV2Path, site), req, &respBody)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
