package runstore

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
)

// ClaimParams configures a single claim attempt by a worker.
type ClaimParams struct {
	Scope         string // capacity scope (e.g. "global")
	ClaimedBy     string // worker/instance id
	Now           time.Time
	LeaseTTL      time.Duration // must be >= run_timeout + grace (design §6.3)
	MaxConcurrent int
}

// ClaimResult reports whether a queued run was claimed.
type ClaimResult struct {
	Claimed    bool
	RunID      string
	LeaseToken string // fencing token — required to heartbeat/finalize this run
	DedupKey   string
	Service    string
	Attempts   int
}

// FinalizeParams ends a run the caller owns (identified by the fencing token).
type FinalizeParams struct {
	Scope      string
	RunID      string
	LeaseToken string
	Status     coderca.RunStatus // done | failed | timeout | unparseable
	ResultRef  string
	Now        time.Time
}

// ReapParams sweeps expired leases.
type ReapParams struct {
	Scope       string
	Now         time.Time
	MaxAttempts int
}

// ClaimNext atomically claims the oldest eligible queued run, subject to the
// global concurrency cap enforced via a single locked coderca_capacity row.
// Returns Claimed=false when nothing is queued or the cap is reached (§6.3).
func (s *Store) ClaimNext(ctx context.Context, p ClaimParams) (ClaimResult, error) {
	// STUB — replaced in GREEN.
	return ClaimResult{}, nil
}

// Heartbeat extends the lease for a run the caller still owns (fenced by
// lease_token). Returns false if the run is no longer owned/running.
func (s *Store) Heartbeat(ctx context.Context, runID, leaseToken string, leaseUntil time.Time) (bool, error) {
	// STUB — replaced in GREEN.
	return false, nil
}

// Finalize performs a fenced terminal transition and releases one capacity
// slot — but only when the fenced update affects exactly one row. A stale
// owner whose run was reaped/reclaimed affects zero rows and must NOT
// decrement capacity (§6.3). Returns whether the caller finalized the run.
func (s *Store) Finalize(ctx context.Context, p FinalizeParams) (bool, error) {
	// STUB — replaced in GREEN.
	return false, nil
}

// Reap requeues runs whose lease expired (→ failed once attempts >=
// MaxAttempts) and reconciles coderca_capacity.running to the live-leased
// count. Returns the number of runs requeued/failed.
func (s *Store) Reap(ctx context.Context, p ReapParams) (int, error) {
	// STUB — replaced in GREEN.
	return 0, nil
}
