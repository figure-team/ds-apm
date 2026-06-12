package clirunner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
)

// TestMain doubles as a fake CLI / grandchild when the role env is set, so the
// Runner can exec this same test binary instead of a real claude/codex.
func TestMain(m *testing.M) {
	switch os.Getenv("CODERCA_FAKE_ROLE") {
	case "cli":
		fakeCLI()
	case "grandchild":
		fakeGrandchild()
	default:
		os.Exit(m.Run())
	}
}

func fakeCLI() {
	if os.Getenv("CODERCA_FAKE_SPAWN_CHILD") == "1" {
		c := exec.Command(os.Args[0])
		c.Env = append(os.Environ(), "CODERCA_FAKE_ROLE=grandchild")
		_ = c.Start() // inherits our process group; do not wait
	}
	if file := os.Getenv("CODERCA_FAKE_STDOUT_FILE"); file != "" {
		if b, err := os.ReadFile(file); err == nil {
			_, _ = os.Stdout.Write(b)
		}
	}
	if ms := os.Getenv("CODERCA_FAKE_SLEEP_MS"); ms != "" {
		if d, err := strconv.Atoi(ms); err == nil {
			time.Sleep(time.Duration(d) * time.Millisecond)
		}
	}
	if code := os.Getenv("CODERCA_FAKE_EXIT"); code != "" {
		n, _ := strconv.Atoi(code)
		os.Exit(n)
	}
	os.Exit(0)
}

func fakeGrandchild() {
	if f := os.Getenv("CODERCA_FAKE_GRANDCHILD_PIDFILE"); f != "" {
		_ = os.WriteFile(f, []byte(strconv.Itoa(os.Getpid())), 0o600)
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stdout.txt")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func claudeSpec(t *testing.T, env ...string) Spec {
	return Spec{
		Agent:        AgentClaude,
		Binary:       os.Args[0],
		Model:        "fake-model",
		Checkout:     t.TempDir(),
		MaxBudgetUSD: "1",
		ExtraEnv:     append([]string{"CODERCA_FAKE_ROLE=cli"}, env...),
	}
}

func TestRunnerHappyPathParsesOutput(t *testing.T) {
	out := "Here is my analysis:\n\n```json\n{\"baseline_commit\":\"a1b2c3\",\"root_cause\":\"pool exhausted\",\"proposed_fix\":\"close rows\",\"confidence\":\"medium\",\"limitations\":\"static only\"}\n```\n"
	spec := claudeSpec(t, "CODERCA_FAKE_STDOUT_FILE="+writeTemp(t, out))

	res, status, err := NewRunner(WithTimeout(10 * time.Second)).Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if status != coderca.RunStatusDone {
		t.Fatalf("status = %q, want done", status)
	}
	if res.BaselineCommit != "a1b2c3" {
		t.Errorf("BaselineCommit = %q, want a1b2c3", res.BaselineCommit)
	}
	if res.RootCause != "pool exhausted" {
		t.Errorf("RootCause = %q", res.RootCause)
	}
	if res.Confidence != "medium" {
		t.Errorf("Confidence = %q", res.Confidence)
	}
	if res.Raw != out {
		t.Errorf("Raw not retained verbatim")
	}
}

func TestRunnerTimeoutKillsAndReports(t *testing.T) {
	spec := claudeSpec(t, "CODERCA_FAKE_SLEEP_MS=30000")

	start := time.Now()
	res, status, err := NewRunner(WithTimeout(300*time.Millisecond), WithWaitDelay(500*time.Millisecond)).
		Run(context.Background(), spec)
	elapsed := time.Since(start)

	if status != coderca.RunStatusTimeout {
		t.Fatalf("status = %q, want timeout", status)
	}
	if err == nil {
		t.Error("expected a timeout error")
	}
	if elapsed > 5*time.Second {
		t.Errorf("Run took %s; subprocess was not killed promptly", elapsed)
	}
	_ = res
}

func TestRunnerNonZeroExitIsFailed(t *testing.T) {
	spec := claudeSpec(t, "CODERCA_FAKE_EXIT=3")

	_, status, err := NewRunner(WithTimeout(10 * time.Second)).Run(context.Background(), spec)
	if status != coderca.RunStatusFailed {
		t.Fatalf("status = %q, want failed", status)
	}
	if err == nil {
		t.Error("expected an error for non-zero exit")
	}
}

func TestRunnerBuildErrorIsFailedNoExec(t *testing.T) {
	// Missing checkout → BuildArgs rejects before any subprocess.
	spec := Spec{Agent: AgentClaude, Binary: os.Args[0], Model: "m", MaxBudgetUSD: "1"}
	_, status, err := NewRunner().Run(context.Background(), spec)
	if status != coderca.RunStatusFailed {
		t.Fatalf("status = %q, want failed", status)
	}
	if err == nil {
		t.Fatal("expected ErrNoCheckout")
	}
}

func TestRunnerCodexJSONLOutput(t *testing.T) {
	block := "Done.\n\n```json\n{\"baseline_commit\":\"c0ffee\",\"root_cause\":\"unbounded retry loop\",\"confidence\":\"low\"}\n```"
	event := map[string]any{
		"type": "item.completed",
		"item": map[string]any{"type": "agent_message", "text": block},
	}
	line, _ := json.Marshal(event)
	stdoutFile := writeTemp(t, string(line)+"\n")

	spec := Spec{
		Agent:    AgentCodex,
		Binary:   os.Args[0],
		Model:    "fake-model",
		Checkout: t.TempDir(),
		ExtraEnv: []string{"CODERCA_FAKE_ROLE=cli", "CODERCA_FAKE_STDOUT_FILE=" + stdoutFile},
	}
	res, status, err := NewRunner(WithTimeout(10 * time.Second)).Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if status != coderca.RunStatusDone {
		t.Fatalf("status = %q, want done", status)
	}
	if res.BaselineCommit != "c0ffee" {
		t.Errorf("BaselineCommit = %q, want c0ffee", res.BaselineCommit)
	}
	if res.RootCause != "unbounded retry loop" {
		t.Errorf("RootCause = %q", res.RootCause)
	}
	// Raw retains the full JSONL stdout for audit (not the reconstructed text).
	if !strings.Contains(res.Raw, "agent_message") {
		t.Errorf("Raw not retained verbatim: %q", res.Raw)
	}
}

func TestRunnerTimeoutKillsProcessGroup(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process-group containment is Linux-specific (§6.5)")
	}
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	spec := claudeSpec(t,
		"CODERCA_FAKE_SPAWN_CHILD=1",
		"CODERCA_FAKE_GRANDCHILD_PIDFILE="+pidFile,
		"CODERCA_FAKE_SLEEP_MS=30000",
	)

	_, status, _ := NewRunner(WithTimeout(1500*time.Millisecond), WithWaitDelay(500*time.Millisecond)).
		Run(context.Background(), spec)
	if status != coderca.RunStatusTimeout {
		t.Fatalf("status = %q, want timeout", status)
	}

	pid := waitForPidFile(t, pidFile)
	// The grandchild shared the killed process group; it must die.
	if !waitUntilDead(pid, 5*time.Second) {
		t.Errorf("grandchild pid %d still alive: process group not killed", pid)
	}
}

func waitForPidFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			if pid, convErr := strconv.Atoi(string(b)); convErr == nil {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("grandchild never wrote its pid to %s", path)
	return 0
}

func waitUntilDead(pid int, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil { // ESRCH → gone
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return syscall.Kill(pid, 0) != nil
}
