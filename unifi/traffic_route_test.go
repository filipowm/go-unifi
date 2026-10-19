package unifi //nolint: testpackage

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	trafficRouteSite = "default"
	trafficRouteID   = "00000000-0000-0000-0000-000000000001"
)

// fullTrafficRouteJSON sets every modeled field, including each target_devices type.
const fullTrafficRouteJSON = `{
	"_id": "00000000-0000-0000-0000-000000000001",
	"description": "Example route",
	"domains": [
		{"domain": "example.com", "port_ranges": [{"port_start": 8000, "port_stop": 8080}], "ports": [443]}
	],
	"enabled": true,
	"ip_addresses": [
		{"ip_or_subnet": "192.0.2.0/24", "ip_version": "v4", "port_ranges": [{"port_start": 1000, "port_stop": 2000}], "ports": [53]},
		{"ip_or_subnet": "2001:db8::/32", "ip_version": "v6", "port_ranges": [], "ports": []}
	],
	"ip_ranges": [
		{"ip_start": "198.51.100.10", "ip_stop": "198.51.100.20", "ip_version": "v4"}
	],
	"kill_switch_enabled": true,
	"matching_target": "IP",
	"network_id": "00000000-0000-0000-0000-0000000000a1",
	"next_hop": "192.0.2.1",
	"regions": ["US", "CA"],
	"target_devices": [
		{"client_mac": "00:00:5e:00:53:01", "type": "CLIENT"},
		{"network_id": "00000000-0000-0000-0000-0000000000b1", "type": "NETWORK"},
		{"type": "ALL_CLIENTS"}
	]
}`

// controllerShapeTrafficRouteJSON mirrors what a UniFi Network 11.x controller returns for a
// route that never set next_hop: every list present but empty.
const controllerShapeTrafficRouteJSON = `{
	"_id": "00000000-0000-0000-0000-000000000002",
	"description": "Example egress",
	"domains": [],
	"enabled": false,
	"ip_addresses": [],
	"ip_ranges": [],
	"kill_switch_enabled": false,
	"matching_target": "INTERNET",
	"network_id": "00000000-0000-0000-0000-0000000000a1",
	"regions": [],
	"target_devices": []
}`

func trafficRoutesPath() string {
	return apiV2("site/" + trafficRouteSite + "/trafficroutes")
}

func trafficRouteByIDPath() string {
	return trafficRoutesPath() + "/" + trafficRouteID
}

func TestTrafficRouteJSONRoundTrip(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input string
		want  string
	}{
		"every field survives a round trip": {
			input: fullTrafficRouteJSON,
			want:  fullTrafficRouteJSON,
		},
		// next_hop is absent on some controller objects; it comes back as the clearable "".
		"controller shape keeps empty lists and adds empty next_hop": {
			input: controllerShapeTrafficRouteJSON,
			want: `{
				"_id": "00000000-0000-0000-0000-000000000002",
				"description": "Example egress",
				"domains": [],
				"enabled": false,
				"ip_addresses": [],
				"ip_ranges": [],
				"kill_switch_enabled": false,
				"matching_target": "INTERNET",
				"network_id": "00000000-0000-0000-0000-0000000000a1",
				"next_hop": "",
				"regions": [],
				"target_devices": []
			}`,
		},
		"string encoded ports are accepted": {
			input: `{"_id": "x", "matching_target": "DOMAIN", "network_id": "n",
				"domains": [{"domain": "example.com", "ports": ["443"], "port_ranges": [{"port_start": "1", "port_stop": "2"}]}]}`,
			want: `{"_id": "x", "matching_target": "DOMAIN", "network_id": "n", "description": "", "enabled": false,
				"kill_switch_enabled": false, "next_hop": "", "ip_addresses": [], "ip_ranges": [], "regions": [], "target_devices": [],
				"domains": [{"domain": "example.com", "ports": [443], "port_ranges": [{"port_start": 1, "port_stop": 2}]}]}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var route TrafficRoute
			require.NoError(t, json.Unmarshal([]byte(tc.input), &route))

			got, err := json.Marshal(&route)
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(got))
		})
	}
}

func TestTrafficRouteUnmarshalFields(t *testing.T) {
	t.Parallel()

	var route TrafficRoute
	require.NoError(t, json.Unmarshal([]byte(fullTrafficRouteJSON), &route))

	want := TrafficRoute{
		ID:          trafficRouteID,
		Description: "Example route",
		Domains: []TrafficRouteDomains{{
			Domain:     "example.com",
			PortRanges: []TrafficRoutePortRanges{{PortStart: 8000, PortStop: 8080}},
			Ports:      []int{443},
		}},
		Enabled: true,
		IPAddresses: []TrafficRouteIPAddresses{
			{IPOrSubnet: "192.0.2.0/24", IPVersion: "v4", PortRanges: []TrafficRoutePortRanges{{PortStart: 1000, PortStop: 2000}}, Ports: []int{53}},
			{IPOrSubnet: "2001:db8::/32", IPVersion: "v6", PortRanges: []TrafficRoutePortRanges{}, Ports: []int{}},
		},
		IPRanges:          []TrafficRouteIPRanges{{IPStart: "198.51.100.10", IPStop: "198.51.100.20", IPVersion: "v4"}},
		KillSwitchEnabled: true,
		MatchingTarget:    "IP",
		NetworkID:         "00000000-0000-0000-0000-0000000000a1",
		NextHop:           "192.0.2.1",
		Regions:           []string{"US", "CA"},
		TargetDevices: []TrafficRouteTargetDevices{
			{ClientMAC: "00:00:5e:00:53:01", Type: "CLIENT"},
			{NetworkID: "00000000-0000-0000-0000-0000000000b1", Type: "NETWORK"},
			{Type: "ALL_CLIENTS"},
		},
	}
	assert.Equal(t, want, route)
}

// TestTrafficRouteMarshalClearsFields guards the full-replace PUT: cleared values must be
// sent as [] / "" / false, never dropped: the controller validates the PUT body as the complete object.
func TestTrafficRouteMarshalClearsFields(t *testing.T) {
	t.Parallel()

	const clearedTopLevel = `"_id": "00000000-0000-0000-0000-000000000001", "description": "", "enabled": false,
		"kill_switch_enabled": false, "matching_target": "INTERNET", "network_id": "00000000-0000-0000-0000-0000000000a1",
		"next_hop": "", "ip_addresses": [], "ip_ranges": [], "regions": [], "target_devices": []`

	base := func() TrafficRoute {
		return TrafficRoute{ID: trafficRouteID, MatchingTarget: "INTERNET", NetworkID: "00000000-0000-0000-0000-0000000000a1"}
	}

	cases := map[string]struct {
		route func() TrafficRoute
		want  string
	}{
		"nil lists and empty strings are sent": {
			route: base,
			want:  `{` + clearedTopLevel + `, "domains": []}`,
		},
		"explicitly empty lists are sent": {
			route: func() TrafficRoute {
				r := base()
				r.Domains = []TrafficRouteDomains{}
				r.IPAddresses = []TrafficRouteIPAddresses{}
				r.IPRanges = []TrafficRouteIPRanges{}
				r.Regions = []string{}
				r.TargetDevices = []TrafficRouteTargetDevices{}
				return r
			},
			want: `{` + clearedTopLevel + `, "domains": []}`,
		},
		"nested nil port lists are sent": {
			route: func() TrafficRoute {
				r := base()
				r.Domains = []TrafficRouteDomains{{Domain: "example.com"}}
				return r
			},
			want: `{` + clearedTopLevel + `, "domains": [{"domain": "example.com", "port_ranges": [], "ports": []}]}`,
		},
		"nested ip address nil port lists are sent": {
			route: func() TrafficRoute {
				r := base()
				r.IPAddresses = []TrafficRouteIPAddresses{{IPOrSubnet: "192.0.2.10", IPVersion: "v4"}}
				return r
			},
			want: `{"_id": "00000000-0000-0000-0000-000000000001", "description": "", "enabled": false,
				"kill_switch_enabled": false, "matching_target": "INTERNET", "network_id": "00000000-0000-0000-0000-0000000000a1",
				"next_hop": "", "domains": [], "ip_ranges": [], "regions": [], "target_devices": [],
				"ip_addresses": [{"ip_or_subnet": "192.0.2.10", "ip_version": "v4", "port_ranges": [], "ports": []}]}`,
		},
		"target devices only carry the key their type uses": {
			route: func() TrafficRoute {
				r := base()
				r.TargetDevices = []TrafficRouteTargetDevices{
					{ClientMAC: "00:00:5e:00:53:01", Type: "CLIENT"},
					{NetworkID: "00000000-0000-0000-0000-0000000000b1", Type: "NETWORK"},
				}
				return r
			},
			want: `{"_id": "00000000-0000-0000-0000-000000000001", "description": "", "enabled": false,
				"kill_switch_enabled": false, "matching_target": "INTERNET", "network_id": "00000000-0000-0000-0000-0000000000a1",
				"next_hop": "", "domains": [], "ip_addresses": [], "ip_ranges": [], "regions": [],
				"target_devices": [{"client_mac": "00:00:5e:00:53:01", "type": "CLIENT"},
					{"network_id": "00000000-0000-0000-0000-0000000000b1", "type": "NETWORK"}]}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			route := tc.route()
			got, err := json.Marshal(&route)
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(got))
		})
	}
}

// TestTrafficRouteMarshalPointerReceiver pins the documented scope of the [] guarantee: it holds
// for a *TrafficRoute (what every client method passes), including one nested behind a pointer,
// but not for a TrafficRoute marshalled by value.
func TestTrafficRouteMarshalPointerReceiver(t *testing.T) {
	t.Parallel()

	lists := []string{"domains", "ip_addresses", "ip_ranges", "regions", "target_devices"}
	assertLists := func(t *testing.T, got []byte, want string) {
		t.Helper()
		var decoded map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(got, &decoded))
		for _, key := range lists {
			assert.JSONEq(t, want, string(decoded[key]), key)
		}
	}

	t.Run("pointer sends empty lists", func(t *testing.T) {
		t.Parallel()
		got, err := json.Marshal(&TrafficRoute{})
		require.NoError(t, err)
		assertLists(t, got, `[]`)
	})

	t.Run("pointer nested in a wrapper sends empty lists", func(t *testing.T) {
		t.Parallel()
		got, err := json.Marshal(struct {
			Route *TrafficRoute `json:"route"`
		}{Route: &TrafficRoute{}})
		require.NoError(t, err)
		var wrapper struct {
			Route json.RawMessage `json:"route"`
		}
		require.NoError(t, json.Unmarshal(got, &wrapper))
		assertLists(t, wrapper.Route, `[]`)
	})

	t.Run("value marshals nil lists as null", func(t *testing.T) {
		t.Parallel()
		got, err := json.Marshal(TrafficRoute{})
		require.NoError(t, err)
		assertLists(t, got, `null`)
	})
}

func TestListTrafficRoute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		response string
		wantIDs  []string
	}{
		"returns every route": {
			response: `[` + fullTrafficRouteJSON + `,` + controllerShapeTrafficRouteJSON + `]`,
			wantIDs:  []string{trafficRouteID, "00000000-0000-0000-0000-000000000002"},
		},
		"empty list": {
			response: `[]`,
			wantIDs:  []string{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{trafficRoutesPath(), staticJSON(tc.response)})

			got, err := cs.client().ListTrafficRoute(context.Background(), trafficRouteSite)
			require.NoError(t, err)

			req := cs.lastRequest()
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, trafficRoutesPath(), req.Path)

			ids := make([]string, 0, len(got))
			for _, r := range got {
				ids = append(ids, r.ID)
			}
			assert.Equal(t, tc.wantIDs, ids)
		})
	}
}

func TestGetTrafficRoute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		response  string
		status    int
		id        string
		wantDesc  string
		wantErrIs error
		wantErr   bool
	}{
		"found by filtering the list": {
			response: `[` + controllerShapeTrafficRouteJSON + `,` + fullTrafficRouteJSON + `]`,
			id:       trafficRouteID,
			wantDesc: "Example route",
		},
		"missing id yields ErrNotFound": {
			response:  `[` + controllerShapeTrafficRouteJSON + `]`,
			id:        trafficRouteID,
			wantErrIs: ErrNotFound,
		},
		"empty list yields ErrNotFound": {
			response:  `[]`,
			id:        trafficRouteID,
			wantErrIs: ErrNotFound,
		},
		"server error is surfaced": {
			response: `{"errorCode": 500, "message": "boom"}`,
			status:   http.StatusInternalServerError,
			id:       trafficRouteID,
			wantErr:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status := tc.status
			if status == 0 {
				status = http.StatusOK
			}
			cs := newControllerServer(t, route{trafficRoutesPath(), staticJSONStatus(status, tc.response)})

			got, err := cs.client().GetTrafficRoute(context.Background(), trafficRouteSite, tc.id)

			// Get must use the list endpoint, never GET /trafficroutes/<id>.
			req := cs.lastRequest()
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, trafficRoutesPath(), req.Path)
			assert.Equal(t, 1, cs.requestCount())

			switch {
			case tc.wantErrIs != nil:
				require.ErrorIs(t, err, tc.wantErrIs)
				assert.Nil(t, got)
			case tc.wantErr:
				require.Error(t, err)
				var serverErr *ServerError
				require.ErrorAs(t, err, &serverErr)
				assert.Nil(t, got)
			default:
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tc.id, got.ID)
				assert.Equal(t, tc.wantDesc, got.Description)
			}
		})
	}
}

func TestCreateTrafficRoute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		route    *TrafficRoute
		wantBody string
	}{
		"minimal internet route sends every clearable field": {
			route: &TrafficRoute{
				Description:    "Example egress",
				Enabled:        true,
				MatchingTarget: "INTERNET",
				NetworkID:      "00000000-0000-0000-0000-0000000000a1",
				TargetDevices:  []TrafficRouteTargetDevices{{Type: "ALL_CLIENTS"}},
			},
			wantBody: `{"description": "Example egress", "enabled": true, "kill_switch_enabled": false,
				"matching_target": "INTERNET", "network_id": "00000000-0000-0000-0000-0000000000a1", "next_hop": "",
				"domains": [], "ip_addresses": [], "ip_ranges": [], "regions": [], "target_devices": [{"type": "ALL_CLIENTS"}]}`,
		},
		"domain route with kill switch": {
			route: &TrafficRoute{
				Description:       "Example domains",
				Enabled:           true,
				KillSwitchEnabled: true,
				MatchingTarget:    "DOMAIN",
				NetworkID:         "00000000-0000-0000-0000-0000000000a1",
				Domains:           []TrafficRouteDomains{{Domain: "example.com", Ports: []int{443}}},
				TargetDevices:     []TrafficRouteTargetDevices{{ClientMAC: "00:00:5e:00:53:01", Type: "CLIENT"}},
			},
			wantBody: `{"description": "Example domains", "enabled": true, "kill_switch_enabled": true,
				"matching_target": "DOMAIN", "network_id": "00000000-0000-0000-0000-0000000000a1", "next_hop": "",
				"domains": [{"domain": "example.com", "port_ranges": [], "ports": [443]}],
				"ip_addresses": [], "ip_ranges": [], "regions": [],
				"target_devices": [{"client_mac": "00:00:5e:00:53:01", "type": "CLIENT"}]}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{trafficRoutesPath(), func(w http.ResponseWriter, r *http.Request) {
				// Echo the body back with a controller-assigned id.
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				body["_id"] = trafficRouteID
				writeJSON(t, w, body)
			}})

			got, err := cs.client().CreateTrafficRoute(context.Background(), trafficRouteSite, tc.route)
			require.NoError(t, err)

			req := cs.lastRequest()
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, trafficRoutesPath(), req.Path)
			assert.JSONEq(t, tc.wantBody, string(req.Body))

			require.NotNil(t, got)
			assert.Equal(t, trafficRouteID, got.ID)
			assert.Equal(t, tc.route.Description, got.Description)
			assert.Equal(t, tc.route.MatchingTarget, got.MatchingTarget)
		})
	}
}

func TestCreateTrafficRouteHardValidation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		route   *TrafficRoute
		wantErr bool
	}{
		"valid route passes": {
			route: &TrafficRoute{MatchingTarget: "REGION", NetworkID: "n", Regions: []string{"US"}},
		},
		"unknown matching target is rejected": {
			route:   &TrafficRoute{MatchingTarget: "EVERYTHING", NetworkID: "n"},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{trafficRoutesPath(), echoJSON})
			c := cs.clientWith(func(cfg *ClientConfig) { cfg.ValidationMode = HardValidation })

			_, err := c.CreateTrafficRoute(context.Background(), trafficRouteSite, tc.route)
			if tc.wantErr {
				require.ErrorContains(t, err, "failed validating request body")
				assert.Equal(t, 0, cs.countRequestsTo(trafficRoutesPath()), "an invalid body must not reach the controller")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 1, cs.countRequestsTo(trafficRoutesPath()))
		})
	}
}

func TestUpdateTrafficRoute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate   func(r *TrafficRoute)
		wantBody string
	}{
		"unchanged route is sent back verbatim": {
			mutate:   func(*TrafficRoute) {},
			wantBody: fullTrafficRouteJSON,
		},
		"clearing every clearable field sends empty values": {
			mutate: func(r *TrafficRoute) {
				r.Description = ""
				r.Domains = nil
				r.IPAddresses = []TrafficRouteIPAddresses{}
				r.IPRanges = nil
				r.Regions = nil
				r.TargetDevices = nil
				r.NextHop = ""
				r.KillSwitchEnabled = false
				r.Enabled = false
				r.MatchingTarget = "INTERNET"
			},
			wantBody: `{"_id": "00000000-0000-0000-0000-000000000001", "description": "", "enabled": false,
				"kill_switch_enabled": false, "matching_target": "INTERNET", "network_id": "00000000-0000-0000-0000-0000000000a1",
				"next_hop": "", "domains": [], "ip_addresses": [], "ip_ranges": [], "regions": [], "target_devices": []}`,
		},
		"clearing nested ports sends empty lists": {
			mutate: func(r *TrafficRoute) {
				r.Domains[0].Ports = nil
				r.Domains[0].PortRanges = nil
				r.IPAddresses = r.IPAddresses[:1]
				r.IPAddresses[0].Ports = []int{}
				r.IPAddresses[0].PortRanges = nil
			},
			wantBody: `{
				"_id": "00000000-0000-0000-0000-000000000001",
				"description": "Example route",
				"domains": [{"domain": "example.com", "port_ranges": [], "ports": []}],
				"enabled": true,
				"ip_addresses": [{"ip_or_subnet": "192.0.2.0/24", "ip_version": "v4", "port_ranges": [], "ports": []}],
				"ip_ranges": [{"ip_start": "198.51.100.10", "ip_stop": "198.51.100.20", "ip_version": "v4"}],
				"kill_switch_enabled": true,
				"matching_target": "IP",
				"network_id": "00000000-0000-0000-0000-0000000000a1",
				"next_hop": "192.0.2.1",
				"regions": ["US", "CA"],
				"target_devices": [
					{"client_mac": "00:00:5e:00:53:01", "type": "CLIENT"},
					{"network_id": "00000000-0000-0000-0000-0000000000b1", "type": "NETWORK"},
					{"type": "ALL_CLIENTS"}
				]
			}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{trafficRouteByIDPath(), echoJSON})

			var r TrafficRoute
			require.NoError(t, json.Unmarshal([]byte(fullTrafficRouteJSON), &r))
			tc.mutate(&r)

			got, err := cs.client().UpdateTrafficRoute(context.Background(), trafficRouteSite, &r)
			require.NoError(t, err)

			req := cs.lastRequest()
			assert.Equal(t, http.MethodPut, req.Method)
			assert.Equal(t, trafficRouteByIDPath(), req.Path)
			assert.JSONEq(t, tc.wantBody, string(req.Body))

			require.NotNil(t, got)
			assert.Equal(t, trafficRouteID, got.ID)
			assert.Equal(t, r.Description, got.Description)
		})
	}
}

func TestDeleteTrafficRoute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status  int
		wantErr bool
	}{
		"deleted":              {status: http.StatusOK},
		"server error":         {status: http.StatusInternalServerError, wantErr: true},
		"unknown id not found": {status: http.StatusNotFound, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{trafficRouteByIDPath(), func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			}})

			err := cs.client().DeleteTrafficRoute(context.Background(), trafficRouteSite, trafficRouteID)

			req := cs.lastRequest()
			assert.Equal(t, http.MethodDelete, req.Method)
			assert.Equal(t, trafficRouteByIDPath(), req.Path)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
