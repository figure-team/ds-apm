package sourcestate

import (
	"errors"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func TestApply(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	tests := []struct {
		name  string
		prev  SourceState
		facts SyncFacts
		want  SourceState
	}{
		{
			name:  "first successful sync pins baseline",
			prev:  SourceState{},
			facts: SyncFacts{Branch: "main", HeadCommit: "abc123"},
			want:  SourceState{BranchName: "main", Fetched: true, BaselineCommit: "abc123", LastSyncAt: nowStr, LastSyncStatus: "ok"},
		},
		{
			name:  "successful re-sync advances baseline",
			prev:  SourceState{BranchName: "main", Fetched: true, BaselineCommit: "old", LastSyncAt: "earlier", LastSyncStatus: "ok"},
			facts: SyncFacts{Branch: "main", HeadCommit: "new"},
			want:  SourceState{BranchName: "main", Fetched: true, BaselineCommit: "new", LastSyncAt: nowStr, LastSyncStatus: "ok"},
		},
		{
			name:  "head commit is trimmed",
			prev:  SourceState{},
			facts: SyncFacts{Branch: "main", HeadCommit: "  abc\n"},
			want:  SourceState{BranchName: "main", Fetched: true, BaselineCommit: "abc", LastSyncAt: nowStr, LastSyncStatus: "ok"},
		},
		{
			name:  "failed re-sync keeps last-good baseline",
			prev:  SourceState{BranchName: "main", Fetched: true, BaselineCommit: "good", LastSyncAt: "earlier", LastSyncStatus: "ok"},
			facts: SyncFacts{Branch: "main", Err: errors.New("boom")},
			want:  SourceState{BranchName: "main", Fetched: true, BaselineCommit: "good", LastSyncAt: nowStr, LastSyncStatus: "error: boom"},
		},
		{
			name:  "first sync failure leaves repo unfetched",
			prev:  SourceState{},
			facts: SyncFacts{Branch: "main", Err: errors.New("dial tcp: timeout")},
			want:  SourceState{BranchName: "main", Fetched: false, BaselineCommit: "", LastSyncAt: nowStr, LastSyncStatus: "error: dial tcp: timeout"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Apply(tc.prev, tc.facts, now)
			if got != tc.want {
				t.Errorf("Apply() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestStateOf(t *testing.T) {
	repo := ruletypes.CodebaseRepo{
		BranchName:     "main",
		Fetched:        true,
		BaselineCommit: "abc",
		LastSyncAt:     "2026-06-12T09:00:00Z",
		LastSyncStatus: "ok",
	}
	got := StateOf(repo)
	want := SourceState{BranchName: "main", Fetched: true, BaselineCommit: "abc", LastSyncAt: "2026-06-12T09:00:00Z", LastSyncStatus: "ok"}
	if got != want {
		t.Errorf("StateOf() = %+v, want %+v", got, want)
	}
}
