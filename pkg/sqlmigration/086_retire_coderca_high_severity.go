package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// retireCodercaHighSeverity (CF-11) retires the "high" min-severity level so
// the Code RCA gate shares one severity vocabulary with alert routing
// (critical|error|warning|info). "high" ranked equal to "error", so existing
// rows storing min_severity='high' are converted to 'error' — a lossless
// rewrite. Without this, removing "high" from severityRank would make the gate
// fail-closed for those orgs (unknown level ranks 0).
//
// The ds_codebase_config.min_severity column DEFAULT stays 'high' in the DDL,
// but it is dormant: the store always writes min_severity explicitly (Upsert)
// and fresh orgs get the Go default (DefaultCodebaseRCAConfig → "error"), so
// the column default is never materialized.
type retireCodercaHighSeverity struct {
	sqlstore sqlstore.SQLStore
}

func NewRetireCodercaHighSeverityFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("retire_coderca_high_severity"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &retireCodercaHighSeverity{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *retireCodercaHighSeverity) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *retireCodercaHighSeverity) Up(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx,
		`UPDATE ds_codebase_config SET min_severity = 'error' WHERE min_severity = 'high'`)
	return err
}

func (migration *retireCodercaHighSeverity) Down(ctx context.Context, db *bun.DB) error {
	return nil // data-only; not reversible
}
