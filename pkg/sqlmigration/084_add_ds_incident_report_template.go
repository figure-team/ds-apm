package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addDSIncidentReportTemplate struct {
	sqlstore sqlstore.SQLStore
}

// NewAddDSIncidentReportTemplateFactory creates the per-org incident-report
// template table. The managed 양식 (a Go text/template) is stored here so each
// org can override the built-in layout.
func NewAddDSIncidentReportTemplateFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_ds_incident_report_template"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addDSIncidentReportTemplate{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addDSIncidentReportTemplate) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addDSIncidentReportTemplate) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ds_incident_report_template (
			org_id      TEXT NOT NULL PRIMARY KEY,
			template    TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		)
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *addDSIncidentReportTemplate) Down(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS ds_incident_report_template`); err != nil {
		return err
	}
	return tx.Commit()
}
