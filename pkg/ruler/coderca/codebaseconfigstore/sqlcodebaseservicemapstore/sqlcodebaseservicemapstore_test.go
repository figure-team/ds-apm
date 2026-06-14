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

func TestServiceMapStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	// Upsert → Get ok → Delete → Get returns NotFound.
	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-1", ServiceName: "payments", RepoID: "repo-pay"}))
	_, err := store.Get(ctx, "org-1", "payments")
	require.NoError(t, err)

	require.NoError(t, store.Delete(ctx, "org-1", "payments"))

	_, err = store.Get(ctx, "org-1", "payments")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseServiceMapNotFound)
}

func TestServiceMapStore_DeleteTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-1", ServiceName: "payments", RepoID: "r1"}))
	require.NoError(t, store.Upsert(ctx, ruletypes.CodebaseServiceMap{OrgID: "org-2", ServiceName: "payments", RepoID: "r2"}))

	// Delete only org-1's row.
	require.NoError(t, store.Delete(ctx, "org-1", "payments"))

	// org-1 row gone.
	_, err := store.Get(ctx, "org-1", "payments")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseServiceMapNotFound)

	// org-2 row still present.
	got, err := store.Get(ctx, "org-2", "payments")
	require.NoError(t, err)
	require.Equal(t, "r2", got.RepoID)
}

func TestServiceMapStore_DeleteIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	// Delete a non-existent row must return nil (idempotent).
	require.NoError(t, store.Delete(ctx, "org-x", "no-such-service"))
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
