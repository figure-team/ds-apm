// Package testfixtures provides a thin wrapper over go-testfixtures/testfixtures
// for loading canonical seed rows into a real-SQLite SQLStore created by
// pkg/sqlstore/sqlitesqlstoretest. It exists so individual store packages
// don't repeat boilerplate around dialect, paths, and the in-memory DB
// safety check.
package testfixtures

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	tf "github.com/go-testfixtures/testfixtures/v3"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/sqlstore"
)

// DefaultFixtureDir returns the canonical fixture directory (tests/fixtures/go)
// resolved from this source file's location, so callers don't have to know
// the repo layout.
func DefaultFixtureDir() string {
	_, here, _, _ := runtime.Caller(0)
	// here = <repo>/pkg/testfixtures/loader.go
	return filepath.Join(filepath.Dir(here), "..", "..", "tests", "fixtures", "go")
}

// Load wipes and reseeds the given tables from YAML files in fixtureDir.
// Each entry in tables is a base filename without extension; the loader
// expects fixtureDir/<table>.yml. Use this in tests that want a known
// initial DB state — typically read-path tests. Mutation tests are better
// served by inline helpers.
func Load(t *testing.T, store sqlstore.SQLStore, fixtureDir string, tables ...string) {
	t.Helper()
	require.NotEmpty(t, tables, "testfixtures.Load: at least one table is required")

	files := make([]string, 0, len(tables))
	for _, table := range tables {
		files = append(files, filepath.Join(fixtureDir, table+".yml"))
	}

	loader, err := tf.New(
		tf.Database(store.SQLDB()),
		tf.Dialect("sqlite"),
		tf.Files(files...),
		// :memory: sqlite has no name suffix to match; skip the guard.
		tf.DangerousSkipTestDatabaseCheck(),
	)
	require.NoError(t, err, "testfixtures: new loader")
	require.NoError(t, loader.Load(), fmt.Sprintf("testfixtures: load %v", tables))
}
