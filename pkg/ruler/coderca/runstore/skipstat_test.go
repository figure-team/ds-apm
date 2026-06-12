package runstore

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/stretchr/testify/require"
)

func day(now time.Time) string { return now.UTC().Format("2006-01-02") }

func TestAdmit_RecordsBudgetSkipCounter(t *testing.T) {
	ctx := context.Background()
	store, _ := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	// cap=1: first admits, next two are budget-exhausted skips.
	p1 := baseParams("org1", "k1", now)
	p1.MaxRunsPerDay = 1
	_, err := store.Admit(ctx, p1)
	require.NoError(t, err)
	for _, k := range []string{"k2", "k3"} {
		p := baseParams("org1", k, now)
		p.MaxRunsPerDay = 1
		r, err := store.Admit(ctx, p)
		require.NoError(t, err)
		require.Equal(t, coderca.SkipBudgetExhausted, r.Reason)
	}

	got, err := store.SkipStat(ctx, "org1", coderca.SkipBudgetExhausted, day(now))
	require.NoError(t, err)
	require.Equal(t, 2, got, "budget skips must aggregate into one counter row")
}

func TestAdmit_RecordsQueueSkipCounter(t *testing.T) {
	ctx := context.Background()
	store, _ := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	p1 := baseParams("org1", "k1", now)
	p1.MaxQueueDepth = 1
	_, err := store.Admit(ctx, p1)
	require.NoError(t, err)

	p2 := baseParams("org1", "k2", now)
	p2.MaxQueueDepth = 1
	r, err := store.Admit(ctx, p2)
	require.NoError(t, err)
	require.Equal(t, coderca.SkipQueueFull, r.Reason)

	got, err := store.SkipStat(ctx, "org1", coderca.SkipQueueFull, day(now))
	require.NoError(t, err)
	require.Equal(t, 1, got)
}

func TestRecordSkip_TriggerLayerReasons(t *testing.T) {
	ctx := context.Background()
	store, _ := newRunStore(t)
	now := time.Unix(1_700_000_000, 0)

	for i := 0; i < 3; i++ {
		require.NoError(t, store.RecordSkip(ctx, "org1", coderca.SkipNoAnomaly, now))
	}
	require.NoError(t, store.RecordSkip(ctx, "org1", coderca.SkipNoRepoMapping, now))

	n, err := store.SkipStat(ctx, "org1", coderca.SkipNoAnomaly, day(now))
	require.NoError(t, err)
	require.Equal(t, 3, n)

	m, err := store.SkipStat(ctx, "org1", coderca.SkipNoRepoMapping, day(now))
	require.NoError(t, err)
	require.Equal(t, 1, m)
}
