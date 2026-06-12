package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// alterDSAIStrategyHistoryMulti evolves ds_ai_strategy_history so that multiple
// incidents sharing an alert_fingerprint (recurrences of the same failure) can
// coexist as distinct rows, enabling "past N occurrences of the same failure"
// lookups (FR-CF2.6).
//
// Two changes:
//   - drop the UNIQUE (org_id, alert_fingerprint) index — it forced one row per
//     fingerprint and blocked recurrence accumulation — replacing it with a
//     plain (non-unique) index for lookup performance;
//   - add a generated_at column mirroring the record's generatedAt so recency
//     ordering does not have to parse the JSON payload.
//
// The (org_id, incident_id) primary key is unchanged, so re-generating a
// strategy for the same incident still overwrites in place. Pre-existing rows
// keep generated_at = '' (they sort oldest); no JSON backfill is attempted.
type alterDSAIStrategyHistoryMulti struct {
	sqlstore sqlstore.SQLStore
}

func NewAlterDSAIStrategyHistoryMultiFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("alter_ds_ai_strategy_history_multi"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &alterDSAIStrategyHistoryMulti{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *alterDSAIStrategyHistoryMulti) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *alterDSAIStrategyHistoryMulti) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		ALTER TABLE ds_ai_strategy_history
			ADD COLUMN generated_at TEXT NOT NULL DEFAULT ''
	`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_ds_ai_strategy_history_org_fp`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_ds_ai_strategy_history_org_fp
			ON ds_ai_strategy_history(org_id, alert_fingerprint)
			WHERE alert_fingerprint <> ''
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *alterDSAIStrategyHistoryMulti) Down(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_ds_ai_strategy_history_org_fp`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_ds_ai_strategy_history_org_fp
			ON ds_ai_strategy_history(org_id, alert_fingerprint)
			WHERE alert_fingerprint <> ''
	`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		ALTER TABLE ds_ai_strategy_history DROP COLUMN generated_at
	`); err != nil {
		return err
	}

	return tx.Commit()
}
