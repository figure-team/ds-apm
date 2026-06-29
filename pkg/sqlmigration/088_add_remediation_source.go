package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// addRemediationSource adds the script-origin columns to ds_remediation_execution
// (design §6.1/§6.2). Additive: existing rows default to source='runbook' so the
// legacy first-approved behaviour is preserved.
type addRemediationSource struct {
	sqlstore sqlstore.SQLStore
}

func NewAddRemediationSourceFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_remediation_source"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addRemediationSource{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addRemediationSource) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addRemediationSource) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// columnExists guards re-runs: ALTER ADD COLUMN is not IF NOT EXISTS across
	// all dialects, so check the information first via a harmless SELECT.
	addColumn := func(name, ddl string) error {
		if _, err := tx.ExecContext(ctx, "SELECT "+name+" FROM ds_remediation_execution LIMIT 1"); err == nil {
			return nil // column already present
		}
		_, err := tx.ExecContext(ctx, ddl)
		return err
	}

	if err := addColumn("source",
		`ALTER TABLE ds_remediation_execution ADD COLUMN source TEXT NOT NULL DEFAULT 'runbook'`); err != nil {
		return err
	}
	if err := addColumn("selection_rationale",
		`ALTER TABLE ds_remediation_execution ADD COLUMN selection_rationale TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	return tx.Commit()
}

func (migration *addRemediationSource) Down(ctx context.Context, db *bun.DB) error {
	// SQLite lacks DROP COLUMN on older versions; the columns are additive and
	// harmless, so Down is a no-op (mirrors other additive DS migrations).
	return nil
}
