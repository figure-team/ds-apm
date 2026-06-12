package sourcestate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// ErrNoDefaultBranch is the sync failure when a repo has no configured branch
// to track — there is nothing to resolve a baseline against.
var ErrNoDefaultBranch = errors.New("no default branch configured")

// GitRunner abstracts the git operations a sync needs against a repo's cached
// bare clone. Implementations shell out to git (design §12); tests fake it.
type GitRunner interface {
	// Fetch ensures the cached clone for repo exists and is up to date.
	Fetch(ctx context.Context, repo ruletypes.CodebaseRepo) error
	// ResolveHead returns the commit SHA at the tip of branch in the clone.
	ResolveHead(ctx context.Context, repo ruletypes.CodebaseRepo, branch string) (string, error)
}

// Syncer wires a GitRunner to the pure transition (Apply). The clock is
// injected so the transition stays deterministic under test.
type Syncer struct {
	git GitRunner
	now func() time.Time
}

// NewSyncer constructs a Syncer over the given git runner and clock.
func NewSyncer(git GitRunner, now func() time.Time) *Syncer {
	return &Syncer{git: git, now: now}
}

// Sync fetches the repo's cached clone, resolves the tracked branch HEAD as the
// baseline, and returns the next source state. Git failures are folded into the
// state (LastSyncStatus) rather than returned, so a transient error is visible
// in the UI without losing the last-good baseline (design §8).
//
func (s *Syncer) Sync(ctx context.Context, repo ruletypes.CodebaseRepo) SourceState {
	prev := StateOf(repo)
	branch := strings.TrimSpace(repo.DefaultBranch)
	if branch == "" {
		return Apply(prev, SyncFacts{Branch: branch, Err: ErrNoDefaultBranch}, s.now())
	}
	if err := s.git.Fetch(ctx, repo); err != nil {
		return Apply(prev, SyncFacts{Branch: branch, Err: fmt.Errorf("fetch: %w", err)}, s.now())
	}
	head, err := s.git.ResolveHead(ctx, repo, branch)
	if err != nil {
		return Apply(prev, SyncFacts{Branch: branch, Err: fmt.Errorf("resolve head: %w", err)}, s.now())
	}
	return Apply(prev, SyncFacts{Branch: branch, HeadCommit: head}, s.now())
}
