// Package runstore is the DB-backed cost-control core of CF-11: atomic
// admission (dedup + budget + queue) and (later) worker lease/claim. All
// volume enforcement happens inside DB transactions so it holds under
// concurrency and across restarts/replicas (design §6).
package runstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/sqlstore"
)

// newRunID returns a random 128-bit hex id (no external uuid dependency).
func newRunID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// AdmitParams is the input to a single admission decision. Thresholds are
// passed in (sourced from codebase_config) so the logic stays testable.
type AdmitParams struct {
	OrgID    string
	Service  string
	DedupKey string
	Now      time.Time

	CooldownWindow time.Duration // sliding dedup window
	MaxRunsPerDay  int           // per-org daily run cap
	MaxQueueDepth  int           // max queued runs per org
}

// AdmitResult reports the admission decision. Exactly one run is created when
// Admitted is true; otherwise Reason explains the skip.
type AdmitResult struct {
	Admitted    bool
	RunID       string             // set when Admitted
	Reason      coderca.SkipReason // set when !Admitted
	PriorRunRef string             // set when Reason == deduped
}

// Store is the SQL-backed run/admission store.
type Store struct {
	sqlstore sqlstore.SQLStore
}

// New returns a run store backed by the given SQLStore. Migration 081 must
// have run (coderca_run / coderca_admission / coderca_budget).
func New(store sqlstore.SQLStore) *Store {
	return &Store{sqlstore: store}
}

// Admit atomically decides whether a candidate signal becomes a queued run.
// In one transaction it enforces dedup (sliding cooldown), per-day budget, and
// queue depth; on the live SQLite backend the single-writer connection
// serializes concurrent admits, so for one dedup_key exactly one run is
// created (design §6.2).
func (s *Store) Admit(ctx context.Context, p AdmitParams) (AdmitResult, error) {
	// Defensive: a non-positive daily cap admits nothing (feature should be
	// gated off upstream before reaching here).
	if p.MaxRunsPerDay <= 0 {
		return AdmitResult{Reason: coderca.SkipBudgetExhausted}, nil
	}

	nowUnix := p.Now.Unix()
	day := p.Now.UTC().Format("2006-01-02")
	cooldownSecs := int64(p.CooldownWindow / time.Second)

	var res AdmitResult
	err := s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		db := s.sqlstore.BunDBCtx(ctx)

		// 1. Dedup (read). A row within the sliding cooldown is a duplicate.
		var lastAdmitted int64
		var lastRunRef string
		scanErr := db.NewRaw(
			"SELECT last_admitted_at, last_run_ref FROM coderca_admission WHERE org_id = ? AND dedup_key = ?",
			p.OrgID, p.DedupKey,
		).Scan(ctx, &lastAdmitted, &lastRunRef)
		switch {
		case scanErr == nil:
			if nowUnix-lastAdmitted < cooldownSecs {
				if _, err := db.ExecContext(ctx,
					"UPDATE coderca_admission SET hit_count = hit_count + 1 WHERE org_id = ? AND dedup_key = ?",
					p.OrgID, p.DedupKey,
				); err != nil {
					return err
				}
				res = AdmitResult{Reason: coderca.SkipDeduped, PriorRunRef: lastRunRef}
				return nil
			}
		case errors.Is(scanErr, sql.ErrNoRows):
			// fresh key — fall through to admit
		default:
			return scanErr
		}

		// 2. Queue depth (read). Checked before any write so a rejection never
		//    over-counts the budget.
		var queued int
		if err := db.NewRaw(
			"SELECT COUNT(*) FROM coderca_run WHERE org_id = ? AND status = ?",
			p.OrgID, string(coderca.RunStatusQueued),
		).Scan(ctx, &queued); err != nil {
			return err
		}
		if queued >= p.MaxQueueDepth {
			res = AdmitResult{Reason: coderca.SkipQueueFull}
			return nil
		}

		// 3. Budget (conditional atomic increment). 0 rows affected ⇒ at/over cap.
		r, err := db.ExecContext(ctx,
			`INSERT INTO coderca_budget (org_id, day, used) VALUES (?, ?, 1)
			 ON CONFLICT (org_id, day) DO UPDATE SET used = used + 1 WHERE coderca_budget.used < ?`,
			p.OrgID, day, p.MaxRunsPerDay,
		)
		if err != nil {
			return err
		}
		affected, err := r.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			res = AdmitResult{Reason: coderca.SkipBudgetExhausted}
			return nil
		}

		// 4. Insert the queued run.
		runID, err := newRunID()
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO coderca_run (run_id, org_id, service, dedup_key, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			runID, p.OrgID, p.Service, p.DedupKey, string(coderca.RunStatusQueued), nowUnix,
		); err != nil {
			return err
		}

		// 5. Upsert the admission row (pins the sliding window + last run ref).
		if _, err := db.ExecContext(ctx,
			`INSERT INTO coderca_admission (org_id, dedup_key, last_admitted_at, hit_count, last_run_ref)
			 VALUES (?, ?, ?, 0, ?)
			 ON CONFLICT (org_id, dedup_key) DO UPDATE SET last_admitted_at = ?, last_run_ref = ?`,
			p.OrgID, p.DedupKey, nowUnix, runID, nowUnix, runID,
		); err != nil {
			return err
		}

		res = AdmitResult{Admitted: true, RunID: runID}
		return nil
	})
	if err != nil {
		return AdmitResult{}, err
	}
	return res, nil
}
