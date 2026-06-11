// Package sqlcodebaseservicemapstore is the SQL-backed implementation of
// ruletypes.CodebaseServiceMapStore (CF-11 service→repo mapping, design §8).
package sqlcodebaseservicemapstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type serviceMapStore struct {
	sqlstore sqlstore.SQLStore
}

// New returns a CodebaseServiceMapStore backed by the given SQLStore. Migration
// 081 must have run (table ds_codebase_service_map).
func New(store sqlstore.SQLStore) ruletypes.CodebaseServiceMapStore {
	return &serviceMapStore{sqlstore: store}
}

func (s *serviceMapStore) Upsert(ctx context.Context, m ruletypes.CodebaseServiceMap) error {
	storable, err := ruletypes.FromDomainCodebaseServiceMap(m)
	if err != nil {
		return err
	}
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewInsert().
			Model(storable).
			On("CONFLICT (org_id, service_name) DO UPDATE").
			Set("repo_id = EXCLUDED.repo_id").
			Set("subpath = EXCLUDED.subpath").
			Exec(ctx)
		return err
	})
}

func (s *serviceMapStore) Get(ctx context.Context, orgID, serviceName string) (ruletypes.CodebaseServiceMap, error) {
	storable := new(ruletypes.StorableCodebaseServiceMap)
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(storable).
		Where("org_id = ?", orgID).
		Where("service_name = ?", serviceName).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.CodebaseServiceMap{}, ruletypes.ErrCodebaseServiceMapNotFound
	}
	if err != nil {
		return ruletypes.CodebaseServiceMap{}, err
	}
	return storable.ToDomain(), nil
}

func (s *serviceMapStore) List(ctx context.Context, orgID string) ([]ruletypes.CodebaseServiceMap, error) {
	var storables []ruletypes.StorableCodebaseServiceMap
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(&storables).
		Where("org_id = ?", orgID).
		Order("service_name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ruletypes.CodebaseServiceMap, 0, len(storables))
	for i := range storables {
		out = append(out, storables[i].ToDomain())
	}
	return out, nil
}
