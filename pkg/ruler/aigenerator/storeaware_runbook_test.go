package aigenerator

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/aiconfigstoretest"
	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// countingDrafter is a stub RunbookDrafter recording Draft calls — used to
// assert whether the store-aware wrapper delegated to the env fallback.
type countingDrafter struct {
	calls int
	out   ruletypes.Runbook
}

func (d *countingDrafter) Draft(_ context.Context, _ ruletypes.RunbookDraftRequest) (ruletypes.Runbook, error) {
	d.calls++
	return d.out, nil
}

func newDrafter(store ruletypes.AIConfigStore, fb ruletypes.RunbookDrafter) *StoreAwareRunbookDrafter {
	return NewStoreAwareRunbookDrafter(store, secretbox.PlaintextCipher(), fb)
}

func TestStoreAwareRunbookDrafter_FallsBackWhenNoOrg(t *testing.T) {
	fb := &countingDrafter{out: ruletypes.Runbook{Title: "fallback"}}
	got, err := newDrafter(aiconfigstoretest.New(), fb).
		Draft(context.Background(), ruletypes.RunbookDraftRequest{}) // no OrgID
	require.NoError(t, err)
	require.Equal(t, "fallback", got.Title)
	require.Equal(t, 1, fb.calls)
}

func TestStoreAwareRunbookDrafter_FallsBackWhenStoreMiss(t *testing.T) {
	fb := &countingDrafter{out: ruletypes.Runbook{Title: "fallback"}}
	got, err := newDrafter(aiconfigstoretest.New(), fb).
		Draft(context.Background(), ruletypes.RunbookDraftRequest{OrgID: "org-1"})
	require.NoError(t, err)
	require.Equal(t, "fallback", got.Title)
	require.Equal(t, 1, fb.calls)
}

func TestStoreAwareRunbookDrafter_FallsBackWhenStoredConfigNotLLM(t *testing.T) {
	fb := &countingDrafter{out: ruletypes.Runbook{Title: "fallback"}}
	store := aiconfigstoretest.New()
	// provider=local is a stored config that is NOT an llm credential.
	require.NoError(t, store.Upsert(context.Background(), validLocalConfig("org-1"),
		func(s string) (string, error) { return s, nil }))

	got, err := newDrafter(store, fb).
		Draft(context.Background(), ruletypes.RunbookDraftRequest{OrgID: "org-1"})
	require.NoError(t, err)
	require.Equal(t, "fallback", got.Title, "non-llm stored config must fall back to env drafter")
	require.Equal(t, 1, fb.calls)
}
