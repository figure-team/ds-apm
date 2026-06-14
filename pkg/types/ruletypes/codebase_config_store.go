package ruletypes

import (
	"context"
	"errors"
)

var ErrCodebaseRepoNotFound = errors.New("codebase repo not found")

// CodebaseRepoStore persists CF-11 repo registrations + source state. All
// methods are org-scoped; the credential is encrypted/decrypted via the
// provided closures (secretbox), so the store never sees a master key.
type CodebaseRepoStore interface {
	Upsert(ctx context.Context, repo CodebaseRepo, encrypt func(string) (string, error)) error
	Get(ctx context.Context, orgID, repoID string, decrypt func(string) (string, error)) (CodebaseRepo, error)
	List(ctx context.Context, orgID string, decrypt func(string) (string, error)) ([]CodebaseRepo, error)
	Delete(ctx context.Context, orgID, repoID string) error
}
