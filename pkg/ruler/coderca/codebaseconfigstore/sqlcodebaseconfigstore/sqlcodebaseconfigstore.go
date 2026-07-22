// Package sqlcodebaseconfigstore is the SQL-backed implementation of
// ruletypes.CodebaseRepoStore (CF-11 repo registration + source state).
package sqlcodebaseconfigstore

import (
	"context"
	"database/sql"
	"errors"

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
	storable, err := ruletypes.FromDomainCodebaseRepo(repo, encrypt)
	if err != nil {
		return err
	}
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewInsert().
			Model(storable).
			On("CONFLICT (org_id, repo_id) DO UPDATE").
			Set("git_url = EXCLUDED.git_url").
			Set("default_branch = EXCLUDED.default_branch").
			Set("credential_ciphertext = EXCLUDED.credential_ciphertext").
			Set("enabled = EXCLUDED.enabled").
			Set("artifact_path = EXCLUDED.artifact_path").
			Set("branch_name = EXCLUDED.branch_name").
			Set("fetched = EXCLUDED.fetched").
			Set("baseline_commit = EXCLUDED.baseline_commit").
			Set("last_sync_at = EXCLUDED.last_sync_at").
			Set("last_sync_status = EXCLUDED.last_sync_status").
			Exec(ctx)
		return err
	})
}

func (s *codebaseRepoStore) Get(ctx context.Context, orgID, repoID string, decrypt func(string) (string, error)) (ruletypes.CodebaseRepo, error) {
	storable := new(ruletypes.StorableCodebaseRepo)
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(storable).
		Where("org_id = ?", orgID).
		Where("repo_id = ?", repoID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.CodebaseRepo{}, ruletypes.ErrCodebaseRepoNotFound
	}
	if err != nil {
		return ruletypes.CodebaseRepo{}, err
	}
	return storable.ToDomain(decrypt)
}

func (s *codebaseRepoStore) Delete(ctx context.Context, orgID, repoID string) error {
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewDelete().
			TableExpr("ds_codebase_repo").
			Where("org_id = ?", orgID).
			Where("repo_id = ?", repoID).
			Exec(ctx)
		return err
	})
}

func (s *codebaseRepoStore) List(ctx context.Context, orgID string, decrypt func(string) (string, error)) ([]ruletypes.CodebaseRepo, error) {
	var storables []ruletypes.StorableCodebaseRepo
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(&storables).
		Where("org_id = ?", orgID).
		Order("repo_id ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ruletypes.CodebaseRepo, 0, len(storables))
	for i := range storables {
		repo, err := storables[i].ToDomain(decrypt)
		if err != nil {
			return nil, err
		}
		out = append(out, repo)
	}
	return out, nil
}
