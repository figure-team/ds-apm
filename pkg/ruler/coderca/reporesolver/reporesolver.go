// Package reporesolver maps (org, service) → the repo CF-11 should analyze,
// composing the service-map store, the repo store, and the pure M2 resolver. It
// satisfies the engine's RepoResolver port; the concrete decryptor (secretbox)
// is injected at the integration seam.
package reporesolver

import (
	"context"
	"errors"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Resolver resolves a service to its registered repo (with decrypted read
// credential) for the worker.
type Resolver struct {
	maps    ruletypes.CodebaseServiceMapStore
	repos   ruletypes.CodebaseRepoStore
	decrypt func(string) (string, error)
}

// New builds a Resolver over the two stores and a credential decryptor.
func New(maps ruletypes.CodebaseServiceMapStore, repos ruletypes.CodebaseRepoStore, decrypt func(string) (string, error)) *Resolver {
	return &Resolver{maps: maps, repos: repos, decrypt: decrypt}
}

// ResolveRepo returns the repo to analyze for (orgID, service). ok=false (with a
// nil error) means "skip as no_repo_mapping": an unmapped service, a mapping
// pointing at a missing repo, or a disabled repo. A store error propagates.
//
func (r *Resolver) ResolveRepo(ctx context.Context, orgID, service string) (ruletypes.CodebaseRepo, string, bool, error) {
	mappings, err := r.maps.List(ctx, orgID)
	if err != nil {
		return ruletypes.CodebaseRepo{}, "", false, err
	}
	m, ok := coderca.ResolveServiceRepo(mappings, orgID, service)
	if !ok {
		return ruletypes.CodebaseRepo{}, "", false, nil
	}
	repo, err := r.repos.Get(ctx, orgID, m.RepoID, r.decrypt)
	if err != nil {
		if errors.Is(err, ruletypes.ErrCodebaseRepoNotFound) {
			return ruletypes.CodebaseRepo{}, "", false, nil // mapping points at a missing repo
		}
		return ruletypes.CodebaseRepo{}, "", false, err
	}
	if !repo.Enabled {
		return ruletypes.CodebaseRepo{}, "", false, nil // repo registration disabled
	}
	return repo, m.Subpath, true, nil
}
