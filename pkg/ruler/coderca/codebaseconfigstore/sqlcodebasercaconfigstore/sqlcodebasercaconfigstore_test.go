package sqlcodebasercaconfigstore

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// applyDSCodebaseConfigDDL mirrors migration 083 (production registers the
// migration via a seam; tests apply DDL directly, per the ai_config pattern).
func applyDSCodebaseConfigDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	_, err := ss.BunDB().ExecContext(ctx, `
		CREATE TABLE ds_codebase_config (
			org_id                        TEXT    NOT NULL PRIMARY KEY,
			enabled                       BOOLEAN NOT NULL DEFAULT FALSE,
			min_severity                  TEXT    NOT NULL DEFAULT 'high',
			cooldown_window_secs          INTEGER NOT NULL DEFAULT 21600,
			max_runs_per_day              INTEGER NOT NULL DEFAULT 20,
			max_queue_depth               INTEGER NOT NULL DEFAULT 50,
			max_concurrent_runs           INTEGER NOT NULL DEFAULT 1,
			allow_unbound_without_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at                    TEXT    NOT NULL DEFAULT ''
		)`)
	return err
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyDSCodebaseConfigDDL(context.Background(), ss))
	return New(ss)
}

func TestRCAConfigUpsertGetRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "org-1")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRCAConfigNotFound)

	cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
	cfg.Enabled = true
	cfg.MaxRunsPerDay = 5
	require.NoError(t, store.Upsert(ctx, cfg))

	got, err := store.Get(ctx, "org-1")
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, 5, got.MaxRunsPerDay)

	// 업데이트 경로: 같은 org 재-Upsert가 INSERT가 아니라 UPDATE로 동작
	cfg.MaxRunsPerDay = 7
	require.NoError(t, store.Upsert(ctx, cfg))
	got, err = store.Get(ctx, "org-1")
	require.NoError(t, err)
	assert.Equal(t, 7, got.MaxRunsPerDay)

	// 테넌트 격리: 다른 org는 not-found
	_, err = store.Get(ctx, "org-2")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRCAConfigNotFound)
}
