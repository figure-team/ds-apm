package coderca

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func TestResolveServiceRepo(t *testing.T) {
	maps := []ruletypes.CodebaseServiceMap{
		{OrgID: "org1", ServiceName: "payments", RepoID: "repo-pay"},
		{OrgID: "org1", ServiceName: "orders", RepoID: "repo-mono", Subpath: "services/orders"},
		{OrgID: "org2", ServiceName: "payments", RepoID: "repo-pay-2"},
	}

	tests := []struct {
		name        string
		orgID       string
		service     string
		wantOK      bool
		wantRepoID  string
		wantSubpath string
	}{
		{name: "exact match", orgID: "org1", service: "payments", wantOK: true, wantRepoID: "repo-pay"},
		{name: "monorepo subpath returned", orgID: "org1", service: "orders", wantOK: true, wantRepoID: "repo-mono", wantSubpath: "services/orders"},
		{name: "tenant isolation: same service, other org", orgID: "org2", service: "payments", wantOK: true, wantRepoID: "repo-pay-2"},
		{name: "unmapped service skips", orgID: "org1", service: "billing", wantOK: false},
		{name: "mapped service under wrong org does not match", orgID: "org3", service: "payments", wantOK: false},
		{name: "empty service never resolves", orgID: "org1", service: "", wantOK: false},
		{name: "service name is trimmed on lookup", orgID: "org1", service: "  payments  ", wantOK: true, wantRepoID: "repo-pay"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveServiceRepo(maps, tc.orgID, tc.service)
			if ok != tc.wantOK {
				t.Fatalf("ResolveServiceRepo() ok = %v, want %v", ok, tc.wantOK)
			}
			if got.RepoID != tc.wantRepoID {
				t.Errorf("RepoID = %q, want %q", got.RepoID, tc.wantRepoID)
			}
			if got.Subpath != tc.wantSubpath {
				t.Errorf("Subpath = %q, want %q", got.Subpath, tc.wantSubpath)
			}
		})
	}
}

func TestResolveServiceRepoEmptyMaps(t *testing.T) {
	if _, ok := ResolveServiceRepo(nil, "org1", "payments"); ok {
		t.Error("ResolveServiceRepo() over nil maps must not resolve")
	}
}
