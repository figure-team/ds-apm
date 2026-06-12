package reporesolver

import (
	"context"
	"errors"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// fakeMaps implements ruletypes.CodebaseServiceMapStore in memory.
type fakeMaps struct {
	rows    []ruletypes.CodebaseServiceMap
	listErr error
}

func (f *fakeMaps) Upsert(context.Context, ruletypes.CodebaseServiceMap) error { return nil }
func (f *fakeMaps) Get(context.Context, string, string) (ruletypes.CodebaseServiceMap, error) {
	return ruletypes.CodebaseServiceMap{}, ruletypes.ErrCodebaseServiceMapNotFound
}
func (f *fakeMaps) List(_ context.Context, orgID string) ([]ruletypes.CodebaseServiceMap, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []ruletypes.CodebaseServiceMap
	for _, m := range f.rows {
		if m.OrgID == orgID {
			out = append(out, m)
		}
	}
	return out, nil
}

// fakeRepos implements ruletypes.CodebaseRepoStore in memory.
type fakeRepos struct {
	rows map[string]ruletypes.CodebaseRepo // key: org/repoID
}

func (f *fakeRepos) Upsert(context.Context, ruletypes.CodebaseRepo, func(string) (string, error)) error {
	return nil
}
func (f *fakeRepos) List(context.Context, string, func(string) (string, error)) ([]ruletypes.CodebaseRepo, error) {
	return nil, nil
}
func (f *fakeRepos) Get(_ context.Context, orgID, repoID string, decrypt func(string) (string, error)) (ruletypes.CodebaseRepo, error) {
	r, ok := f.rows[orgID+"/"+repoID]
	if !ok {
		return ruletypes.CodebaseRepo{}, ruletypes.ErrCodebaseRepoNotFound
	}
	cred, err := decrypt(r.Credential)
	if err != nil {
		return ruletypes.CodebaseRepo{}, err
	}
	r.Credential = cred
	return r, nil
}

func decShout(s string) (string, error) { return "DEC:" + s, nil }

func TestResolveRepoHappyPath(t *testing.T) {
	maps := &fakeMaps{rows: []ruletypes.CodebaseServiceMap{
		{OrgID: "org1", ServiceName: "payments", RepoID: "repo-pay", Subpath: "svc/pay"},
	}}
	repos := &fakeRepos{rows: map[string]ruletypes.CodebaseRepo{
		"org1/repo-pay": {OrgID: "org1", RepoID: "repo-pay", Enabled: true, Credential: "ct"},
	}}
	r := New(maps, repos, decShout)

	repo, subpath, ok, err := r.ResolveRepo(context.Background(), "org1", "payments")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if repo.RepoID != "repo-pay" {
		t.Errorf("RepoID = %q, want repo-pay", repo.RepoID)
	}
	if subpath != "svc/pay" {
		t.Errorf("subpath = %q, want svc/pay", subpath)
	}
	if repo.Credential != "DEC:ct" {
		t.Errorf("credential not decrypted: %q", repo.Credential)
	}
}

func TestResolveRepoUnmapped(t *testing.T) {
	r := New(&fakeMaps{}, &fakeRepos{}, decShout)
	_, _, ok, err := r.ResolveRepo(context.Background(), "org1", "billing")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("ok = true for unmapped service, want false")
	}
}

func TestResolveRepoDanglingMapping(t *testing.T) {
	maps := &fakeMaps{rows: []ruletypes.CodebaseServiceMap{
		{OrgID: "org1", ServiceName: "payments", RepoID: "gone"},
	}}
	r := New(maps, &fakeRepos{rows: map[string]ruletypes.CodebaseRepo{}}, decShout)
	_, _, ok, err := r.ResolveRepo(context.Background(), "org1", "payments")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("ok = true for a mapping pointing at a missing repo, want false")
	}
}

func TestResolveRepoDisabledRepo(t *testing.T) {
	maps := &fakeMaps{rows: []ruletypes.CodebaseServiceMap{
		{OrgID: "org1", ServiceName: "payments", RepoID: "repo-pay"},
	}}
	repos := &fakeRepos{rows: map[string]ruletypes.CodebaseRepo{
		"org1/repo-pay": {OrgID: "org1", RepoID: "repo-pay", Enabled: false},
	}}
	r := New(maps, repos, decShout)
	_, _, ok, err := r.ResolveRepo(context.Background(), "org1", "payments")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("ok = true for a disabled repo, want false")
	}
}

func TestResolveRepoCrossTenant(t *testing.T) {
	maps := &fakeMaps{rows: []ruletypes.CodebaseServiceMap{
		{OrgID: "org2", ServiceName: "payments", RepoID: "repo-pay"},
	}}
	repos := &fakeRepos{rows: map[string]ruletypes.CodebaseRepo{
		"org2/repo-pay": {OrgID: "org2", RepoID: "repo-pay", Enabled: true},
	}}
	r := New(maps, repos, decShout)
	_, _, ok, err := r.ResolveRepo(context.Background(), "org1", "payments")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("ok = true resolving across tenants, want false")
	}
}

func TestResolveRepoStoreErrorPropagates(t *testing.T) {
	maps := &fakeMaps{listErr: errors.New("db down")}
	r := New(maps, &fakeRepos{}, decShout)
	if _, _, _, err := r.ResolveRepo(context.Background(), "org1", "payments"); err == nil {
		t.Error("store error must propagate")
	}
}
