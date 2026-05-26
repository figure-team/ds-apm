package sqlaihistorystore

import (
	"context"
	"testing"

	signoztf "github.com/SigNoz/signoz/pkg/testfixtures"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) ruletypes.AIStrategyHistoryStore {
	t.Helper()
	ctx := context.Background()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyHistoryDDL(ctx, ss))
	return NewAIStrategyHistoryStore(ss)
}

// newSeededStore returns a store pre-loaded from tests/fixtures/go/ds_ai_strategy_history.yml.
func newSeededStore(t *testing.T) (ruletypes.AIStrategyHistoryStore, sqlstore.SQLStore) {
	t.Helper()
	ctx := context.Background()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyHistoryDDL(ctx, ss))
	signoztf.Load(t, ss, signoztf.DefaultFixtureDir(), "ds_ai_strategy_history")
	return NewAIStrategyHistoryStore(ss), ss
}

func applyHistoryDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	stmts := []string{
		`CREATE TABLE ds_ai_strategy_history (
			org_id              TEXT      NOT NULL,
			incident_id         TEXT      NOT NULL,
			alert_fingerprint   TEXT      NOT NULL DEFAULT '',
			contract_version    TEXT      NOT NULL,
			payload             TEXT      NOT NULL,
			PRIMARY KEY (org_id, incident_id)
		)`,
		`CREATE UNIQUE INDEX idx_ds_ai_strategy_history_org_fp
			ON ds_ai_strategy_history(org_id, alert_fingerprint)
			WHERE alert_fingerprint <> ''`,
	}
	for _, stmt := range stmts {
		if _, err := ss.BunDB().ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func makeRecord(t *testing.T, incidentID, fingerprint string) ruletypes.AIStrategyHistoryRecord {
	t.Helper()
	strategy := ruletypes.AIStrategy{
		ContractVersion:  ruletypes.AIStrategyContractVersion,
		StrategyID:       "strat-" + incidentID,
		IncidentID:       incidentID,
		AlertFingerprint: fingerprint,
		Status:           ruletypes.AIStrategyStatusUnavailable,
		Language:         "ko-KR",
		Confidence:       ruletypes.AIConfidenceLow,
		Limitations:      []string{"test"},
		Audit: ruletypes.AIStrategyAudit{
			PromptVersion:    "ds-ir-ko-v1",
			Model:            "deterministic-local",
			GeneratedAt:      "2026-05-20T09:00:00Z",
			RedactionApplied: true,
		},
	}
	record, err := ruletypes.NewAIStrategyHistoryRecord(strategy)
	require.NoError(t, err)
	return record
}

func TestHistoryStore_UpsertGetByIncidentID(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	rec := makeRecord(t, "inc-1", "fp-1")

	require.NoError(t, store.Upsert(ctx, "org-1", rec))

	got, ok, err := store.GetLatest(ctx, "org-1", ruletypes.AIStrategyHistoryLookupRequest{IncidentID: "inc-1"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, rec, got)
}

func TestHistoryStore_GetByFingerprint(t *testing.T) {
	ctx := context.Background()
	store, _ := newSeededStore(t)

	// Fixture seeds (org-1, inc-1, fp-1).
	got, ok, err := store.GetLatest(ctx, "org-1", ruletypes.AIStrategyHistoryLookupRequest{AlertFingerprint: "fp-1"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "inc-1", got.IncidentID)
}

func TestHistoryStore_CrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store, _ := newSeededStore(t)

	// Fixture seeds (org-A, inc-1, fp-abc). C2 regression check is on org-X.
	_, ok, err := store.GetLatest(ctx, "org-X", ruletypes.AIStrategyHistoryLookupRequest{IncidentID: "inc-1"})
	require.NoError(t, err)
	require.False(t, ok, "C2 regression: cross-tenant history visible via incidentID")

	_, ok, err = store.GetLatest(ctx, "org-X", ruletypes.AIStrategyHistoryLookupRequest{AlertFingerprint: "fp-abc"})
	require.NoError(t, err)
	require.False(t, ok, "C2 regression: cross-tenant history visible via fingerprint")
}

func TestHistoryStore_UpsertOverwrite(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	first := makeRecord(t, "inc-1", "fp-1")
	first.Strategy.Headline = "first"
	require.NoError(t, store.Upsert(ctx, "org-1", first))

	second := makeRecord(t, "inc-1", "fp-1")
	second.Strategy.Headline = "second"
	require.NoError(t, store.Upsert(ctx, "org-1", second))

	got, ok, err := store.GetLatest(ctx, "org-1", ruletypes.AIStrategyHistoryLookupRequest{IncidentID: "inc-1"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "second", got.Strategy.Headline)
}

func TestHistoryStore_EmptyFingerprintAllowed(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	// Two records with empty fingerprint (preview mode) coexist (partial unique index excludes them)
	require.NoError(t, store.Upsert(ctx, "org-1", makeRecord(t, "inc-1", "")))
	require.NoError(t, store.Upsert(ctx, "org-1", makeRecord(t, "inc-2", "")))

	got1, ok, err := store.GetLatest(ctx, "org-1", ruletypes.AIStrategyHistoryLookupRequest{IncidentID: "inc-1"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "inc-1", got1.IncidentID)

	got2, ok, err := store.GetLatest(ctx, "org-1", ruletypes.AIStrategyHistoryLookupRequest{IncidentID: "inc-2"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "inc-2", got2.IncidentID)
}

func TestHistoryStore_InvalidLookupReturnsError(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_, ok, err := store.GetLatest(ctx, "org-1", ruletypes.AIStrategyHistoryLookupRequest{})
	require.Error(t, err)
	require.False(t, ok)
}
