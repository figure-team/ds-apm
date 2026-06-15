// Package sqltemplatestore persists the per-org incident-report template (the
// managed 양식) in the ds_incident_report_template table (migration 084).
package sqltemplatestore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
)

type storableTemplate struct {
	bun.BaseModel `bun:"table:ds_incident_report_template"`

	OrgID     string `bun:"org_id,pk,notnull,type:text"`
	Template  string `bun:"template,notnull,type:text"`
	UpdatedAt string `bun:"updated_at,notnull,type:text"`
}

// Store reads/writes the per-org incident-report template.
type Store struct {
	sqlstore sqlstore.SQLStore
}

// New returns a Store backed by the given SQLStore. Migration 084 must have run.
func New(store sqlstore.SQLStore) *Store {
	return &Store{sqlstore: store}
}

// Get returns the org's template and whether one is set. A missing row is not an
// error — the caller falls back to the default template.
func (s *Store) Get(ctx context.Context, orgID string) (string, bool, error) {
	st := new(storableTemplate)
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(st).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return st.Template, true, nil
}

// Upsert sets the org's template (empty template clears to default behavior at
// read time, but is stored verbatim).
func (s *Store) Upsert(ctx context.Context, orgID, template string) error {
	st := &storableTemplate{
		OrgID:     orgID,
		Template:  template,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewInsert().
			Model(st).
			On("CONFLICT (org_id) DO UPDATE").
			Set("template = EXCLUDED.template").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		return err
	})
}
