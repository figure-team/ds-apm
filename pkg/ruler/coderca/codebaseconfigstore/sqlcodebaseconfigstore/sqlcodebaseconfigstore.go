// Package sqlcodebaseconfigstore is the SQL-backed implementation of
// ruletypes.CodebaseRepoStore (CF-11 repo registration + source state).
package sqlcodebaseconfigstore

import (
	"context"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type codebaseRepoStore struct {
	sqlstore sqlstore.SQLStore
}

// New returns a CodebaseRepoStore backed by the given SQLStore. Migration 081
// must have run (table ds_codebase_repo).
func New(store sqlstore.SQLStore) ruletypes.CodebaseRepoStore {
	return &codebaseRepoStore{sqlstore: store}
}

func (s *codebaseRepoStore) Upsert(ctx context.Context, repo ruletypes.CodebaseRepo, encrypt func(string) (string, error)) error {
	// STUB — replaced in GREEN.
	return nil
}

func (s *codebaseRepoStore) Get(ctx context.Context, orgID, repoID string, decrypt func(string) (string, error)) (ruletypes.CodebaseRepo, error) {
	// STUB — replaced in GREEN.
	return ruletypes.CodebaseRepo{}, ruletypes.ErrCodebaseRepoNotFound
}

func (s *codebaseRepoStore) List(ctx context.Context, orgID string, decrypt func(string) (string, error)) ([]ruletypes.CodebaseRepo, error) {
	// STUB — replaced in GREEN.
	return nil, nil
}
