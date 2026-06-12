package runstore

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/stretchr/testify/require"
)

// TestFloodSim_M1ExitGate is the M1 gate (design §13): under a flood of
// concurrent admits for a small set of error signatures, the cost-control core
// must guarantee (1) no double-admit — each distinct dedup_key yields at most
// one run within the cooldown — and (2) no duplicate DB finalization — each run
// finalizes exactly once and the concurrency semaphore returns cleanly to zero.
// This must pass before any CLI-spawning code (M3) is built.
func TestFloodSim_M1ExitGate(t *testing.T) {
	ctx := context.Background()
	store, ss := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	const (
		distinctKeys     = 5
		floodPerKey      = 12 // concurrent duplicate alerts per signature
		totalAdmitCalls  = distinctKeys * floodPerKey
	)

	type outcome struct {
		admitted bool
		runID    string
		key      string
	}
	outcomes := make([]outcome, totalAdmitCalls)

	var wg sync.WaitGroup
	i := 0
	for k := 0; k < distinctKeys; k++ {
		key := fmt.Sprintf("sig-%d", k)
		for w := 0; w < floodPerKey; w++ {
			idx := i
			i++
			wg.Add(1)
			go func(idx int, key string) {
				defer wg.Done()
				res, err := store.Admit(ctx, baseParams("org1", key, now))
				if err == nil {
					outcomes[idx] = outcome{admitted: res.Admitted, runID: res.RunID, key: key}
				}
			}(idx, key)
		}
	}
	wg.Wait()

	// (1) No double-admit: exactly one run per distinct key.
	require.Equal(t, distinctKeys, countRuns(t, ss, "org1"), "flood must collapse to one run per signature")

	admitsByKey := map[string]int{}
	runIDs := map[string]bool{}
	for _, o := range outcomes {
		if o.admitted {
			admitsByKey[o.key]++
			require.NotEmpty(t, o.runID)
			require.False(t, runIDs[o.runID], "run id must be unique")
			runIDs[o.runID] = true
		}
	}
	require.Len(t, runIDs, distinctKeys)
	for k := 0; k < distinctKeys; k++ {
		require.Equal(t, 1, admitsByKey[fmt.Sprintf("sig-%d", k)], "each signature admits exactly once")
	}

	// (2) No duplicate DB finalize: drain via claim → finalize-once; a repeat
	//     finalize with the same token is fenced out, and the semaphore returns
	//     to exactly zero (no leak, no negative).
	finalized := 0
	for {
		c, err := store.ClaimNext(ctx, claimParams("worker", now, 2))
		require.NoError(t, err)
		if !c.Claimed {
			break
		}
		ok, err := store.Finalize(ctx, FinalizeParams{Scope: scope, RunID: c.RunID, LeaseToken: c.LeaseToken, Status: coderca.RunStatusDone, Now: now})
		require.NoError(t, err)
		require.True(t, ok)

		dup, err := store.Finalize(ctx, FinalizeParams{Scope: scope, RunID: c.RunID, LeaseToken: c.LeaseToken, Status: coderca.RunStatusDone, Now: now})
		require.NoError(t, err)
		require.False(t, dup, "duplicate DB finalize must be rejected")

		finalized++
	}
	require.Equal(t, distinctKeys, finalized)
	require.Equal(t, 0, running(t, ss), "capacity must return to exactly zero")

	var notDone int
	require.NoError(t, ss.BunDB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM coderca_run WHERE org_id = ? AND status != ?", "org1", "done").Scan(&notDone))
	require.Equal(t, 0, notDone, "all runs reach a terminal done state")
}
