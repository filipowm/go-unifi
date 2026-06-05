package unifi //nolint: testpackage

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeJSON is a small helper that marshals v and writes it as the response
// body. It sets a non-zero Content-Length implicitly (httptest buffers the
// body), which matters because handleResponse skips decoding on ContentLength==0.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(v))
}

// TestCreateUser exercises the branchy nested-group response handling in the
// hand-written CreateUser wrapper: success, the malformed-group length guard, the
// Meta.error() failure path, and the ErrNotFound on an inner len != 1.
func TestCreateUser(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/group/user")

	type testCase struct {
		// response is the raw JSON the mock controller returns.
		response string
		// status is the HTTP status; defaults to 200 when zero.
		status   int
		wantName string
		wantErr  bool
		// wantErrIs, when set, asserts errors.Is(err, wantErrIs).
		wantErrIs error
		// wantErrMsg, when set, asserts the error message contains the substring.
		wantErrMsg string
		// wantServerErr, when true, asserts the error is a *ServerError (and is NOT
		// the ErrNotFound sentinel).
		wantServerErr bool
	}

	cases := map[string]testCase{
		"success returns inner user": {
			response: `{"meta":{"rc":"ok"},"data":[{"Meta":{"rc":"ok"},"data":[{"_id":"u1","name":"alice"}]}]}`,
			wantName: "alice",
		},
		"malformed group: outer data not length 1": {
			response:   `{"meta":{"rc":"ok"},"data":[]}`,
			wantErr:    true,
			wantErrMsg: "malformed group response",
		},
		// ARCH-10/O5: the soft (HTTP 200) meta rc:error check is now centralized in
		// handleResponse and gated on the TOP-LEVEL meta envelope. A top-level
		// meta.rc=="error" is surfaced as a *ServerError carrying the rc/msg.
		"top-level Meta error is surfaced": {
			response:   `{"meta":{"rc":"error","msg":"api.err.Invalid"},"data":[]}`,
			wantErr:    true,
			wantErrMsg: "api.err.Invalid",
		},
		// ARCH-10 (restored): the centralized top-level rc:error check does not see
		// the NESTED per-object meta (data[0].Meta). CreateUser keeps its own nested
		// Meta.error() check, so a nested rc=="error" with empty inner data surfaces
		// a *ServerError carrying the inner rc/msg — NOT ErrNotFound. This preserves
		// the pre-Wave-2 behavior (eliminating the previously-documented
		// ARCH-10-user breaking change).
		"inner Meta error is surfaced as ServerError": {
			response:      `{"meta":{"rc":"ok"},"data":[{"Meta":{"rc":"error","msg":"api.err.Invalid"},"data":[]}]}`,
			wantErr:       true,
			wantServerErr: true,
			wantErrMsg:    "api.err.Invalid",
		},
		// Genuine empty inner WITHOUT a nested error (inner rc absent, inner len 0)
		// still maps to ErrNotFound.
		"inner data not length 1 yields ErrNotFound": {
			response:  `{"meta":{"rc":"ok"},"data":[{"Meta":{"rc":"ok"},"data":[]}]}`,
			wantErr:   true,
			wantErrIs: ErrNotFound,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				_, _ = w.Write([]byte(tc.response))
			}})
			c := cs.client()

			got, err := c.CreateUser(context.Background(), site, &User{Name: "alice"})

			// The wrapper must POST to the group endpoint regardless of outcome.
			assert.Equal(t, http.MethodPost, cs.lastRequest().Method)
			assert.Equal(t, path, cs.lastRequest().Path)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs)
				}
				if tc.wantServerErr {
					var serverErr *ServerError
					require.ErrorAs(t, err, &serverErr)
					require.NotErrorIs(t, err, ErrNotFound, "a nested soft rc:error is NOT a 404")
				}
				if tc.wantErrMsg != "" {
					require.ErrorContains(t, err, tc.wantErrMsg)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantName, got.Name)
		})
	}
}

// TestCreateUserSendsNestedObjectsBody asserts the wrapper wraps the user in the
// {objects:[{data:...}]} envelope the controller expects.
func TestCreateUserSendsNestedObjectsBody(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/group/user")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"Meta":{"rc":"ok"},"data":[{"name":"bob"}]}]}`))
	}})
	c := cs.client()

	_, err := c.CreateUser(context.Background(), site, &User{Name: "bob"})
	require.NoError(t, err)

	var sent struct {
		Objects []struct {
			Data User `json:"data"`
		} `json:"objects"`
	}
	require.NoError(t, json.Unmarshal(cs.lastRequest().Body, &sent))
	require.Len(t, sent.Objects, 1)
	assert.Equal(t, "bob", sent.Objects[0].Data.Name)
}

// TestOverrideUserFingerprint asserts the DELETE-vs-PUT method selection keyed on
// devIdOverride and that the request targets the v2 fingerprint_override path.
func TestOverrideUserFingerprint(t *testing.T) {
	t.Parallel()

	const (
		site = "default"
		mac  = "00:11:22:33:44:55"
	)
	path := apiV2("site/" + site + "/station/" + mac + "/fingerprint_override")

	cases := map[string]struct {
		devIDOverride int
		wantMethod    string
	}{
		"zero override deletes": {devIDOverride: 0, wantMethod: http.MethodDelete},
		"nonzero override puts": {devIDOverride: 42, wantMethod: http.MethodPut},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(t, w, map[string]any{"mac": mac, "dev_id_override": tc.devIDOverride, "search_query": ""})
			}})
			c := cs.client()

			err := c.OverrideUserFingerprint(context.Background(), site, mac, tc.devIDOverride)
			require.NoError(t, err)

			req := cs.lastRequest()
			assert.Equal(t, tc.wantMethod, req.Method)
			assert.Equal(t, path, req.Path, "must hit the v2 fingerprint_override path")

			// The request body must carry the override fields.
			var body map[string]any
			require.NoError(t, json.Unmarshal(req.Body, &body))
			assert.Equal(t, mac, body["mac"])
			assert.EqualValues(t, tc.devIDOverride, body["dev_id_override"])
		})
	}
}

// TestBlockUserByMAC asserts the stamgr command body (cmd=block-sta, mac=...) and
// the success / ErrNotFound branches keyed on the returned user count.
func TestBlockUserByMAC(t *testing.T) {
	t.Parallel()

	const (
		site = "default"
		mac  = "aa:bb:cc:dd:ee:ff"
	)
	path := apiV1Path("s/" + site + "/cmd/stamgr")

	cases := map[string]struct {
		response  string
		wantErrIs error
	}{
		"one user blocked": {response: `{"meta":{"rc":"ok"},"data":[{"_id":"u1"}]}`},
		"no user found":    {response: `{"meta":{"rc":"ok"},"data":[]}`, wantErrIs: ErrNotFound},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.response))
			}})
			c := cs.client()

			err := c.BlockUserByMAC(context.Background(), site, mac)

			req := cs.lastRequest()
			assert.Equal(t, http.MethodPost, req.Method)
			var body map[string]any
			require.NoError(t, json.Unmarshal(req.Body, &body))
			assert.Equal(t, "block-sta", body["cmd"])
			assert.Equal(t, mac, body["mac"])

			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDeleteUserByMAC asserts the forget-sta command sends the MAC inside a macs
// array (not a bare mac field).
func TestDeleteUserByMAC(t *testing.T) {
	t.Parallel()

	const (
		site = "default"
		mac  = "aa:bb:cc:dd:ee:ff"
	)
	path := apiV1Path("s/" + site + "/cmd/stamgr")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"_id":"u1"}]}`))
	}})
	c := cs.client()

	require.NoError(t, c.DeleteUserByMAC(context.Background(), site, mac))

	var body struct {
		Cmd  string   `json:"cmd"`
		MACs []string `json:"macs"`
	}
	require.NoError(t, json.Unmarshal(cs.lastRequest().Body, &body))
	assert.Equal(t, "forget-sta", body.Cmd)
	assert.Equal(t, []string{mac}, body.MACs)
}

// TestListUserUnwrapsEnvelope asserts ListUser unwraps the {meta,data} envelope
// and returns the inner slice with multiple elements intact.
func TestListUserUnwrapsEnvelope(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/rest/user")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"_id":"u1","name":"alice"},{"_id":"u2","name":"bob"}]}`))
	}})
	c := cs.client()

	users, err := c.ListUser(context.Background(), site)
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, "alice", users[0].Name)
	assert.Equal(t, "bob", users[1].Name)
	assert.Equal(t, http.MethodGet, cs.lastRequest().Method)
}

// TestGetUserNotFound asserts the getUser len != 1 guard maps to ErrNotFound, and
// that the sentinel survives the wrapper's %w wrap chain (TEST-05 guard).
func TestGetUserNotFound(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/rest/user/missing")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
	}})
	c := cs.client()

	got, err := c.GetUser(context.Background(), site, "missing")
	assert.Nil(t, got)
	require.ErrorIs(t, err, ErrNotFound)
}
