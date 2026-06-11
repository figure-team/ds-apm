package sourcestate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Manager realizes the engine's source-preparer port (design §8): it keeps a
// mirror clone current via a ShellGitRunner, pins the default-branch HEAD as the
// baseline, and materializes a disposable per-run worktree checkout at that
// commit. The returned cleanup removes the worktree (so a flood of runs cannot
// accrete disk).
type Manager struct {
	git          *ShellGitRunner
	checkoutBase string
}

// NewManager builds a Manager that creates per-run checkouts under checkoutBase
// using the given git runner.
func NewManager(git *ShellGitRunner, checkoutBase string) *Manager {
	return &Manager{git: git, checkoutBase: checkoutBase}
}

// Prepare fetches the repo, resolves the default-branch baseline, and creates a
// disposable worktree checkout at that commit (narrowed to subpath when set).
// cleanup removes the worktree and is always safe to defer.
//
// D2 STUB: returns empty → checkout/baseline assertions fail (RED).
func (m *Manager) Prepare(ctx context.Context, repo ruletypes.CodebaseRepo, subpath string) (checkoutDir, baseline string, cleanup func(), err error) {
	return "", "", func() {}, nil
}

// uniqueCheckoutDir returns a fresh per-run checkout path under checkoutBase.
func (m *Manager) uniqueCheckoutDir(repo ruletypes.CodebaseRepo) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return filepath.Join(
		m.checkoutBase,
		sanitizePathComponent(repo.OrgID),
		sanitizePathComponent(repo.RepoID),
		hex.EncodeToString(b[:]),
	), nil
}

// AddWorktree creates a detached worktree of repo's mirror at commit, in dest.
func (g *ShellGitRunner) AddWorktree(ctx context.Context, repo ruletypes.CodebaseRepo, commit, dest string) error {
	if mkErr := os.MkdirAll(filepath.Dir(dest), 0o700); mkErr != nil {
		return fmt.Errorf("sourcestate: create checkout parent: %w", mkErr)
	}
	if _, err := g.run(ctx, g.MirrorDir(repo), nil, "worktree", "add", "--detach", "--quiet", dest, commit); err != nil {
		return fmt.Errorf("sourcestate: worktree add: %w", err)
	}
	return nil
}

// RemoveWorktree detaches a worktree from repo's mirror registry (best effort).
func (g *ShellGitRunner) RemoveWorktree(ctx context.Context, repo ruletypes.CodebaseRepo, dest string) error {
	_, err := g.run(ctx, g.MirrorDir(repo), nil, "worktree", "remove", "--force", dest)
	return err
}

// cleanSubpath neutralizes traversal so subpath cannot escape the checkout root.
func cleanSubpath(sub string) string {
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return ""
	}
	return filepath.Clean("/" + sub)[1:]
}
