package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addDSApmStores struct {
	sqlstore sqlstore.SQLStore
}

func NewAddDSApmStoresFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_ds_apm_stores"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addDSApmStores{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addDSApmStores) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addDSApmStores) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ds_sop_documents (
			org_id              TEXT      NOT NULL,
			sop_id              TEXT      NOT NULL,
			version             TEXT      NOT NULL,
			contract_version    TEXT      NOT NULL,
			title               TEXT      NOT NULL,
			updated_at          TEXT      NOT NULL,
			payload             TEXT      NOT NULL,
			PRIMARY KEY (org_id, sop_id, version)
		)
	`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_ds_sop_documents_org_id_sop_id
			ON ds_sop_documents(org_id, sop_id)
	`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ds_ai_strategy_history (
			org_id              TEXT      NOT NULL,
			incident_id         TEXT      NOT NULL,
			alert_fingerprint   TEXT      NOT NULL DEFAULT '',
			contract_version    TEXT      NOT NULL,
			payload             TEXT      NOT NULL,
			PRIMARY KEY (org_id, incident_id)
		)
	`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_ds_ai_strategy_history_org_fp
			ON ds_ai_strategy_history(org_id, alert_fingerprint)
			WHERE alert_fingerprint <> ''
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *addDSApmStores) Down(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS ds_ai_strategy_history`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS ds_sop_documents`); err != nil {
		return err
	}
	return tx.Commit()
}
