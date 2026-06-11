// Package runstore is the DB-backed cost-control core of CF-11: atomic
// admission (dedup + budget + queue) and (later) worker lease/claim. All
// volume enforcement happens inside DB transactions so it holds under
// concurrency and across restarts/replicas (design §6).
package runstore

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/sqlstore"
)

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
	// STUB — replaced in GREEN.
	return AdmitResult{}, nil
}
