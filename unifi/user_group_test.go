package unifi //nolint: testpackage

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserGroupCreateUpdateNotFoundContract is the BEHAVIORAL guard for the
// template-generated create/update path. The golden text in
// codegen/testdata/widget_v1.golden pins the *rendered* fmt.Errorf, but nothing
// asserted the *runtime* semantics: a successful-but-wrong-shape create/update
// (HTTP 200, top-level meta rc:ok, but data not length 1) must NOT collapse into
// ErrNotFound — that sentinel is reserved for the get/list-single path. Instead
// it must surface a descriptive "unexpected response: expected 1 X, got N" error.
//
// Without this test the contract survives only until the next intentional
// template change forces a golden regen, at which point a reviewer regenerating
// could silently re-bless a reintroduced ErrNotFound because nothing pins the
// behavior. We drive it through the clean template-generated UserGroup wrappers
// (CreateUserGroup / UpdateUserGroup are pure pass-throughs to the generated
// createUserGroup / updateUserGroup), NOT the user.go wrapper, which has its own
// nested-envelope semantics already covered by TestCreateUser.
//
// The companion get-path assertion confirms the deliberate split: the SAME
// wrong-count payload on getUserGroup DOES yield ErrNotFound, so this test pins
// both halves of the contract.
func TestUserGroupCreateUpdateNotFoundContract(t *testing.T) {
	t.Parallel()

	const (
		site = "default"
		id   = "ug1"
	)
	createPath := apiV1Path("s/" + site + "/rest/usergroup")
	idPath := apiV1Path("s/" + site + "/rest/usergroup/" + id)

	// call invokes a single create/update/get wrapper against a mock controller
	// that returns the given raw JSON body with HTTP 200.
	type result struct {
		got *UserGroup
		err error
	}

	cases := map[string]struct {
		// op selects which wrapper to drive.
		op string
		// path is the endpoint the wrapper is expected to hit.
		path string
		// response is the raw JSON the mock controller returns (HTTP 200).
		response string
		// wantUnexpected asserts the "unexpected response" error and that
		// the error is NOT ErrNotFound.
		wantUnexpected bool
		// wantNotFound asserts ErrNotFound (the get-path contract).
		wantNotFound bool
		// wantID asserts the happy-path echoed resource id.
		wantID string
	}{
		// --- create: wrong count must be "unexpected response", NOT ErrNotFound ---
		"create empty data is unexpected, not ErrNotFound": {
			op:             "create",
			path:           createPath,
			response:       `{"meta":{"rc":"ok"},"data":[]}`,
			wantUnexpected: true,
		},
		"create two rows is unexpected, not ErrNotFound": {
			op:             "create",
			path:           createPath,
			response:       `{"meta":{"rc":"ok"},"data":[{"_id":"a"},{"_id":"b"}]}`,
			wantUnexpected: true,
		},
		"create exactly one row succeeds": {
			op:       "create",
			path:     createPath,
			response: `{"meta":{"rc":"ok"},"data":[{"_id":"ug1","name":"grp"}]}`,
			wantID:   "ug1",
		},
		// --- update: same contract on the PUT path ---
		"update empty data is unexpected, not ErrNotFound": {
			op:             "update",
			path:           idPath,
			response:       `{"meta":{"rc":"ok"},"data":[]}`,
			wantUnexpected: true,
		},
		"update two rows is unexpected, not ErrNotFound": {
			op:             "update",
			path:           idPath,
			response:       `{"meta":{"rc":"ok"},"data":[{"_id":"a"},{"_id":"b"}]}`,
			wantUnexpected: true,
		},
		"update exactly one row succeeds": {
			op:       "update",
			path:     idPath,
			response: `{"meta":{"rc":"ok"},"data":[{"_id":"ug1","name":"grp"}]}`,
			wantID:   "ug1",
		},
		// --- get: the deliberate counterpart — wrong count IS ErrNotFound ---
		"get empty data yields ErrNotFound": {
			op:           "get",
			path:         idPath,
			response:     `{"meta":{"rc":"ok"},"data":[]}`,
			wantNotFound: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs := newControllerServer(t, route{tc.path, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.response))
			}})
			c := cs.client()

			var res result
			switch tc.op {
			case "create":
				res.got, res.err = c.CreateUserGroup(context.Background(), site, &UserGroup{Name: "grp"})
			case "update":
				res.got, res.err = c.UpdateUserGroup(context.Background(), site, &UserGroup{ID: id, Name: "grp"})
			case "get":
				res.got, res.err = c.GetUserGroup(context.Background(), site, id)
			default:
				t.Fatalf("unknown op %q", tc.op)
			}

			assert.Equal(t, tc.path, cs.lastRequest().Path)

			switch {
			case tc.wantUnexpected:
				assert.Nil(t, res.got)
				// The whole point: a wrong-count create/update is NOT
				// a not-found, it is an unexpected response shape.
				require.NotErrorIs(t, res.err, ErrNotFound)
				require.ErrorContains(t, res.err, "unexpected response: expected 1")
				require.ErrorContains(t, res.err, "UserGroup")
			case tc.wantNotFound:
				assert.Nil(t, res.got)
				require.ErrorIs(t, res.err, ErrNotFound)
			default:
				require.NoError(t, res.err)
				require.NotNil(t, res.got)
				assert.Equal(t, tc.wantID, res.got.ID)
			}
		})
	}
}
