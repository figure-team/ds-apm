package sqlcodebaseservicemapstore

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// applyServiceMapDDL mirrors migration 081 (tests apply DDL directly; production
// registers the migration via a seam).
func applyServiceMapDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	_, err := ss.BunDB().ExecContext(ctx, `
		CREATE TABLE ds_codebase_service_map (
			org_id        TEXT NOT NULL,
			service_name  TEXT NOT NULL,
			repo_id       TEXT NOT NULL,
			subpath       TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, service_name)
		)`)
	return err
}

func newStore(t *testing.T) ruletypes.CodebaseServiceMapStore {
	t.Helper()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyServiceMapDDL(context.Background(), ss))
	return New(ss)
}

func TestServiceMapStore_UpsertGet(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{
		OrgID: "org-1", ServiceName: "payments", RepoID: "repo-pay", Subpath: "services/pay",
	}))

	got, err := store.Get(ctx, "org-1", "payments")
	require.NoError(t, err)
	require.Equal(t, "repo-pay", got.RepoID)
	require.Equal(t, "services/pay", got.Subpath)
}

func TestServiceMapStore_GetNotFound(t *testing.T) {
	store := newStore(t)
	_, err := store.Get(context.Background(), "org-x", "missing")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseServiceMapNotFound)
}

func TestServiceMapStore_CrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-A", ServiceName: "payments", RepoID: "rA"}))

	_, err := store.Get(ctx, "org-B", "payments")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseServiceMapNotFound)
}

func TestServiceMapStore_ListIsOrgScoped(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-A", ServiceName: "payments", RepoID: "r1"}))
	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-A", ServiceName: "orders", RepoID: "r2"}))
	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-B", ServiceName: "billing", RepoID: "r3"}))

	got, err := store.List(ctx, "org-A")
	require.NoError(t, err)
	require.Len(t, got, 2, "List must be org-scoped")
}

func TestServiceMapStore_UpsertOverwrites(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-1", ServiceName: "payments", RepoID: "old"}))
	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-1", ServiceName: "payments", RepoID: "new", Subpath: "x"}))

	got, err := store.Get(ctx, "org-1", "payments")
	require.NoError(t, err)
	require.Equal(t, "new", got.RepoID)
	require.Equal(t, "x", got.Subpath)
}
