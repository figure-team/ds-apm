package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addDSAIConfig struct {
	sqlstore sqlstore.SQLStore
}

func NewAddDSAIConfigFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_ds_ai_config"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addDSAIConfig{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addDSAIConfig) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addDSAIConfig) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ds_ai_config (
			org_id              TEXT      NOT NULL PRIMARY KEY,
			provider            TEXT      NOT NULL,
			llm_provider        TEXT      NOT NULL DEFAULT '',
			transport           TEXT      NOT NULL DEFAULT '',
			model               TEXT      NOT NULL DEFAULT '',
			api_key_ciphertext  TEXT      NOT NULL DEFAULT '',
			binary_path         TEXT      NOT NULL DEFAULT '',
			timeout_seconds     INTEGER   NOT NULL DEFAULT 0,
			updated_at          TEXT      NOT NULL
		)
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *addDSAIConfig) Down(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS ds_ai_config`); err != nil {
		return err
	}
	return tx.Commit()
}
