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

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ds_codebase_repo (
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
		)`,
		// (org_id, service_name) -> repo_id [+ optional monorepo subpath] (design §8).
		`CREATE TABLE IF NOT EXISTS ds_codebase_service_map (
			org_id        TEXT NOT NULL,
			service_name  TEXT NOT NULL,
			repo_id       TEXT NOT NULL,
			subpath       TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, service_name)
		)`,
		// One RCA run. Timestamps are INTEGER unix seconds for cooldown/lease
		// arithmetic. Lease fields back the DB-backed worker (design §6.3).
		`CREATE TABLE IF NOT EXISTS coderca_run (
			run_id          TEXT    NOT NULL PRIMARY KEY,
			org_id          TEXT    NOT NULL,
			service         TEXT    NOT NULL DEFAULT '',
			dedup_key       TEXT    NOT NULL,
			status          TEXT    NOT NULL,
			baseline_commit TEXT    NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL,
			claimed_by      TEXT    NOT NULL DEFAULT '',
			lease_token     TEXT    NOT NULL DEFAULT '',
			lease_until     INTEGER NOT NULL DEFAULT 0,
			heartbeat_at    INTEGER NOT NULL DEFAULT 0,
			attempts        INTEGER NOT NULL DEFAULT 0,
			finished_at     INTEGER NOT NULL DEFAULT 0,
			result_ref      TEXT    NOT NULL DEFAULT ''
		)`,
		// Dedup linchpin: one row per (org, dedup_key); sliding cooldown via
		// last_admitted_at; hit_count aggregates suppressed duplicates (§6.2/6.4).
		`CREATE TABLE IF NOT EXISTS coderca_admission (
			org_id           TEXT    NOT NULL,
			dedup_key        TEXT    NOT NULL,
			last_admitted_at INTEGER NOT NULL,
			hit_count        INTEGER NOT NULL DEFAULT 0,
			last_run_ref     TEXT    NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, dedup_key)
		)`,
		// Per-(org, day) atomic run counter (§6.2).
		`CREATE TABLE IF NOT EXISTS coderca_budget (
			org_id TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			used   INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, day)
		)`,
		// Locked concurrency semaphore (§6.3).
		`CREATE TABLE IF NOT EXISTS coderca_capacity (
			scope               TEXT    NOT NULL PRIMARY KEY,
			running             INTEGER NOT NULL DEFAULT 0,
			max_concurrent_runs INTEGER NOT NULL DEFAULT 1
		)`,
		// Aggregated skip counters: one row per (org, reason, day) (§6.4).
		`CREATE TABLE IF NOT EXISTS coderca_skip_stat (
			org_id TEXT    NOT NULL,
			reason TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			count  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, reason, day)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (migration *addDSCodebaseConfig) Down(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, table := range []string{
		"coderca_skip_stat", "coderca_capacity", "coderca_budget",
		"coderca_admission", "coderca_run", "ds_codebase_service_map", "ds_codebase_repo",
	} {
		if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			return err
		}
	}
	return tx.Commit()
}
