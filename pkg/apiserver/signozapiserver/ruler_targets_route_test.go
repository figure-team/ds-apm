package signozapiserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestRemediationTargetRouteOrdering locks the registration-order invariant that
// addRulerRoutes relies on (design §3.1): gorilla/mux matches in registration
// order, so the /remediation/targets routes and their fixed sub-paths
// (keygen/fingerprint/test) MUST be registered before the /remediation/{id} and
// /remediation/targets/{targetId} wildcard routes. This test registers the same
// patterns in the same relative order with sentinel handlers and asserts each
// request resolves to the intended route — if someone reorders the real routes so
// a wildcard shadows a fixed path, the mirrored ordering here documents the
// contract that must hold.
func TestRemediationTargetRouteOrdering(t *testing.T) {
	router := mux.NewRouter()
	register := func(pattern, id, method string) {
		router.HandleFunc(pattern, func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte(id))
		}).Methods(method)
	}

	// Same order as addRulerRoutes: fixed paths first, wildcards last.
	register("/api/v2/ds/remediation/config", "GetRemediationConfig", http.MethodGet)
	register("/api/v2/ds/remediation/targets", "ListRemediationTargets", http.MethodGet)
	register("/api/v2/ds/remediation/targets", "CreateRemediationTarget", http.MethodPost)
	register("/api/v2/ds/remediation/targets/keygen", "KeygenRemediationTarget", http.MethodPost)
	register("/api/v2/ds/remediation/targets/fingerprint", "FingerprintRemediationTarget", http.MethodPost)
	register("/api/v2/ds/remediation/targets/test", "TestRemediationTarget", http.MethodPost)
	register("/api/v2/ds/remediation/targets/{targetId}", "UpdateRemediationTarget", http.MethodPut)
	register("/api/v2/ds/remediation/targets/{targetId}", "DeleteRemediationTarget", http.MethodDelete)
	register("/api/v2/ds/remediation/{id}", "GetRemediation", http.MethodGet)

	cases := []struct {
		method, path, wantID string
	}{
		{http.MethodGet, "/api/v2/ds/remediation/targets", "ListRemediationTargets"},
		{http.MethodPost, "/api/v2/ds/remediation/targets", "CreateRemediationTarget"},
		{http.MethodPost, "/api/v2/ds/remediation/targets/keygen", "KeygenRemediationTarget"},
		{http.MethodPost, "/api/v2/ds/remediation/targets/fingerprint", "FingerprintRemediationTarget"},
		{http.MethodPost, "/api/v2/ds/remediation/targets/test", "TestRemediationTarget"},
		{http.MethodPut, "/api/v2/ds/remediation/targets/abc-123", "UpdateRemediationTarget"},
		{http.MethodDelete, "/api/v2/ds/remediation/targets/abc-123", "DeleteRemediationTarget"},
		// The wildcard {id} must NOT swallow /targets.
		{http.MethodGet, "/api/v2/ds/remediation/rem-42", "GetRemediation"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rw := httptest.NewRecorder()
			router.ServeHTTP(rw, httptest.NewRequest(tc.method, tc.path, http.NoBody))
			if got := rw.Body.String(); got != tc.wantID {
				t.Fatalf("route %s %s: matched %q, want %q", tc.method, tc.path, got, tc.wantID)
			}
		})
	}
}
