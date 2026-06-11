// Package sqlcodebaseservicemapstore is the SQL-backed implementation of
// ruletypes.CodebaseServiceMapStore (CF-11 service→repo mapping, design §8).
package sqlcodebaseservicemapstore

import (
	"context"

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

// E2 STUB: no write → Get/List assertions fail (RED).
func (s *serviceMapStore) Upsert(ctx context.Context, m ruletypes.CodebaseServiceMap) error {
	return nil
}

// E2 STUB: returns the zero mapping, no error → assertions fail (RED).
func (s *serviceMapStore) Get(ctx context.Context, orgID, serviceName string) (ruletypes.CodebaseServiceMap, error) {
	return ruletypes.CodebaseServiceMap{}, nil
}

// E2 STUB: returns nothing → List assertion fails (RED).
func (s *serviceMapStore) List(ctx context.Context, orgID string) ([]ruletypes.CodebaseServiceMap, error) {
	return nil, nil
}
