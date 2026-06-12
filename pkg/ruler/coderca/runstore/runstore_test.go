package runstore

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/stretchr/testify/require"
)

func applyRunstoreDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	stmts := []string{
		`CREATE TABLE coderca_run (
			run_id          TEXT    NOT NULL PRIMARY KEY,
			org_id          TEXT    NOT NULL,
			service         TEXT    NOT NULL DEFAULT '',
			dedup_key       TEXT    NOT NULL,
			status          TEXT    NOT NULL,
			baseline_commit TEXT    NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL,
			claimed_by      TEXT    NOT NULL DEFAULT '',
			lease_token     TEXT    NOT NULL DEFAULT '',
			lease_until     INTEGER NOT NULL DEFAULT 0,
			heartbeat_at    INTEGER NOT NULL DEFAULT 0,
			attempts        INTEGER NOT NULL DEFAULT 0,
			finished_at     INTEGER NOT NULL DEFAULT 0,
			result_ref      TEXT    NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE coderca_admission (
			org_id           TEXT    NOT NULL,
			dedup_key        TEXT    NOT NULL,
			last_admitted_at INTEGER NOT NULL,
			hit_count        INTEGER NOT NULL DEFAULT 0,
			last_run_ref     TEXT    NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, dedup_key)
		)`,
		`CREATE TABLE coderca_budget (
			org_id TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			used   INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, day)
		)`,
		`CREATE TABLE coderca_capacity (
			scope               TEXT    NOT NULL PRIMARY KEY,
			running             INTEGER NOT NULL DEFAULT 0,
			max_concurrent_runs INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE coderca_skip_stat (
			org_id TEXT    NOT NULL,
			reason TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			count  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, reason, day)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := ss.BunDB().ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func newRunStore(t *testing.T) (*Store, sqlstore.SQLStore) {
	t.Helper()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyRunstoreDDL(context.Background(), ss))
	return New(ss), ss
}

func countRuns(t *testing.T, ss sqlstore.SQLStore, orgID string) int {
	t.Helper()
	var n int
	require.NoError(t, ss.BunDB().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM coderca_run WHERE org_id = ?", orgID).Scan(&n))
	return n
}

func baseParams(orgID, dedupKey string, now time.Time) AdmitParams {
	return AdmitParams{
		OrgID:          orgID,
		Service:        "payments",
		DedupKey:       dedupKey,
		Now:            now,
		CooldownWindow: 6 * time.Hour,
		MaxRunsPerDay:  100,
		MaxQueueDepth:  100,
	}
}

func TestAdmit_FirstAdmits(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	res, err := store.Admit(ctx, baseParams("org1", "k1", now))
	require.NoError(t, err)
	require.True(t, res.Admitted)
	require.NotEmpty(t, res.RunID)
	require.Equal(t, 1, countRuns(t, ss, "org1"))
}

func TestAdmit_DedupWithinCooldown(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	first, err := store.Admit(ctx, baseParams("org1", "k1", now))
	require.NoError(t, err)
	require.True(t, first.Admitted)

	// 1 hour later, still inside the 6h cooldown.
	dup, err := store.Admit(ctx, baseParams("org1", "k1", now.Add(time.Hour)))
	require.NoError(t, err)
	require.False(t, dup.Admitted)
	require.Equal(t, coderca.SkipDeduped, dup.Reason)
	require.Equal(t, first.RunID, dup.PriorRunRef)
	require.Equal(t, 1, countRuns(t, ss, "org1"), "dedup must not create a second run")
}

func TestAdmit_AdmitsAfterCooldown(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	_, err := store.Admit(ctx, baseParams("org1", "k1", now))
	require.NoError(t, err)

	later := now.Add(6*time.Hour + time.Second)
	res, err := store.Admit(ctx, baseParams("org1", "k1", later))
	require.NoError(t, err)
	require.True(t, res.Admitted, "cooled-down key must admit again")
	require.Equal(t, 2, countRuns(t, ss, "org1"))
}

func TestAdmit_BudgetExhausted(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	p := baseParams("org1", "k1", now)
	p.MaxRunsPerDay = 1
	r1, err := store.Admit(ctx, p)
	require.NoError(t, err)
	require.True(t, r1.Admitted)

	// Different key, same org/day → over budget.
	p2 := baseParams("org1", "k2", now)
	p2.MaxRunsPerDay = 1
	r2, err := store.Admit(ctx, p2)
	require.NoError(t, err)
	require.False(t, r2.Admitted)
	require.Equal(t, coderca.SkipBudgetExhausted, r2.Reason)
	require.Equal(t, 1, countRuns(t, ss, "org1"))
}

func TestAdmit_QueueFull(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	p := baseParams("org1", "k1", now)
	p.MaxQueueDepth = 1
	r1, err := store.Admit(ctx, p)
	require.NoError(t, err)
	require.True(t, r1.Admitted)

	// One run already queued → queue full for the next distinct key.
	p2 := baseParams("org1", "k2", now)
	p2.MaxQueueDepth = 1
	r2, err := store.Admit(ctx, p2)
	require.NoError(t, err)
	require.False(t, r2.Admitted)
	require.Equal(t, coderca.SkipQueueFull, r2.Reason)
	require.Equal(t, 1, countRuns(t, ss, "org1"))
}

// The core atomicity invariant: a flood of concurrent admits for ONE key
// produces exactly one run (design §6.2; SQLite single-writer serialization).
func TestAdmit_ConcurrentSameKey_ExactlyOne(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	const n = 20
	var wg sync.WaitGroup
	results := make([]AdmitResult, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = store.Admit(ctx, baseParams("org1", "hot-key", now))
		}(i)
	}
	wg.Wait()

	admitted := 0
	for i := 0; i < n; i++ {
		require.NoError(t, errs[i])
		if results[i].Admitted {
			admitted++
		}
	}
	require.Equal(t, 1, admitted, "exactly one concurrent admit must win")
	require.Equal(t, 1, countRuns(t, ss, "org1"), "exactly one run row under concurrency")
}
