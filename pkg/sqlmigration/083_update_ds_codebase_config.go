package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// updateDSCodebaseConfig (CF-11 integration stage) adds the per-org config
// table that design §8 named `codebase_config` (migration 082 shipped only the
// repo/map/cost tables) and widens coderca_run with the persisted RCA report
// so the run-history API can serve report bodies.
type updateDSCodebaseConfig struct {
	sqlstore sqlstore.SQLStore
}

func NewUpdateDSCodebaseConfigFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("update_ds_codebase_config"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &updateDSCodebaseConfig{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *updateDSCodebaseConfig) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *updateDSCodebaseConfig) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ds_codebase_config (
			org_id                        TEXT    NOT NULL PRIMARY KEY,
			enabled                       BOOLEAN NOT NULL DEFAULT FALSE,
			min_severity                  TEXT    NOT NULL DEFAULT 'high',
			cooldown_window_secs          INTEGER NOT NULL DEFAULT 21600,
			max_runs_per_day              INTEGER NOT NULL DEFAULT 20,
			max_queue_depth               INTEGER NOT NULL DEFAULT 50,
			max_concurrent_runs           INTEGER NOT NULL DEFAULT 1,
			allow_unbound_without_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at                    TEXT    NOT NULL DEFAULT ''
		)`,
		`ALTER TABLE coderca_run ADD COLUMN root_cause   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE coderca_run ADD COLUMN proposed_fix TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE coderca_run ADD COLUMN confidence   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE coderca_run ADD COLUMN limitations  TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (migration *updateDSCodebaseConfig) Down(ctx context.Context, db *bun.DB) error {
	return nil // additive only
}
