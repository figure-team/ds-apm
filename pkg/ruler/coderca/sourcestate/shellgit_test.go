package sourcestate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// initOriginRepo creates a local origin repo with one commit on main and
// returns its path + the HEAD sha.
func initOriginRepo(t *testing.T) (path, headSHA string) {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "--quiet", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", ".")
	git(t, dir, "commit", "--quiet", "-m", "init")
	return dir, git(t, dir, "rev-parse", "HEAD")
}

func testRepo(origin string) ruletypes.CodebaseRepo {
	return ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1", GitURL: origin, DefaultBranch: "main"}
}

func TestShellGitRunnerFetchAndResolve(t *testing.T) {
	origin, sha := initOriginRepo(t)
	gr := NewShellGitRunner(t.TempDir())
	repo := testRepo(origin)

	if err := gr.Fetch(context.Background(), repo); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got, err := gr.ResolveHead(context.Background(), repo, "main")
	if err != nil {
		t.Fatalf("ResolveHead: %v", err)
	}
	if got != sha {
		t.Errorf("ResolveHead = %q, want %q", got, sha)
	}
	// Re-fetch of an existing mirror must also succeed.
	if err := gr.Fetch(context.Background(), repo); err != nil {
		t.Errorf("re-fetch: %v", err)
	}
}

func TestShellGitRunnerFetchSeesNewCommits(t *testing.T) {
	origin, _ := initOriginRepo(t)
	gr := NewShellGitRunner(t.TempDir())
	repo := testRepo(origin)

	if err := gr.Fetch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	// New commit upstream.
	if err := os.WriteFile(filepath.Join(origin, "next.txt"), []byte("more\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, origin, "add", ".")
	git(t, origin, "commit", "--quiet", "-m", "second")
	want := git(t, origin, "rev-parse", "HEAD")

	if err := gr.Fetch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	got, err := gr.ResolveHead(context.Background(), repo, "main")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("after re-fetch ResolveHead = %q, want %q (new commit)", got, want)
	}
}

func TestShellGitRunnerResolveUnknownBranch(t *testing.T) {
	origin, _ := initOriginRepo(t)
	gr := NewShellGitRunner(t.TempDir())
	repo := testRepo(origin)
	if err := gr.Fetch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if _, err := gr.ResolveHead(context.Background(), repo, "does-not-exist"); err == nil {
		t.Error("expected an error resolving an unknown branch")
	}
}

func envVal(env []string, key string) (string, bool) {
	p := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, p) {
			return e[len(p):], true
		}
	}
	return "", false
}

func TestGitCredentialEnvKeepsSecretOffDisk(t *testing.T) {
	const secret = "super-secret-token-9f8e7d"
	env, cleanup, err := gitCredentialEnv(secret)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := envVal(env, "GIT_TERMINAL_PROMPT"); !ok {
		t.Error("GIT_TERMINAL_PROMPT must be set (fail closed, no prompt)")
	}
	pass, ok := envVal(env, "GIT_ASKPASS_PASS")
	if !ok || pass != secret {
		t.Errorf("secret must be delivered via env: got %q ok=%v", pass, ok)
	}
	askpass, ok := envVal(env, "GIT_ASKPASS")
	if !ok {
		t.Fatal("GIT_ASKPASS script path must be set for a credential")
	}
	b, err := os.ReadFile(askpass)
	if err != nil {
		t.Fatalf("read askpass script: %v", err)
	}
	if strings.Contains(string(b), secret) {
		t.Error("askpass script must not contain the secret in plaintext on disk")
	}
}

func TestGitCredentialEnvEmptyCredential(t *testing.T) {
	env, cleanup, err := gitCredentialEnv("")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := envVal(env, "GIT_ASKPASS"); ok {
		t.Error("no askpass script expected for an empty credential")
	}
	if val, ok := envVal(env, "GIT_TERMINAL_PROMPT"); !ok || val != "0" {
		t.Error("GIT_TERMINAL_PROMPT=0 must still be set to fail closed")
	}
}
