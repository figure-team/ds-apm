package sourcestate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type fakeGit struct {
	fetchErr      error
	head          string
	resolveErr    error
	fetchCalls    int
	resolveCalls  int
	resolveBranch string
}

func (f *fakeGit) Fetch(ctx context.Context, repo ruletypes.CodebaseRepo) error {
	f.fetchCalls++
	return f.fetchErr
}

func (f *fakeGit) ResolveHead(ctx context.Context, repo ruletypes.CodebaseRepo, branch string) (string, error) {
	f.resolveCalls++
	f.resolveBranch = branch
	return f.head, f.resolveErr
}

func fixedNow() time.Time { return time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC) }

func TestSyncerHappyPath(t *testing.T) {
	git := &fakeGit{head: "deadbeef"}
	s := NewSyncer(git, fixedNow)
	repo := ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1", DefaultBranch: "main"}

	got := s.Sync(context.Background(), repo)

	if got.LastSyncStatus != "ok" {
		t.Fatalf("status = %q, want ok", got.LastSyncStatus)
	}
	if got.BaselineCommit != "deadbeef" {
		t.Errorf("baseline = %q, want deadbeef", got.BaselineCommit)
	}
	if !got.Fetched {
		t.Error("Fetched = false, want true")
	}
	if got.BranchName != "main" {
		t.Errorf("branch = %q, want main", got.BranchName)
	}
	if git.resolveBranch != "main" {
		t.Errorf("resolved branch = %q, want main (default branch tracked)", git.resolveBranch)
	}
	if git.fetchCalls != 1 {
		t.Errorf("fetchCalls = %d, want 1", git.fetchCalls)
	}
}

func TestSyncerFetchFailurePreservesBaseline(t *testing.T) {
	git := &fakeGit{fetchErr: errors.New("network down")}
	s := NewSyncer(git, fixedNow)
	repo := ruletypes.CodebaseRepo{
		OrgID: "org1", RepoID: "r1", DefaultBranch: "main",
		Fetched: true, BaselineCommit: "lastgood", BranchName: "main",
	}

	got := s.Sync(context.Background(), repo)

	if got.LastSyncStatus == "ok" {
		t.Fatal("status must not be ok on fetch failure")
	}
	if got.BaselineCommit != "lastgood" {
		t.Errorf("baseline = %q, want lastgood (preserved)", got.BaselineCommit)
	}
	if !got.Fetched {
		t.Error("Fetched should remain true: the cached clone is still usable")
	}
	if git.resolveCalls != 0 {
		t.Errorf("ResolveHead called %d times, want 0 (fetch failed first)", git.resolveCalls)
	}
}

func TestSyncerResolveFailure(t *testing.T) {
	git := &fakeGit{resolveErr: errors.New("unknown revision")}
	s := NewSyncer(git, fixedNow)
	repo := ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1", DefaultBranch: "main"}

	got := s.Sync(context.Background(), repo)

	if got.LastSyncStatus == "ok" {
		t.Fatal("status must not be ok on resolve failure")
	}
	if got.Fetched {
		t.Error("Fetched should be false: never resolved a baseline")
	}
	if got.BranchName != "main" {
		t.Errorf("branch = %q, want main (the branch we tried)", got.BranchName)
	}
}

func TestSyncerNoDefaultBranch(t *testing.T) {
	git := &fakeGit{head: "deadbeef"}
	s := NewSyncer(git, fixedNow)
	repo := ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1", DefaultBranch: ""}

	got := s.Sync(context.Background(), repo)

	if got.LastSyncStatus == "ok" {
		t.Fatal("status must not be ok without a configured branch")
	}
	if git.fetchCalls != 0 {
		t.Errorf("fetchCalls = %d, want 0 (no branch → no git work)", git.fetchCalls)
	}
}
