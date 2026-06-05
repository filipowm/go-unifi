package unifi //nolint: testpackage

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListSitesUnwrapsEnvelope asserts ListSites returns the inner data slice
// from the {meta,data} envelope with all elements intact.
func TestListSitesUnwrapsEnvelope(t *testing.T) {
	t.Parallel()

	path := apiV1Path("self/sites")
	cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"_id":"s1","name":"default","desc":"Default"},{"_id":"s2","name":"site2","desc":"Second"}]}`))
	}})
	c := cs.client()

	sites, err := c.ListSites(context.Background())
	require.NoError(t, err)
	require.Len(t, sites, 2)
	assert.Equal(t, "default", sites[0].Name)
	assert.Equal(t, "Second", sites[1].Description)
	assert.Equal(t, http.MethodGet, cs.lastRequest().Method)
}

// TestGetSite asserts the linear search over the sites list: a matching _id is
// returned, a miss maps to the %w-safe ErrNotFound.
func TestGetSite(t *testing.T) {
	t.Parallel()

	path := apiV1Path("self/sites")
	listBody := `{"meta":{"rc":"ok"},"data":[{"_id":"s1","name":"default"},{"_id":"s2","name":"site2"}]}`

	cases := map[string]struct {
		id        string
		wantName  string
		wantErrIs error
	}{
		"found by id":  {id: "s2", wantName: "site2"},
		"missing 404s": {id: "ghost", wantErrIs: ErrNotFound},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{path, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(listBody))
			}})
			c := cs.client()

			got, err := c.GetSite(context.Background(), tc.id)
			if tc.wantErrIs != nil {
				assert.Nil(t, got)
				require.ErrorIs(t, err, tc.wantErrIs)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantName, got.Name)
		})
	}
}
