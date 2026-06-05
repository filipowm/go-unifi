package unifi //nolint: testpackage

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSetting exercises the key-search loop in GetSetting: the requested key is
// located among several settings and decoded into its concrete fields type; a
// missing key maps to ErrNotFound; an unknown (unregistered) key surfaces the
// newFields "unexpected key" error.
func TestGetSetting(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/get/setting")

	cases := map[string]struct {
		key        string
		response   string
		wantErr    bool
		wantErrIs  error
		wantErrMsg string
		assertOK   func(t *testing.T, s *Setting, fields any)
	}{
		"key found decodes typed fields": {
			key:      SettingNtpKey,
			response: `{"meta":{"rc":"ok"},"data":[{"_id":"x","key":"country"},{"_id":"y","key":"ntp","ntp_server_1":"pool.ntp.org"}]}`,
			assertOK: func(t *testing.T, s *Setting, fields any) {
				t.Helper()
				assert.Equal(t, SettingNtpKey, s.Key)
				ntp, ok := fields.(*SettingNtp)
				require.True(t, ok, "fields must be a *SettingNtp, got %T", fields)
				assert.Equal(t, "pool.ntp.org", ntp.NtpServer1)
			},
		},
		"key absent yields ErrNotFound": {
			key:       SettingNtpKey,
			response:  `{"meta":{"rc":"ok"},"data":[{"_id":"x","key":"country"}]}`,
			wantErr:   true,
			wantErrIs: ErrNotFound,
		},
		"unknown key surfaces newFields error": {
			key:        "definitely-not-real",
			response:   `{"meta":{"rc":"ok"},"data":[{"_id":"x","key":"definitely-not-real"}]}`,
			wantErr:    true,
			wantErrMsg: `unexpected key "definitely-not-real"`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.response))
			}})
			c := cs.client()

			s, fields, err := c.GetSetting(context.Background(), site, tc.key)

			assert.Equal(t, http.MethodGet, cs.lastRequest().Method)
			assert.Equal(t, path, cs.lastRequest().Path)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, s)
				assert.Nil(t, fields)
				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs)
				}
				if tc.wantErrMsg != "" {
					require.ErrorContains(t, err, tc.wantErrMsg)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, s)
			tc.assertOK(t, s, fields)
		})
	}
}

// TestSetSetting exercises the SetSetting wrapper: it must PUT to the per-key
// set/setting path, then locate the matching key in the response and decode it
// into the concrete fields type. The key-absent branch maps to ErrNotFound.
func TestSetSetting(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/set/setting/" + SettingNtpKey)

	cases := map[string]struct {
		response  string
		wantErrIs error
		assertOK  func(t *testing.T, fields any)
	}{
		"key found decodes typed fields": {
			response: `{"meta":{"rc":"ok"},"data":[{"_id":"y","key":"ntp","ntp_server_2":"time.google.com"}]}`,
			assertOK: func(t *testing.T, fields any) {
				t.Helper()
				ntp, ok := fields.(*SettingNtp)
				require.True(t, ok, "fields must be a *SettingNtp, got %T", fields)
				assert.Equal(t, "time.google.com", ntp.NtpServer2)
			},
		},
		"key absent yields ErrNotFound": {
			response:  `{"meta":{"rc":"ok"},"data":[{"_id":"x","key":"country"}]}`,
			wantErrIs: ErrNotFound,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.response))
			}})
			c := cs.client()

			fields, err := c.SetSetting(context.Background(), site, SettingNtpKey, &SettingNtp{Key: SettingNtpKey})

			req := cs.lastRequest()
			assert.Equal(t, http.MethodPut, req.Method, "SetSetting must use PUT")
			assert.Equal(t, path, req.Path)

			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
				assert.Nil(t, fields)
				return
			}
			require.NoError(t, err)
			tc.assertOK(t, fields)
		})
	}
}

// TestGetSettingErrNotFoundSurvivesWrap is the TEST-05 sentinel-survival guard: a
// real HTTP 404 from the controller flows through GetSetting's
// `fmt.Errorf("unable to get setting %s: %w", ...)` wrap, and errors.Is must
// still resolve it to the ErrNotFound sentinel (via ServerError.Is mapping 404 ->
// ErrNotFound). This proves the sentinel survives a %w wrap chain rather than
// only being returned bare.
func TestGetSettingErrNotFoundSurvivesWrap(t *testing.T) {
	t.Parallel()

	const site = "default"
	path := apiV1Path("s/" + site + "/get/setting")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		// A genuine 404 (not the empty-data 200 case) so the sentinel must come
		// from the ServerError.Is mapping, then traverse the wrapper's %w wrap.
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"meta":{"rc":"error","msg":"not found"}}`))
	}})
	c := cs.client()

	s, fields, err := c.GetSetting(context.Background(), site, SettingNtpKey)
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Nil(t, fields)

	// The error must be a wrapped (not bare) ErrNotFound: it carries the wrapper's
	// context message AND still satisfies errors.Is against the sentinel.
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorContains(t, err, "unable to get setting")

	// Belt-and-suspenders: a *ServerError is reachable in the chain too, confirming
	// the 404 origin of the sentinel rather than a bare-return ErrNotFound.
	var se *ServerError
	require.ErrorAs(t, err, &se, "expected a *ServerError in the chain")
	assert.Equal(t, http.StatusNotFound, se.StatusCode)

	// Sanity: the bare sentinel itself obviously matches, so the assertion above is
	// meaningful only because the wrap preserved it.
	require.ErrorIs(t, fmt.Errorf("ctx: %w", ErrNotFound), ErrNotFound)
}
