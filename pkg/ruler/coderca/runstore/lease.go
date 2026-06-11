package runstore

import (
	"context"
	"database/sql"
	"errors"
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
	nowUnix := p.Now.Unix()
	leaseUntil := p.Now.Add(p.LeaseTTL).Unix()

	var res ClaimResult
	err := s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		db := s.sqlstore.BunDBCtx(ctx)

		// 1. Oldest eligible queued run (read). Nothing queued → no claim, no
		//    capacity touched.
		var runID, dedupKey, service string
		var attempts int
		scanErr := db.NewRaw(
			`SELECT run_id, dedup_key, service, attempts FROM coderca_run
			 WHERE status = ? AND (lease_until = 0 OR lease_until < ?)
			 ORDER BY created_at ASC LIMIT 1`,
			string(coderca.RunStatusQueued), nowUnix,
		).Scan(ctx, &runID, &dedupKey, &service, &attempts)
		if errors.Is(scanErr, sql.ErrNoRows) {
			return nil
		}
		if scanErr != nil {
			return scanErr
		}

		// 2. Take a capacity slot via the single locked capacity row. 0 rows
		//    affected ⇒ at cap (the row's conditional UPDATE serializes the
		//    decision); leave the run untouched.
		if _, err := db.ExecContext(ctx,
			`INSERT INTO coderca_capacity (scope, running, max_concurrent_runs) VALUES (?, 0, ?)
			 ON CONFLICT (scope) DO NOTHING`,
			p.Scope, p.MaxConcurrent,
		); err != nil {
			return err
		}
		slot, err := db.ExecContext(ctx,
			`UPDATE coderca_capacity SET running = running + 1, max_concurrent_runs = ?
			 WHERE scope = ? AND running < ?`,
			p.MaxConcurrent, p.Scope, p.MaxConcurrent,
		)
		if err != nil {
			return err
		}
		got, err := slot.RowsAffected()
		if err != nil {
			return err
		}
		if got == 0 {
			return nil // at capacity
		}

		// 3. Mark the run running with a fresh fencing token.
		leaseToken, err := newRunID()
		if err != nil {
			return err
		}
		marked, err := db.ExecContext(ctx,
			`UPDATE coderca_run SET status = ?, claimed_by = ?, lease_token = ?,
			        lease_until = ?, heartbeat_at = ?, attempts = attempts + 1
			 WHERE run_id = ? AND status = ?`,
			string(coderca.RunStatusRunning), p.ClaimedBy, leaseToken,
			leaseUntil, nowUnix, runID, string(coderca.RunStatusQueued),
		)
		if err != nil {
			return err
		}
		mrows, err := marked.RowsAffected()
		if err != nil {
			return err
		}
		if mrows == 0 {
			// Lost a race for this run (cannot happen on the serialized SQLite
			// writer; defensive for PG). Release the slot we took.
			if _, err := db.ExecContext(ctx,
				"UPDATE coderca_capacity SET running = running - 1 WHERE scope = ? AND running > 0",
				p.Scope,
			); err != nil {
				return err
			}
			return nil
		}

		res = ClaimResult{
			Claimed:    true,
			RunID:      runID,
			LeaseToken: leaseToken,
			DedupKey:   dedupKey,
			Service:    service,
			Attempts:   attempts + 1,
		}
		return nil
	})
	if err != nil {
		return ClaimResult{}, err
	}
	return res, nil
}

// Heartbeat extends the lease for a run the caller still owns (fenced by
// lease_token). Returns false if the run is no longer owned/running.
func (s *Store) Heartbeat(ctx context.Context, runID, leaseToken string, leaseUntil time.Time) (bool, error) {
	var ok bool
	err := s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		r, err := s.sqlstore.BunDBCtx(ctx).ExecContext(ctx,
			`UPDATE coderca_run SET lease_until = ?, heartbeat_at = ?
			 WHERE run_id = ? AND lease_token = ? AND status = ?`,
			leaseUntil.Unix(), leaseUntil.Unix(), runID, leaseToken, string(coderca.RunStatusRunning),
		)
		if err != nil {
			return err
		}
		n, err := r.RowsAffected()
		if err != nil {
			return err
		}
		ok = n == 1
		return nil
	})
	return ok, err
}

// Finalize performs a fenced terminal transition and releases one capacity
// slot — but only when the fenced update affects exactly one row. A stale
// owner whose run was reaped/reclaimed affects zero rows and must NOT
// decrement capacity (§6.3). Returns whether the caller finalized the run.
func (s *Store) Finalize(ctx context.Context, p FinalizeParams) (bool, error) {
	var ok bool
	err := s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		db := s.sqlstore.BunDBCtx(ctx)

		// Fenced terminal transition.
		r, err := db.ExecContext(ctx,
			`UPDATE coderca_run SET status = ?, result_ref = ?, finished_at = ?, lease_until = 0
			 WHERE run_id = ? AND lease_token = ? AND status = ?`,
			string(p.Status), p.ResultRef, p.Now.Unix(), p.RunID, p.LeaseToken, string(coderca.RunStatusRunning),
		)
		if err != nil {
			return err
		}
		n, err := r.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			// Stale owner (run reaped/reclaimed): do NOT decrement capacity.
			return nil
		}

		// We owned it: release exactly one capacity slot.
		if _, err := db.ExecContext(ctx,
			"UPDATE coderca_capacity SET running = running - 1 WHERE scope = ? AND running > 0",
			p.Scope,
		); err != nil {
			return err
		}
		ok = true
		return nil
	})
	return ok, err
}

// Reap requeues runs whose lease expired (→ failed once attempts >=
// MaxAttempts) and reconciles coderca_capacity.running to the live-leased
// count. Returns the number of runs requeued/failed.
func (s *Store) Reap(ctx context.Context, p ReapParams) (int, error) {
	nowUnix := p.Now.Unix()
	var reaped int
	err := s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		db := s.sqlstore.BunDBCtx(ctx)

		// Requeue expired-lease running runs; fail those past max_attempts.
		r, err := db.ExecContext(ctx,
			`UPDATE coderca_run
			 SET status = CASE WHEN attempts >= ? THEN ? ELSE ? END,
			     claimed_by = '', lease_token = '', lease_until = 0
			 WHERE status = ? AND lease_until > 0 AND lease_until < ?`,
			p.MaxAttempts, string(coderca.RunStatusFailed), string(coderca.RunStatusQueued),
			string(coderca.RunStatusRunning), nowUnix,
		)
		if err != nil {
			return err
		}
		n, err := r.RowsAffected()
		if err != nil {
			return err
		}
		reaped = int(n)

		// Reconcile capacity.running to the live-leased count (corrects drift
		// from any missed decrement).
		if _, err := db.ExecContext(ctx,
			`UPDATE coderca_capacity
			 SET running = (SELECT COUNT(*) FROM coderca_run WHERE status = ? AND lease_until > ?)
			 WHERE scope = ?`,
			string(coderca.RunStatusRunning), nowUnix, p.Scope,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return reaped, nil
}
