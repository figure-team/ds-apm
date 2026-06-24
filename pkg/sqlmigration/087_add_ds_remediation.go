package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// addDSRemediation creates the human-gated auto-remediation tables (design §4).
// ds_remediation_execution holds one approve→execute→verify lifecycle row;
// ds_remediation_config is the per-org master switch + timing knobs.
//
// SEAM: register this factory in pkg/signoz/provider.go's migration factory
// list at the integration stage (mirrors 082's comment).
type addDSRemediation struct {
	sqlstore sqlstore.SQLStore
}

func NewAddDSRemediationFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_ds_remediation"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addDSRemediation{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addDSRemediation) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addDSRemediation) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmts := []string{
		// One approve→execute→verify lifecycle. Timestamps are RFC3339 TEXT
		// (mirrors ds_codebase_repo.last_sync_at). exit_code nullable via -1
		// sentinel kept out of band — stored as INTEGER, NULL when unset.
		`CREATE TABLE IF NOT EXISTS ds_remediation_execution (
			id                 TEXT    NOT NULL PRIMARY KEY,
			org_id             TEXT    NOT NULL,
			incident_id        TEXT    NOT NULL DEFAULT '',
			alert_fingerprint  TEXT    NOT NULL DEFAULT '',
			sop_id             TEXT    NOT NULL DEFAULT '',
			sop_version        TEXT    NOT NULL DEFAULT '',
			runbook_id         TEXT    NOT NULL DEFAULT '',
			script_snapshot    TEXT    NOT NULL DEFAULT '',
			status             TEXT    NOT NULL,
			proposed_at        TEXT    NOT NULL DEFAULT '',
			approved_at        TEXT    NOT NULL DEFAULT '',
			executed_at        TEXT    NOT NULL DEFAULT '',
			terminal_at        TEXT    NOT NULL DEFAULT '',
			approved_by        TEXT    NOT NULL DEFAULT '',
			exit_code          INTEGER,
			output_snippet     TEXT    NOT NULL DEFAULT '',
			verify_result      TEXT    NOT NULL DEFAULT '',
			expires_at         TEXT    NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ds_remediation_org_incident
			ON ds_remediation_execution (org_id, incident_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ds_remediation_org_status
			ON ds_remediation_execution (org_id, status)`,
		// Per-org master switch + timing knobs (design §6). One row per org.
		`CREATE TABLE IF NOT EXISTS ds_remediation_config (
			org_id                 TEXT    NOT NULL PRIMARY KEY,
			execution_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
			proposal_ttl_seconds   INTEGER NOT NULL DEFAULT 1800,
			exec_timeout_seconds   INTEGER NOT NULL DEFAULT 300,
			verify_window_seconds  INTEGER NOT NULL DEFAULT 600,
			max_concurrent         INTEGER NOT NULL DEFAULT 1
		)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (migration *addDSRemediation) Down(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, table := range []string{"ds_remediation_config", "ds_remediation_execution"} {
		if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			return err
		}
	}
	return tx.Commit()
}
