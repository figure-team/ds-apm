package sourcestate

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// DefaultGitBinary is the git executable used when none is configured.
const DefaultGitBinary = "git"

// DefaultGitTimeout bounds a single git invocation (clone/fetch/rev-parse).
const DefaultGitTimeout = 2 * time.Minute

// ShellGitRunner implements GitRunner by shelling out to git against one cached
// mirror clone per repo under baseDir (design §8). It never writes a credential
// to disk in plaintext: a private repo's read token is delivered to git via an
// askpass helper that reads the secret from the (in-memory) child env (§9).
type ShellGitRunner struct {
	baseDir string
	binary  string
	timeout time.Duration
}

// NewShellGitRunner returns a runner caching mirror clones under baseDir.
func NewShellGitRunner(baseDir string) *ShellGitRunner {
	return &ShellGitRunner{baseDir: baseDir, binary: DefaultGitBinary, timeout: DefaultGitTimeout}
}

// Fetch ensures the cached mirror for repo exists and is up to date.
//
// D1 STUB: no-op → ResolveHead assertions fail (RED).
func (g *ShellGitRunner) Fetch(ctx context.Context, repo ruletypes.CodebaseRepo) error {
	return nil
}

// ResolveHead returns the commit SHA at the tip of branch in the cached mirror.
//
// D1 STUB: returns empty → assertions fail (RED).
func (g *ShellGitRunner) ResolveHead(ctx context.Context, repo ruletypes.CodebaseRepo, branch string) (string, error) {
	return "", nil
}

// gitCredentialEnv builds the env additions (and cleanup) that deliver a git
// read credential WITHOUT writing the secret to disk: an askpass helper script
// (which contains no secret — it reads GIT_ASKPASS_PASS from the env) plus
// GIT_TERMINAL_PROMPT=0 to fail closed instead of prompting (design §9). An
// empty credential yields only GIT_TERMINAL_PROMPT=0 and a no-op cleanup.
//
// D1 STUB: returns nil → credential-env assertions fail (RED).
func gitCredentialEnv(credential string) (env []string, cleanup func(), err error) {
	return nil, func() {}, nil
}

var _ GitRunner = (*ShellGitRunner)(nil)
