// Package reporesolver maps (org, service) → the repo CF-11 should analyze,
// composing the service-map store, the repo store, and the pure M2 resolver. It
// satisfies the engine's RepoResolver port; the concrete decryptor (secretbox)
// is injected at the integration seam.
package reporesolver

import (
	"context"

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
// E3 STUB: always ok=true with a zero repo → every assertion fails (RED).
func (r *Resolver) ResolveRepo(ctx context.Context, orgID, service string) (ruletypes.CodebaseRepo, string, bool, error) {
	return ruletypes.CodebaseRepo{}, "", true, nil
}
