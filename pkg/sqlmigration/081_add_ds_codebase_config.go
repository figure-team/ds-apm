package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// addDSCodebaseConfig creates the CF-11 (code RCA) tables. The cost-control
// tables (coderca_run/admission/budget/capacity/skip_stat) are added in the
// admission/lease milestone; this migration starts with repo registration.
//
// SEAM (design §11): register this factory in
// pkg/signoz/provider.go > NewSQLMigrationProviderFactories at the integration
// stage. It is intentionally NOT registered from this worktree.
type addDSCodebaseConfig struct {
	sqlstore sqlstore.SQLStore
}

func NewAddDSCodebaseConfigFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_ds_codebase_config"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addDSCodebaseConfig{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addDSCodebaseConfig) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addDSCodebaseConfig) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ds_codebase_repo (
			org_id                 TEXT      NOT NULL,
			repo_id                TEXT      NOT NULL,
			git_url                TEXT      NOT NULL,
			default_branch         TEXT      NOT NULL DEFAULT '',
			credential_ciphertext  TEXT      NOT NULL DEFAULT '',
			enabled                BOOLEAN   NOT NULL DEFAULT FALSE,
			branch_name            TEXT      NOT NULL DEFAULT '',
			fetched                BOOLEAN   NOT NULL DEFAULT FALSE,
			baseline_commit        TEXT      NOT NULL DEFAULT '',
			last_sync_at           TEXT      NOT NULL DEFAULT '',
			last_sync_status       TEXT      NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, repo_id)
		)
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *addDSCodebaseConfig) Down(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS ds_codebase_repo`); err != nil {
		return err
	}
	return tx.Commit()
}
