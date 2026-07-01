package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// addRemediationTarget creates ds_remediation_target and adds the frozen target
// snapshot columns to ds_remediation_execution (design §3.1/§3.2). Additive.
type addRemediationTarget struct {
	sqlstore sqlstore.SQLStore
}

func NewAddRemediationTargetFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_remediation_target"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addRemediationTarget{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addRemediationTarget) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addRemediationTarget) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS ds_remediation_target (
  id TEXT NOT NULL,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  host TEXT NOT NULL,
  port INTEGER NOT NULL DEFAULT 22,
  ssh_user TEXT NOT NULL,
  sealed_credential TEXT NOT NULL,
  credential_kind TEXT NOT NULL DEFAULT 'private_key',
  host_key_fingerprint TEXT NOT NULL,
  service_selectors TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (org_id, id)
)`); err != nil {
		return err
	}

	// Frozen target-snapshot columns on execution (design §3.2). Additive; guard re-runs.
	addColumn := func(name, ddl string) error {
		if _, err := tx.ExecContext(ctx, "SELECT "+name+" FROM ds_remediation_execution LIMIT 1"); err == nil {
			return nil
		}
		_, err := tx.ExecContext(ctx, ddl)
		return err
	}
	cols := []struct{ name, ddl string }{
		{"target_id", `ALTER TABLE ds_remediation_execution ADD COLUMN target_id TEXT NOT NULL DEFAULT ''`},
		{"target_host", `ALTER TABLE ds_remediation_execution ADD COLUMN target_host TEXT NOT NULL DEFAULT ''`},
		{"target_port", `ALTER TABLE ds_remediation_execution ADD COLUMN target_port INTEGER NOT NULL DEFAULT 0`},
		{"target_ssh_user", `ALTER TABLE ds_remediation_execution ADD COLUMN target_ssh_user TEXT NOT NULL DEFAULT ''`},
		{"target_host_key_fp", `ALTER TABLE ds_remediation_execution ADD COLUMN target_host_key_fp TEXT NOT NULL DEFAULT ''`},
		{"target_name", `ALTER TABLE ds_remediation_execution ADD COLUMN target_name TEXT NOT NULL DEFAULT ''`},
	}
	for _, c := range cols {
		if err := addColumn(c.name, c.ddl); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (migration *addRemediationTarget) Down(ctx context.Context, db *bun.DB) error {
	// Additive; SQLite DROP COLUMN unsupported on older versions. No-op (mirrors 088).
	return nil
}
