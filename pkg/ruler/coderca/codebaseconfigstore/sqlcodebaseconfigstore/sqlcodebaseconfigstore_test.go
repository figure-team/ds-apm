package sqlcodebaseconfigstore

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func idEnc(s string) (string, error) { return s, nil }
func idDec(s string) (string, error) { return s, nil }

// applyCodebaseRepoDDL mirrors migration 081 (production registers the
// migration via a seam; tests apply DDL directly, per the ai_config pattern).
func applyCodebaseRepoDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	_, err := ss.BunDB().ExecContext(ctx, `
		CREATE TABLE ds_codebase_repo (
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
			artifact_path          TEXT      NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, repo_id)
		)`)
	return err
}

func newStore(t *testing.T) ruletypes.CodebaseRepoStore {
	t.Helper()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyCodebaseRepoDDL(context.Background(), ss))
	return New(ss)
}

func makeRepo(orgID, repoID string) ruletypes.CodebaseRepo {
	return ruletypes.CodebaseRepo{
		ContractVersion: ruletypes.CodebaseRepoContractVersion,
		OrgID:           orgID,
		RepoID:          repoID,
		GitURL:          "https://github.com/acme/" + repoID + ".git",
		DefaultBranch:   "main",
		Credential:      "tok-" + orgID,
		Enabled:         true,
	}
}

func TestCodebaseRepoStore_UpsertGet(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	repo := makeRepo("org-1", "payments")
	require.NoError(t, store.Upsert(ctx, repo, idEnc))

	got, err := store.Get(ctx, "org-1", "payments", idDec)
	require.NoError(t, err)
	require.Equal(t, repo.GitURL, got.GitURL)
	require.Equal(t, repo.DefaultBranch, got.DefaultBranch)
	require.Equal(t, repo.Credential, got.Credential, "credential must survive encrypt->store->decrypt")
	require.True(t, got.Enabled)
}

func TestCodebaseRepoStore_GetNotFound(t *testing.T) {
	store := newStore(t)
	_, err := store.Get(context.Background(), "org-x", "missing", idDec)
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRepoNotFound)
}

func TestCodebaseRepoStore_CrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, makeRepo("org-A", "payments"), idEnc))

	// org-B must not see org-A's repo, even with the same repo_id.
	_, err := store.Get(ctx, "org-B", "payments", idDec)
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRepoNotFound)
}

func TestCodebaseRepoStore_List(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, makeRepo("org-A", "payments"), idEnc))
	require.NoError(t, store.Upsert(ctx, makeRepo("org-A", "orders"), idEnc))
	require.NoError(t, store.Upsert(ctx, makeRepo("org-B", "billing"), idEnc))

	got, err := store.List(ctx, "org-A", idDec)
	require.NoError(t, err)
	require.Len(t, got, 2, "List must be org-scoped")
}

func TestCodebaseRepoStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	// Upsert → Get ok → Delete → Get returns NotFound.
	require.NoError(t, store.Upsert(ctx, makeRepo("org-1", "payments"), idEnc))
	_, err := store.Get(ctx, "org-1", "payments", idDec)
	require.NoError(t, err)

	require.NoError(t, store.Delete(ctx, "org-1", "payments"))

	_, err = store.Get(ctx, "org-1", "payments", idDec)
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRepoNotFound)
}

func TestCodebaseRepoStore_DeleteTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, makeRepo("org-1", "payments"), idEnc))
	require.NoError(t, store.Upsert(ctx, makeRepo("org-2", "payments"), idEnc))

	// Delete only org-1's row.
	require.NoError(t, store.Delete(ctx, "org-1", "payments"))

	// org-1 row gone.
	_, err := store.Get(ctx, "org-1", "payments", idDec)
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRepoNotFound)

	// org-2 row still present.
	got, err := store.Get(ctx, "org-2", "payments", idDec)
	require.NoError(t, err)
	require.Equal(t, "org-2", got.OrgID)
}

func TestCodebaseRepoStore_DeleteIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	// Delete a non-existent row must return nil (idempotent).
	require.NoError(t, store.Delete(ctx, "org-x", "no-such-repo"))
}

func TestCodebaseRepoStore_UpsertOverwriteAndSourceState(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	require.NoError(t, store.Upsert(ctx, makeRepo("org-1", "payments"), idEnc))

	// Source-state update via re-upsert (ON CONFLICT path).
	updated := makeRepo("org-1", "payments")
	updated.DefaultBranch = "develop"
	updated.Fetched = true
	updated.BaselineCommit = "deadbeef"
	updated.LastSyncStatus = "ok"
	require.NoError(t, store.Upsert(ctx, updated, idEnc))

	got, err := store.Get(ctx, "org-1", "payments", idDec)
	require.NoError(t, err)
	require.Equal(t, "develop", got.DefaultBranch)
	require.True(t, got.Fetched)
	require.Equal(t, "deadbeef", got.BaselineCommit)
	require.Equal(t, "ok", got.LastSyncStatus)
}
