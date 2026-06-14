package runstore

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/stretchr/testify/require"
)

// TestFinalizePersistsReportAndBaseline verifies that FinalizeParams report
// fields are written to the DB and readable via GetRun.
func TestFinalizePersistsReportAndBaseline(t *testing.T) {
	ctx := context.Background()
	store, _ := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	runID := seedQueued(t, store, "org1", "rpt-key", now)
	c, err := store.ClaimNext(ctx, claimParams("w1", now.Add(time.Second), 1))
	require.NoError(t, err)
	require.True(t, c.Claimed)
	require.Equal(t, runID, c.RunID)

	ok, err := store.Finalize(ctx, FinalizeParams{
		Scope:          scope,
		RunID:          c.RunID,
		LeaseToken:     c.LeaseToken,
		Status:         coderca.RunStatusDone,
		ResultRef:      "ref-1",
		Now:            now.Add(2 * time.Second),
		BaselineCommit: "abc123",
		RootCause:      "null pointer in handler",
		ProposedFix:    "add nil check before dereference",
		Confidence:     "high",
		Limitations:    "only tested on happy path",
	})
	require.NoError(t, err)
	require.True(t, ok)

	d, err := store.GetRun(ctx, "org1", runID)
	require.NoError(t, err)
	require.Equal(t, runID, d.RunID)
	require.Equal(t, "org1", d.OrgID)
	require.Equal(t, coderca.RunStatusDone, d.Status)
	require.Equal(t, "ref-1", d.ResultRef)
	require.Equal(t, "abc123", d.BaselineCommit)
	require.Equal(t, "null pointer in handler", d.RootCause)
	require.Equal(t, "add nil check before dereference", d.ProposedFix)
	require.Equal(t, "high", d.Confidence)
	require.Equal(t, "only tested on happy path", d.Limitations)
}

// TestListRunsFiltersAndTenantIsolation verifies filtering, ordering, and
// per-org isolation for ListRuns and GetRun.
func TestListRunsFiltersAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store, _ := newRunStore(t)

	// Use distinct timestamps so created_at ordering is deterministic.
	t0 := time.Unix(1_700_000_000, 0)
	t1 := t0.Add(time.Second)
	t2 := t0.Add(2 * time.Second)

	// org-1: two runs with distinct services.
	// Run A (payments, older): admitted at t0, finalized as done.
	apA := AdmitParams{
		OrgID: "org-1", Service: "payments", DedupKey: "dk-A", Now: t0,
		CooldownWindow: 6 * time.Hour, MaxRunsPerDay: 100, MaxQueueDepth: 100,
	}
	resA, err := store.Admit(ctx, apA)
	require.NoError(t, err)
	require.True(t, resA.Admitted)
	runA := resA.RunID

	cA, err := store.ClaimNext(ctx, claimParams("w1", t0.Add(time.Millisecond), 2))
	require.NoError(t, err)
	require.True(t, cA.Claimed)
	require.Equal(t, runA, cA.RunID)

	_, err = store.Finalize(ctx, FinalizeParams{
		Scope: scope, RunID: cA.RunID, LeaseToken: cA.LeaseToken,
		Status: coderca.RunStatusDone, Now: t0.Add(time.Second),
	})
	require.NoError(t, err)

	// Run B (gateway, newer): admitted at t1, left queued.
	apB := AdmitParams{
		OrgID: "org-1", Service: "gateway", DedupKey: "dk-B", Now: t1,
		CooldownWindow: 6 * time.Hour, MaxRunsPerDay: 100, MaxQueueDepth: 100,
	}
	resB, err := store.Admit(ctx, apB)
	require.NoError(t, err)
	require.True(t, resB.Admitted)
	runB := resB.RunID

	// org-2: one run.
	apC := AdmitParams{
		OrgID: "org-2", Service: "payments", DedupKey: "dk-C", Now: t2,
		CooldownWindow: 6 * time.Hour, MaxRunsPerDay: 100, MaxQueueDepth: 100,
	}
	resC, err := store.Admit(ctx, apC)
	require.NoError(t, err)
	require.True(t, resC.Admitted)
	runC := resC.RunID

	// ListRuns(org-1, {}) → exactly 2, newest first (runB then runA).
	all1, err := store.ListRuns(ctx, "org-1", ListRunsParams{})
	require.NoError(t, err)
	require.Len(t, all1, 2, "org-1 must have 2 runs")
	require.Equal(t, runB, all1[0].RunID, "newest run first")
	require.Equal(t, runA, all1[1].RunID, "older run second")

	// ListRuns(org-1, {Status:"done"}) → exactly 1 (runA).
	doneRuns, err := store.ListRuns(ctx, "org-1", ListRunsParams{Status: "done"})
	require.NoError(t, err)
	require.Len(t, doneRuns, 1)
	require.Equal(t, runA, doneRuns[0].RunID)

	// ListRuns(org-1, {Service:"gateway"}) → exactly 1 (runB).
	gwRuns, err := store.ListRuns(ctx, "org-1", ListRunsParams{Service: "gateway"})
	require.NoError(t, err)
	require.Len(t, gwRuns, 1)
	require.Equal(t, runB, gwRuns[0].RunID)

	// ListRuns(org-2, {}) → 1 (org-1 runs not visible).
	all2, err := store.ListRuns(ctx, "org-2", ListRunsParams{})
	require.NoError(t, err)
	require.Len(t, all2, 1)
	require.Equal(t, runC, all2[0].RunID)

	// GetRun(org-2, org-1 runID) → ErrRunNotFound (tenant isolation).
	_, err = store.GetRun(ctx, "org-2", runA)
	require.ErrorIs(t, err, ErrRunNotFound)

	// GetRun(org-1, "nonexistent") → ErrRunNotFound.
	_, err = store.GetRun(ctx, "org-1", "nonexistent-run-id")
	require.ErrorIs(t, err, ErrRunNotFound)
}
