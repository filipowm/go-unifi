package unifi //nolint: testpackage

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDevice asserts the linear search in GetDevice: a matching ID is returned
// and a miss maps to ErrNotFound (via the %w-safe sentinel).
func TestGetDevice(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/stat/device")
	listBody := `{"meta":{"rc":"ok"},"data":[{"_id":"d1"},{"_id":"d2"}]}`

	cases := map[string]struct {
		id        string
		wantID    string
		wantErrIs error
	}{
		"found by id":     {id: "d2", wantID: "d2"},
		"missing id 404s": {id: "nope", wantErrIs: ErrNotFound},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(listBody))
			}})
			c := cs.client()

			got, err := c.GetDevice(context.Background(), site, tc.id)
			if tc.wantErrIs != nil {
				assert.Nil(t, got)
				require.ErrorIs(t, err, tc.wantErrIs)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantID, got.ID)
		})
	}
}

// TestListDeviceUnwrapsEnvelope asserts ListDevice unwraps the {meta,data}
// envelope to the inner slice.
func TestListDeviceUnwrapsEnvelope(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/stat/device")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"_id":"d1"},{"_id":"d2"},{"_id":"d3"}]}`))
	}})
	c := cs.client()

	devices, err := c.ListDevice(context.Background(), site)
	require.NoError(t, err)
	require.Len(t, devices, 3)
	assert.Equal(t, "d1", devices[0].ID)
	assert.Equal(t, http.MethodGet, cs.lastRequest().Method)
}

// TestAdoptDevice asserts the devmgr command body carries cmd=adopt and the MAC,
// and that the request is a POST to the devmgr endpoint.
func TestAdoptDevice(t *testing.T) {
	t.Parallel()

	const (
		site = "default"
		mac  = "00:11:22:33:44:55"
	)
	path := apiV1Path("s/" + site + "/cmd/devmgr")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"}}`))
	}})
	c := cs.client()

	require.NoError(t, c.AdoptDevice(context.Background(), site, mac))

	req := cs.lastRequest()
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, path, req.Path)

	var body struct {
		Cmd string `json:"cmd"`
		MAC string `json:"mac"`
	}
	require.NoError(t, json.Unmarshal(req.Body, &body))
	assert.Equal(t, "adopt", body.Cmd)
	assert.Equal(t, mac, body.MAC)
}

// TestDeleteDevicePropagatesMethod asserts the generated deleteDevice wrapper
// issues a DELETE to the rest/device endpoint.
func TestDeleteDevicePropagatesMethod(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/rest/device/d1")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}})
	c := cs.client()

	require.NoError(t, c.DeleteDevice(context.Background(), site, "d1"))
	assert.Equal(t, http.MethodDelete, cs.lastRequest().Method)
	assert.Equal(t, path, cs.lastRequest().Path)
}
