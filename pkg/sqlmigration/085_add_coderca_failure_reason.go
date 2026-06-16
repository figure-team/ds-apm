package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// addCodercaFailureReason (CF-11) widens coderca_run with a failure_reason
// column so a non-done run (no_repo_mapping, source-prepare/git error, CLI
// failure/timeout) persists *why* it failed. Until now the reason existed only
// in a fire-and-forget audit event, so the run-history UI could show "failed"
// with no explanation.
type addCodercaFailureReason struct {
	sqlstore sqlstore.SQLStore
}

func NewAddCodercaFailureReasonFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_coderca_failure_reason"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addCodercaFailureReason{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addCodercaFailureReason) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addCodercaFailureReason) Up(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx,
		`ALTER TABLE coderca_run ADD COLUMN failure_reason TEXT NOT NULL DEFAULT ''`)
	return err
}

func (migration *addCodercaFailureReason) Down(ctx context.Context, db *bun.DB) error {
	return nil // additive only
}
