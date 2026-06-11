package sourcestate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// MirrorDir returns the cache path for repo's mirror clone. Path components are
// sanitized so a crafted org/repo id cannot escape baseDir.
func (g *ShellGitRunner) MirrorDir(repo ruletypes.CodebaseRepo) string {
	return filepath.Join(g.baseDir, sanitizePathComponent(repo.OrgID), sanitizePathComponent(repo.RepoID)+".git")
}

// Fetch ensures the cached mirror for repo exists and is up to date.
func (g *ShellGitRunner) Fetch(ctx context.Context, repo ruletypes.CodebaseRepo) error {
	dir := g.MirrorDir(repo)
	credEnv, cleanup, err := gitCredentialEnv(repo.Credential)
	if err != nil {
		return err
	}
	defer cleanup()

	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		if mkErr := os.MkdirAll(filepath.Dir(dir), 0o700); mkErr != nil {
			return fmt.Errorf("sourcestate: create mirror parent: %w", mkErr)
		}
		if _, err := g.run(ctx, "", credEnv, "clone", "--mirror", "--quiet", repo.GitURL, dir); err != nil {
			_ = os.RemoveAll(dir) // never leave a half-clone behind
			return fmt.Errorf("sourcestate: clone %s: %w", redactURL(repo.GitURL), err)
		}
		return nil
	}
	if _, err := g.run(ctx, dir, credEnv, "fetch", "--prune", "--quiet"); err != nil {
		return fmt.Errorf("sourcestate: fetch %s: %w", redactURL(repo.GitURL), err)
	}
	return nil
}

// ResolveHead returns the commit SHA at the tip of branch in the cached mirror.
func (g *ShellGitRunner) ResolveHead(ctx context.Context, repo ruletypes.CodebaseRepo, branch string) (string, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", fmt.Errorf("sourcestate: empty branch")
	}
	dir := g.MirrorDir(repo)
	// --verify --quiet exits nonzero (no message) when the ref is absent.
	out, err := g.run(ctx, dir, nil, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("sourcestate: resolve branch %q: %w", branch, err)
	}
	sha := strings.TrimSpace(out)
	if sha == "" {
		return "", fmt.Errorf("sourcestate: branch %q not found", branch)
	}
	return sha, nil
}

// run executes git with an optional working dir and extra env, capturing
// combined output (trimmed into the error on failure).
func (g *ShellGitRunner) run(ctx context.Context, dir string, extraEnv []string, args ...string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, g.binary, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%w: %s", err, truncateGit(string(out), 400))
	}
	return string(out), nil
}

// gitCredentialEnv builds the env additions (and cleanup) that deliver a git
// read credential WITHOUT writing the secret to disk: an askpass helper script
// (which contains no secret — it reads GIT_ASKPASS_PASS from the env) plus
// GIT_TERMINAL_PROMPT=0 to fail closed instead of prompting (design §9). An
// empty credential yields only GIT_TERMINAL_PROMPT=0 and a no-op cleanup.
func gitCredentialEnv(credential string) (env []string, cleanup func(), err error) {
	cleanup = func() {}
	credential = strings.TrimSpace(credential)
	if credential == "" {
		return []string{"GIT_TERMINAL_PROMPT=0"}, cleanup, nil
	}

	user, pass := splitCredential(credential)

	dir, mkErr := os.MkdirTemp("", "coderca-git-askpass-*")
	if mkErr != nil {
		return nil, cleanup, fmt.Errorf("sourcestate: askpass tempdir: %w", mkErr)
	}
	script := filepath.Join(dir, "askpass.sh")
	const body = "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"*[Uu]sername*) printf '%s' \"$GIT_ASKPASS_USER\" ;;\n" +
		"*) printf '%s' \"$GIT_ASKPASS_PASS\" ;;\n" +
		"esac\n"
	if wErr := os.WriteFile(script, []byte(body), 0o700); wErr != nil {
		_ = os.RemoveAll(dir)
		return nil, cleanup, fmt.Errorf("sourcestate: write askpass: %w", wErr)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	return []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=" + script,
		"GIT_ASKPASS_USER=" + user,
		"GIT_ASKPASS_PASS=" + pass,
	}, cleanup, nil
}

// splitCredential parses "user:token" or a bare token (username defaults to
// x-access-token, which common hosts accept for token auth).
func splitCredential(cred string) (user, pass string) {
	if i := strings.IndexByte(cred, ':'); i >= 0 {
		return cred[:i], cred[i+1:]
	}
	return "x-access-token", cred
}

// sanitizePathComponent neutralizes path separators and parent refs so an org
// or repo id cannot traverse out of the cache base dir.
func sanitizePathComponent(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "..", "_")
	if s == "" {
		return "_"
	}
	return s
}

// redactURL strips any user:pass@ userinfo from a URL for safe error logging.
func redactURL(u string) string {
	if i := strings.Index(u, "://"); i >= 0 {
		rest := u[i+3:]
		if at := strings.IndexByte(rest, '@'); at >= 0 {
			return u[:i+3] + "***@" + rest[at+1:]
		}
	}
	return u
}

func truncateGit(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

var _ GitRunner = (*ShellGitRunner)(nil)
