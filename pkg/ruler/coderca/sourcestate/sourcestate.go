// Package sourcestate holds the pure source-state transition logic for CF-11
// (design §8). Real git work sits behind the GitRunner interface; the state
// machine itself — given the facts observed during a sync attempt, compute the
// next tracked state and the pinned baseline commit — is pure and table-tested
// with a fake.
package sourcestate

import (
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// SyncStatusOK marks a sync attempt that fetched and resolved a baseline.
const SyncStatusOK = "ok"

// SourceState is the tracked state of a repo's cached source, surfaced
// read-only in the UI (design §8): which branch, whether a usable local clone
// exists, the pinned baseline commit, and the outcome/time of the last sync.
type SourceState struct {
	BranchName     string
	Fetched        bool
	BaselineCommit string
	LastSyncAt     string // RFC3339
	LastSyncStatus string // SyncStatusOK | "error: <msg>"
}

// SyncFacts are the observations from one sync attempt: the branch tracked, the
// resolved HEAD commit (on success), and the error (on failure). Exactly the
// inputs the pure transition needs — no IO.
type SyncFacts struct {
	Branch     string
	HeadCommit string
	Err        error
}

// StateOf extracts the tracked source state from a stored repo so a sync can
// compute the next state relative to the last-known one.
//
// M2-2 STUB: returns the zero state → assertions fail (RED).
func StateOf(repo ruletypes.CodebaseRepo) SourceState {
	return SourceState{}
}

// Apply computes the next source state from the previous state and the facts of
// a sync attempt. On failure it preserves the last-good baseline (a transient
// fetch error must not erase a usable cached clone). Pure.
//
// M2-2 STUB: returns prev unchanged → success assertions fail (RED).
func Apply(prev SourceState, facts SyncFacts, now time.Time) SourceState {
	return prev
}
