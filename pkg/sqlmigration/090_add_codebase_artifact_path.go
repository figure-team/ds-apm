package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// addCodebaseArtifactPath widens ds_codebase_repo with artifact_path: the
// local root of the operated project. RCA exports ("ds-navi에 산출물 전송")
// are written under <artifact_path>/ds-hub/.
type addCodebaseArtifactPath struct {
	sqlstore sqlstore.SQLStore
}

func NewAddCodebaseArtifactPathFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_codebase_artifact_path"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addCodebaseArtifactPath{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *addCodebaseArtifactPath) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addCodebaseArtifactPath) Up(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx,
		`ALTER TABLE ds_codebase_repo ADD COLUMN artifact_path TEXT NOT NULL DEFAULT ''`)
	return err
}

func (migration *addCodebaseArtifactPath) Down(ctx context.Context, db *bun.DB) error {
	return nil // additive only
}
