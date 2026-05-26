package sqlsopstore

import (
	"context"
	"testing"

	signoztf "github.com/SigNoz/signoz/pkg/testfixtures"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) ruletypes.SOPStore {
	t.Helper()
	ctx := context.Background()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyMigration078(ctx, ss))
	return NewSOPStore(ss)
}

// newSeededStore returns a store pre-loaded from tests/fixtures/go/ds_sop_documents.yml.
func newSeededStore(t *testing.T) (ruletypes.SOPStore, sqlstore.SQLStore) {
	t.Helper()
	ctx := context.Background()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyMigration078(ctx, ss))
	signoztf.Load(t, ss, signoztf.DefaultFixtureDir(), "ds_sop_documents")
	return NewSOPStore(ss), ss
}

func applyMigration078(ctx context.Context, ss sqlstore.SQLStore) error {
	// Mirror the DDL from sqlmigration/078_add_ds_apm_stores.go Up()
	stmts := []string{
		`CREATE TABLE ds_sop_documents (
			org_id              TEXT      NOT NULL,
			sop_id              TEXT      NOT NULL,
			version             TEXT      NOT NULL,
			contract_version    TEXT      NOT NULL,
			title               TEXT      NOT NULL,
			updated_at          TEXT      NOT NULL,
			payload             TEXT      NOT NULL,
			PRIMARY KEY (org_id, sop_id, version)
		)`,
		`CREATE INDEX idx_ds_sop_documents_org_id_sop_id ON ds_sop_documents(org_id, sop_id)`,
	}
	for _, stmt := range stmts {
		if _, err := ss.BunDB().ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func makeDoc(sopID, version string) ruletypes.SOPDocument {
	return ruletypes.SOPDocument{
		ContractVersion: ruletypes.SOPDocumentContractVersion,
		SOPID:           sopID,
		Version:         version,
		Title:           "T-" + sopID,
		BodyMarkdown:    "## step 1\nbody",
		UpdatedAt:       "2026-05-20T09:00:00Z",
		TenantScope: ruletypes.PilotTenantScope{
			ProjectIDs:   []string{"p"},
			Environments: []string{"prod"},
		},
	}
}

func TestSOPStore_UpsertGet(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	doc := makeDoc("S1", "v1")

	require.NoError(t, store.Upsert(ctx, "org-1", doc))

	got, err := store.Get(ctx, "org-1", "S1", "v1")
	require.NoError(t, err)
	require.Equal(t, doc, got)
}

func TestSOPStore_CrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store, _ := newSeededStore(t)

	// Fixture seeds org-A/S1/v1 + org-B/S1/v1. C1 regression check is on org-X (a tenant
	// not present in the fixture) — must see nothing for either org-A's or org-B's keys.
	list, err := store.List(ctx, "org-X")
	require.NoError(t, err)
	require.Empty(t, list, "C1 regression: cross-tenant SOP visible via List")

	_, err = store.Get(ctx, "org-X", "S1", "v1")
	require.ErrorIs(t, err, ruletypes.ErrSOPDocumentNotFound, "C1 regression: Get returned cross-tenant data")

	_, err = store.GetLatest(ctx, "org-X", "S1")
	require.ErrorIs(t, err, ruletypes.ErrSOPDocumentNotFound, "C1 regression: GetLatest returned cross-tenant data")
}

func TestSOPStore_SameSopIDDifferentOrgs(t *testing.T) {
	ctx := context.Background()
	store, _ := newSeededStore(t)

	gotA, err := store.Get(ctx, "org-A", "S1", "v1")
	require.NoError(t, err)
	require.Equal(t, "org-A doc", gotA.Title)

	gotB, err := store.Get(ctx, "org-B", "S1", "v1")
	require.NoError(t, err)
	require.Equal(t, "org-B doc", gotB.Title)
}

func TestSOPStore_VersionCoexistence(t *testing.T) {
	ctx := context.Background()
	store, _ := newSeededStore(t)

	// Fixture seeds org-1/S1/v1 + org-1/S1/v2.
	got, err := store.GetLatest(ctx, "org-1", "S1")
	require.NoError(t, err)
	require.Equal(t, "v2", got.Version)

	list, err := store.List(ctx, "org-1")
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestSOPStore_UpsertIdempotency(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	doc := makeDoc("S1", "v1")
	doc.Title = "first"
	require.NoError(t, store.Upsert(ctx, "org-1", doc))

	doc.Title = "second"
	require.NoError(t, store.Upsert(ctx, "org-1", doc))

	list, err := store.List(ctx, "org-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "second", list[0].Title)
}

func TestSOPStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	require.NoError(t, store.Upsert(ctx, "org-1", makeDoc("S1", "v1")))
	require.NoError(t, store.Delete(ctx, "org-1", "S1", "v1"))

	_, err := store.Get(ctx, "org-1", "S1", "v1")
	require.ErrorIs(t, err, ruletypes.ErrSOPDocumentNotFound)

	err = store.Delete(ctx, "org-1", "missing", "v1")
	require.ErrorIs(t, err, ruletypes.ErrSOPDocumentNotFound)
}

func TestSOPStore_UpsertRunbook_AddsToExistingDocument(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	doc := makeDoc("SOP-RB-001", "v01")
	require.NoError(t, store.Upsert(ctx, "org-1", doc))

	rb := ruletypes.Runbook{
		ID:               "01928374-5566-77ab-89cd-eeff00112233",
		Title:            "Restart",
		Description:      "x",
		ExecutableScript: "#!/bin/bash\necho hi\n",
		Status:           ruletypes.RunbookStatusDraft,
		Confidence:       0.7,
		CreatedAt:        "2026-05-22T00:00:00Z",
		UpdatedAt:        "2026-05-22T00:00:00Z",
		UpdatedBy:        "ai",
	}
	require.NoError(t, store.UpsertRunbook(ctx, "org-1", "SOP-RB-001", "v01", rb))

	got, err := store.Get(ctx, "org-1", "SOP-RB-001", "v01")
	require.NoError(t, err)
	require.Len(t, got.Runbooks, 1)
	require.Equal(t, rb.ID, got.Runbooks[0].ID)
}

func TestSOPStore_UpsertRunbook_UpdatesExisting(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	doc := makeDoc("SOP-RB-002", "v01")
	doc.Runbooks = []ruletypes.Runbook{{
		ID:               "01928374-5566-77ab-89cd-eeff00112234",
		Title:            "Original",
		Description:      "x",
		ExecutableScript: "#!/bin/bash\necho old\n",
		Status:           ruletypes.RunbookStatusDraft,
		Confidence:       0.5,
		CreatedAt:        "2026-05-22T00:00:00Z",
		UpdatedAt:        "2026-05-22T00:00:00Z",
		UpdatedBy:        "ai",
	}}
	require.NoError(t, store.Upsert(ctx, "org-1", doc))

	updated := doc.Runbooks[0]
	updated.Title = "Updated"
	updated.UpdatedAt = "2026-05-22T01:00:00Z"
	require.NoError(t, store.UpsertRunbook(ctx, "org-1", "SOP-RB-002", "v01", updated))

	got, err := store.Get(ctx, "org-1", "SOP-RB-002", "v01")
	require.NoError(t, err)
	require.Len(t, got.Runbooks, 1)
	require.Equal(t, "Updated", got.Runbooks[0].Title)
}

func TestSOPStore_UpsertRunbook_ParentSOPNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	rb := ruletypes.Runbook{ID: "01928374-5566-77ab-89cd-eeff00112233", Title: "x"}
	err := store.UpsertRunbook(ctx, "org-1", "missing-sop", "v01", rb)
	require.ErrorIs(t, err, ruletypes.ErrSOPDocumentNotFound)
}

func TestSOPStore_DeleteRunbook_RemovesEntry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	doc := makeDoc("SOP-RB-003", "v01")
	doc.Runbooks = []ruletypes.Runbook{
		{ID: "01928374-5566-77ab-89cd-eeff00112235", Title: "A",
			Status: ruletypes.RunbookStatusDraft, ExecutableScript: "#!/bin/bash\nhi\n",
			CreatedAt: "2026-05-22T00:00:00Z", UpdatedAt: "2026-05-22T00:00:00Z",
			UpdatedBy: "ai", Confidence: 0.5},
		{ID: "01928374-5566-77ab-89cd-eeff00112236", Title: "B",
			Status: ruletypes.RunbookStatusDraft, ExecutableScript: "#!/bin/bash\nhi\n",
			CreatedAt: "2026-05-22T00:00:00Z", UpdatedAt: "2026-05-22T00:00:00Z",
			UpdatedBy: "ai", Confidence: 0.5},
	}
	require.NoError(t, store.Upsert(ctx, "org-1", doc))

	require.NoError(t, store.DeleteRunbook(ctx, "org-1", "SOP-RB-003", "v01", "01928374-5566-77ab-89cd-eeff00112235"))

	got, err := store.Get(ctx, "org-1", "SOP-RB-003", "v01")
	require.NoError(t, err)
	require.Len(t, got.Runbooks, 1)
	require.Equal(t, "B", got.Runbooks[0].Title)
}

func TestSOPStore_DeleteRunbook_RunbookNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Seed an SOP with one runbook so the parent exists but a different
	// runbookID is requested for delete.
	doc := makeDoc("SOP-RB-004", "v01")
	doc.Runbooks = []ruletypes.Runbook{{
		ID:               "01928374-5566-77ab-89cd-eeff00112237",
		Title:            "Existing",
		Status:           ruletypes.RunbookStatusDraft,
		ExecutableScript: "#!/bin/bash\nhi\n",
		Confidence:       0.5,
		CreatedAt:        "2026-05-22T00:00:00Z",
		UpdatedAt:        "2026-05-22T00:00:00Z",
		UpdatedBy:        "ai",
	}}
	require.NoError(t, store.Upsert(ctx, "org-1", doc))

	err := store.DeleteRunbook(ctx, "org-1", "SOP-RB-004", "v01",
		"01928374-5566-77ab-89cd-eeff999fffff") // not the seeded ID
	require.ErrorIs(t, err, ruletypes.ErrSOPDocumentNotFound)

	// Confirm the existing runbook was not removed (i.e., the failed
	// delete didn't accidentally mutate state).
	got, err := store.Get(ctx, "org-1", "SOP-RB-004", "v01")
	require.NoError(t, err)
	require.Len(t, got.Runbooks, 1)
}
