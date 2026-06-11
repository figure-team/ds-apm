package sourcestate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// initOriginRepoWithSubdir creates a local origin with a file under sub/ and
// returns the path + HEAD sha.
func initOriginRepoWithSubdir(t *testing.T, sub string) (path, headSHA string) {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "--quiet", "-b", "main")
	if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sub, "svc.txt"), []byte("svc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", ".")
	git(t, dir, "commit", "--quiet", "-m", "init")
	return dir, git(t, dir, "rev-parse", "HEAD")
}

func TestManagerPrepareChecksOutBaseline(t *testing.T) {
	origin, sha := initOriginRepo(t)
	mgr := NewManager(NewShellGitRunner(t.TempDir()), t.TempDir())
	repo := testRepo(origin)

	checkout, baseline, cleanup, err := mgr.Prepare(context.Background(), repo, "")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer cleanup()

	if baseline != sha {
		t.Errorf("baseline = %q, want %q", baseline, sha)
	}
	if _, err := os.Stat(filepath.Join(checkout, "README.md")); err != nil {
		t.Errorf("checkout missing README.md: %v", err)
	}
	if got := git(t, checkout, "rev-parse", "HEAD"); got != sha {
		t.Errorf("checkout HEAD = %q, want baseline %q", got, sha)
	}

	cleanup()
	if _, err := os.Stat(checkout); !os.IsNotExist(err) {
		t.Errorf("checkout %q not removed by cleanup", checkout)
	}
}

func TestManagerPrepareNarrowsToSubpath(t *testing.T) {
	origin, _ := initOriginRepoWithSubdir(t, "services/orders")
	mgr := NewManager(NewShellGitRunner(t.TempDir()), t.TempDir())
	repo := testRepo(origin)

	checkout, _, cleanup, err := mgr.Prepare(context.Background(), repo, "services/orders")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer cleanup()

	if filepath.Base(checkout) != "orders" {
		t.Errorf("checkout %q not narrowed to subpath", checkout)
	}
	if _, err := os.Stat(filepath.Join(checkout, "svc.txt")); err != nil {
		t.Errorf("subpath checkout missing svc.txt: %v", err)
	}
}

func TestManagerPrepareFetchFailure(t *testing.T) {
	mgr := NewManager(NewShellGitRunner(t.TempDir()), t.TempDir())
	repo := testRepo("/no/such/repo/path")

	_, _, cleanup, err := mgr.Prepare(context.Background(), repo, "")
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Error("expected an error preparing an unfetchable repo")
	}
}
