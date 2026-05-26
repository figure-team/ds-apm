// Package sqlitesqlstoretest provides a real in-memory SQLite SQLStore for
// integration tests. Unlike sqlstoretest (which uses go-sqlmock to verify
// emitted SQL strings), this helper opens a real SQLite database, lets
// callers apply their migration DDL, and returns a fully functional
// SQLStore that exercises actual SQL semantics. Use this when a test needs
// to verify behavior end-to-end (e.g., PK uniqueness, partial indexes,
// cross-tenant isolation).
package sqlitesqlstoretest

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstore"
	"github.com/stretchr/testify/require"
)

// New returns a real in-memory SQLite SQLStore. The database is empty —
// the caller is responsible for applying any DDL they need (typically by
// invoking a migration's Up method against the returned store's BunDB).
//
// Each call returns an isolated :memory: database; tests cannot leak state
// to each other.
func New(t *testing.T) sqlstore.SQLStore {
	t.Helper()
	ctx := context.Background()

	cfg := sqlstore.Config{
		Provider: "sqlite",
		Sqlite: sqlstore.SqliteConfig{
			Path:            ":memory:",
			BusyTimeout:     time.Second,
			Mode:            "WAL",
			TransactionMode: "deferred",
		},
		Connection: sqlstore.ConnectionConfig{
			MaxOpenConns:    1,
			MaxConnLifetime: time.Hour,
		},
	}

	providerSettings := instrumentationtest.New().ToProviderSettings()
	store, err := sqlitesqlstore.New(ctx, factory.ProviderSettings(providerSettings), cfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = store.SQLDB().Close()
	})

	return store
}
