package runstore

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/stretchr/testify/require"
)

const scope = "global"

func seedQueued(t *testing.T, store *Store, org, key string, now time.Time) string {
	t.Helper()
	res, err := store.Admit(context.Background(), baseParams(org, key, now))
	require.NoError(t, err)
	require.True(t, res.Admitted)
	return res.RunID
}

func claimParams(by string, now time.Time, maxConc int) ClaimParams {
	return ClaimParams{Scope: scope, ClaimedBy: by, Now: now, LeaseTTL: 10 * time.Minute, MaxConcurrent: maxConc}
}

func runStatus(t *testing.T, ss sqlstore.SQLStore, runID string) string {
	t.Helper()
	var st string
	require.NoError(t, ss.BunDB().QueryRowContext(context.Background(),
		"SELECT status FROM coderca_run WHERE run_id = ?", runID).Scan(&st))
	return st
}

func running(t *testing.T, ss sqlstore.SQLStore) int {
	t.Helper()
	var n int
	err := ss.BunDB().QueryRowContext(context.Background(),
		"SELECT running FROM coderca_capacity WHERE scope = ?", scope).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0
	}
	require.NoError(t, err)
	return n
}

func TestClaim_NoneQueued(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	res, err := store.ClaimNext(ctx, claimParams("w1", time.Unix(1_700_000_000, 0), 1))
	require.NoError(t, err)
	require.False(t, res.Claimed)
	require.Equal(t, 0, running(t, ss))
}

func TestClaim_ClaimsOldestAndTakesSlot(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	first := seedQueued(t, store, "org1", "k1", now)
	_ = seedQueued(t, store, "org1", "k2", now.Add(time.Second)) // newer

	res, err := store.ClaimNext(ctx, claimParams("w1", now.Add(2*time.Second), 1))
	require.NoError(t, err)
	require.True(t, res.Claimed)
	require.Equal(t, first, res.RunID, "oldest queued run is claimed first")
	require.NotEmpty(t, res.LeaseToken)
	require.Equal(t, 1, res.Attempts)
	require.Equal(t, "running", runStatus(t, ss, first))
	require.Equal(t, 1, running(t, ss))
}

func TestClaim_ReturnsOrgID(t *testing.T) {
	ctx := context.Background()
	store, _ := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	seedQueued(t, store, "org-7", "k1", now)

	res, err := store.ClaimNext(ctx, claimParams("w1", now.Add(time.Second), 1))
	require.NoError(t, err)
	require.True(t, res.Claimed)
	require.Equal(t, "org-7", res.OrgID, "claim must expose the run's org so the worker can resolve an org-scoped repo")
}

func TestClaim_RespectsCapacitySequential(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	r1 := seedQueued(t, store, "org1", "k1", now)
	_ = seedQueued(t, store, "org1", "k2", now.Add(time.Second))

	c1, err := store.ClaimNext(ctx, claimParams("w1", now, 1))
	require.NoError(t, err)
	require.True(t, c1.Claimed)

	c2, err := store.ClaimNext(ctx, claimParams("w2", now, 1))
	require.NoError(t, err)
	require.False(t, c2.Claimed, "cap=1 reached")
	require.Equal(t, 1, running(t, ss))

	ok, err := store.Finalize(ctx, FinalizeParams{Scope: scope, RunID: c1.RunID, LeaseToken: c1.LeaseToken, Status: coderca.RunStatusDone, Now: now})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 0, running(t, ss))
	require.Equal(t, "done", runStatus(t, ss, r1))

	c3, err := store.ClaimNext(ctx, claimParams("w2", now, 1))
	require.NoError(t, err)
	require.True(t, c3.Claimed, "slot freed → next claim succeeds")
	require.Equal(t, 1, running(t, ss))
}

func TestClaim_ConcurrentRespectsCap(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	for i := 0; i < 6; i++ {
		seedQueued(t, store, "org1", string(rune('a'+i)), now.Add(time.Duration(i)*time.Second))
	}

	const workers = 8
	var wg sync.WaitGroup
	claimed := make([]bool, workers)
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := store.ClaimNext(ctx, claimParams("w", now.Add(10*time.Second), 2))
			errs[i] = err
			claimed[i] = res.Claimed
		}(i)
	}
	wg.Wait()

	n := 0
	for i := 0; i < workers; i++ {
		require.NoError(t, errs[i])
		if claimed[i] {
			n++
		}
	}
	require.Equal(t, 2, n, "global cap=2 bounds concurrent claims")
	require.Equal(t, 2, running(t, ss))
}

func TestReap_RequeuesThenFailsAtMaxAttempts(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	r1 := seedQueued(t, store, "org1", "k1", now)
	c1, err := store.ClaimNext(ctx, claimParams("w1", now, 1))
	require.NoError(t, err)
	require.True(t, c1.Claimed)

	// Lease expires (claimed at now, ttl 10m); reap at now+11m.
	n, err := store.Reap(ctx, ReapParams{Scope: scope, Now: now.Add(11 * time.Minute), MaxAttempts: 2})
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "queued", runStatus(t, ss, r1), "attempts(1) < max(2) → requeued")
	require.Equal(t, 0, running(t, ss), "capacity reconciled after reap")

	// Claim again → attempts=2, then expire + reap → failed.
	c2, err := store.ClaimNext(ctx, claimParams("w1", now.Add(11*time.Minute), 1))
	require.NoError(t, err)
	require.True(t, c2.Claimed)
	require.Equal(t, 2, c2.Attempts)

	n2, err := store.Reap(ctx, ReapParams{Scope: scope, Now: now.Add(22 * time.Minute), MaxAttempts: 2})
	require.NoError(t, err)
	require.Equal(t, 1, n2)
	require.Equal(t, "failed", runStatus(t, ss, r1), "attempts(2) >= max(2) → failed")
	require.Equal(t, 0, running(t, ss))
}

func TestFinalize_FencingRejectsStaleOwner(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	runID := seedQueued(t, store, "org1", "k1", now)
	a, err := store.ClaimNext(ctx, claimParams("wA", now, 1))
	require.NoError(t, err)
	require.True(t, a.Claimed)

	// Lease expires; reaper requeues; worker B reclaims with a fresh token.
	_, err = store.Reap(ctx, ReapParams{Scope: scope, Now: now.Add(11 * time.Minute), MaxAttempts: 5})
	require.NoError(t, err)
	b, err := store.ClaimNext(ctx, claimParams("wB", now.Add(11*time.Minute), 1))
	require.NoError(t, err)
	require.True(t, b.Claimed)
	require.Equal(t, runID, b.RunID)
	require.NotEqual(t, a.LeaseToken, b.LeaseToken)
	require.Equal(t, 1, running(t, ss))

	// Stale owner A tries to finalize — must be fenced out, capacity untouched.
	okA, err := store.Finalize(ctx, FinalizeParams{Scope: scope, RunID: runID, LeaseToken: a.LeaseToken, Status: coderca.RunStatusDone, Now: now.Add(12 * time.Minute)})
	require.NoError(t, err)
	require.False(t, okA, "stale fencing token must be rejected")
	require.Equal(t, 1, running(t, ss), "stale finalize must NOT decrement capacity")
	require.Equal(t, "running", runStatus(t, ss, runID))

	// Live owner B finalizes successfully.
	okB, err := store.Finalize(ctx, FinalizeParams{Scope: scope, RunID: runID, LeaseToken: b.LeaseToken, Status: coderca.RunStatusDone, Now: now.Add(12 * time.Minute)})
	require.NoError(t, err)
	require.True(t, okB)
	require.Equal(t, 0, running(t, ss))
	require.Equal(t, "done", runStatus(t, ss, runID))
}
