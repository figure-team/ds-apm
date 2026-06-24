// Package remediation executes pre-approved Runbook bash scripts under a hard
// timeout + process-group containment (design §5). It reuses clirunner's
// subprocess-containment approach but runs bash directly — no LLM, no checkout.
package remediation

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/cliaudit"
)

const (
	// DefaultExecTimeout is the per-run wall-clock ceiling.
	DefaultExecTimeout = 5 * time.Minute
	defaultWaitDelay   = 2 * time.Second
	maxOutputCapture   = 64 * 1024
)

// ExecResult holds the outcome of a single bash script execution.
type ExecResult struct {
	ExitCode int
	Output   string // combined stdout+stderr, truncated to maxOutputCapture
	TimedOut bool
}

// Executor runs bash scripts under a hard timeout with process-group containment.
type Executor struct {
	timeout time.Duration
}

// NewExecutor returns an Executor with the given hard timeout.
// A zero or negative timeout falls back to DefaultExecTimeout.
func NewExecutor(timeout time.Duration) *Executor {
	if timeout <= 0 {
		timeout = DefaultExecTimeout
	}
	return &Executor{timeout: timeout}
}

// Run executes script via `bash -c` under timeout + group containment. The whole
// process tree is killed on timeout. Output (stdout+stderr combined) is captured
// and truncated to maxOutputCapture bytes. Always logs a cliaudit.Record with
// Via "remediation-exec".
func (e *Executor) Run(ctx context.Context, script string) ExecResult {
	runCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "bash", "-c", script)
	cmd.WaitDelay = defaultWaitDelay
	configureSubprocess(cmd)
	// Override CommandContext's lead-pid kill with a whole-group kill.
	cmd.Cancel = func() error { return killProcessTree(cmd) }

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	start := time.Now()
	runErr := cmd.Run()
	// OPERATOR CONTRACT: Approved runbook scripts must not print secrets to
	// stdout/stderr — the captured snippet below is stored unredacted in
	// ds_remediation_execution.output_snippet and is visible to Viewers via
	// the GET /remediation API. Secret masking is a future extension point.
	out := truncate(buf.String(), maxOutputCapture)

	res := ExecResult{Output: out}
	rec := cliaudit.Record{
		Via:         "remediation-exec",
		Binary:      "bash",
		DurationMS:  time.Since(start).Milliseconds(),
		OutputBytes: len(out),
		Outcome:     "ok",
	}

	switch {
	case runCtx.Err() == context.DeadlineExceeded:
		res.TimedOut = true
		res.ExitCode = -1
		rec.Outcome = "timeout"
	case runErr != nil:
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
		rec.Outcome = "failed"
		rec.Err = truncate(strings.TrimSpace(runErr.Error()), 256)
	default:
		res.ExitCode = 0
	}
	cliaudit.Default().Log(rec)
	return res
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
