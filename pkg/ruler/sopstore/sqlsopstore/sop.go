package sqlsopstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type sopStore struct {
	sqlstore sqlstore.SQLStore
}

// NewSOPStore returns a SOPStore backed by the given SQLStore. Migration
// 078 must have run; the table ds_sop_documents is read directly via
// bun ORM.
func NewSOPStore(store sqlstore.SQLStore) ruletypes.SOPStore {
	return &sopStore{sqlstore: store}
}

func (s *sopStore) Upsert(ctx context.Context, orgID string, doc ruletypes.SOPDocument) error {
	storable, err := ruletypes.FromDomainSOPDocument(orgID, doc)
	if err != nil {
		return err
	}
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewInsert().
			Model(storable).
			On("CONFLICT (org_id, sop_id, version) DO UPDATE").
			Set("contract_version = EXCLUDED.contract_version").
			Set("title = EXCLUDED.title").
			Set("updated_at = EXCLUDED.updated_at").
			Set("payload = EXCLUDED.payload").
			Exec(ctx)
		return err
	})
}

func (s *sopStore) Get(ctx context.Context, orgID, sopID, version string) (ruletypes.SOPDocument, error) {
	storable := new(ruletypes.StorableSOPDocument)
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(storable).
		Where("org_id = ?", orgID).
		Where("sop_id = ?", sopID).
		Where("version = ?", version).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.SOPDocument{}, ruletypes.ErrSOPDocumentNotFound
	}
	if err != nil {
		return ruletypes.SOPDocument{}, err
	}
	return storable.ToDomain()
}

// GetLatest returns the lexicographically highest version for (orgID, sopID).
// See SOPStore interface note: callers must use lexicographically sortable
// version strings.
func (s *sopStore) GetLatest(ctx context.Context, orgID, sopID string) (ruletypes.SOPDocument, error) {
	storable := new(ruletypes.StorableSOPDocument)
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(storable).
		Where("org_id = ?", orgID).
		Where("sop_id = ?", sopID).
		OrderExpr("version DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.SOPDocument{}, ruletypes.ErrSOPDocumentNotFound
	}
	if err != nil {
		return ruletypes.SOPDocument{}, err
	}
	return storable.ToDomain()
}

func (s *sopStore) List(ctx context.Context, orgID string) ([]ruletypes.SOPDocument, error) {
	var rows []ruletypes.StorableSOPDocument
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(&rows).
		Where("org_id = ?", orgID).
		OrderExpr("sop_id ASC, version ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	docs := make([]ruletypes.SOPDocument, 0, len(rows))
	for i := range rows {
		doc, err := rows[i].ToDomain()
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func (s *sopStore) Delete(ctx context.Context, orgID, sopID, version string) error {
	res, err := s.sqlstore.BunDBCtx(ctx).
		NewDelete().
		Model((*ruletypes.StorableSOPDocument)(nil)).
		Where("org_id = ?", orgID).
		Where("sop_id = ?", sopID).
		Where("version = ?", version).
		Exec(ctx)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ruletypes.ErrSOPDocumentNotFound
	}
	return nil
}

func (s *sopStore) UpsertRunbook(ctx context.Context, orgID, sopID, version string, rb ruletypes.Runbook) error {
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		storable := new(ruletypes.StorableSOPDocument)
		err := s.sqlstore.BunDBCtx(ctx).
			NewSelect().
			Model(storable).
			Where("org_id = ?", orgID).
			Where("sop_id = ?", sopID).
			Where("version = ?", version).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return ruletypes.ErrSOPDocumentNotFound
		}
		if err != nil {
			return err
		}
		doc, err := storable.ToDomain()
		if err != nil {
			return err
		}
		replaced := false
		for i := range doc.Runbooks {
			if doc.Runbooks[i].ID == rb.ID {
				doc.Runbooks[i] = rb
				replaced = true
				break
			}
		}
		if !replaced {
			doc.Runbooks = append(doc.Runbooks, rb)
		}
		updated, err := ruletypes.FromDomainSOPDocument(orgID, doc)
		if err != nil {
			return err
		}
		_, err = s.sqlstore.BunDBCtx(ctx).
			NewUpdate().
			Model(updated).
			Column("contract_version", "title", "updated_at", "payload").
			Where("org_id = ?", orgID).
			Where("sop_id = ?", sopID).
			Where("version = ?", version).
			Exec(ctx)
		return err
	})
}

func (s *sopStore) DeleteRunbook(ctx context.Context, orgID, sopID, version, runbookID string) error {
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		storable := new(ruletypes.StorableSOPDocument)
		err := s.sqlstore.BunDBCtx(ctx).
			NewSelect().
			Model(storable).
			Where("org_id = ?", orgID).
			Where("sop_id = ?", sopID).
			Where("version = ?", version).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return ruletypes.ErrSOPDocumentNotFound
		}
		if err != nil {
			return err
		}
		doc, err := storable.ToDomain()
		if err != nil {
			return err
		}
		found := false
		filtered := doc.Runbooks[:0]
		for _, r := range doc.Runbooks {
			if r.ID == runbookID {
				found = true
				continue
			}
			filtered = append(filtered, r)
		}
		if !found {
			return ruletypes.ErrSOPDocumentNotFound
		}
		doc.Runbooks = filtered
		updated, err := ruletypes.FromDomainSOPDocument(orgID, doc)
		if err != nil {
			return err
		}
		_, err = s.sqlstore.BunDBCtx(ctx).
			NewUpdate().
			Model(updated).
			Column("contract_version", "title", "updated_at", "payload").
			Where("org_id = ?", orgID).
			Where("sop_id = ?", sopID).
			Where("version = ?", version).
			Exec(ctx)
		return err
	})
}
