package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addAIOAuthToken struct {
	sqlstore sqlstore.SQLStore
}

func NewAddAIOAuthTokenFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_ai_oauth_token"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addAIOAuthToken{sqlstore: sqlstore}, nil
		},
	)
}

func (m *addAIOAuthToken) Register(migrations *migrate.Migrations) error {
	return migrations.Register(m.Up, m.Down)
}

func (m *addAIOAuthToken) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		ALTER TABLE ds_ai_config
		ADD COLUMN oauth_token_ciphertext TEXT NOT NULL DEFAULT ''
	`); err != nil {
		return err
	}
	return tx.Commit()
}

func (m *addAIOAuthToken) Down(ctx context.Context, db *bun.DB) error {
	// SQLite older than 3.35 lacks DROP COLUMN; safer to leave the column on
	// downgrade. Postgres supports it but consistency wins for a no-op down.
	return nil
}
