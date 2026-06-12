package clirunner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
)

const (
	// DefaultTimeout is the per-run wall-clock ceiling (design §6.5).
	DefaultTimeout = 5 * time.Minute
	// DefaultWaitDelay bounds how long after ctx-cancel we wait before force
	// reaping the subprocess (closing its pipes).
	DefaultWaitDelay = 2 * time.Second
)

// Runner executes an agent CLI under hard timeout + subprocess containment and
// parses its output into an RCAResult.
type Runner struct {
	timeout   time.Duration
	waitDelay time.Duration
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithTimeout sets the per-run wall-clock ceiling.
func WithTimeout(d time.Duration) RunnerOption {
	return func(r *Runner) {
		if d > 0 {
			r.timeout = d
		}
	}
}

// WithWaitDelay sets the post-cancel subprocess reap delay.
func WithWaitDelay(d time.Duration) RunnerOption {
	return func(r *Runner) {
		if d > 0 {
			r.waitDelay = d
		}
	}
}

// NewRunner builds a Runner with the given options (defaults otherwise).
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{timeout: DefaultTimeout, waitDelay: DefaultWaitDelay}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Run drives the agent CLI in the checkout and returns the parsed RCAResult and
// a terminal RunStatus (done | unparseable | timeout | failed). The whole
// subprocess tree is contained per §6.5: own process group + parent-death
// signal, group-killed on ctx/timeout. Raw stdout is always retained on the
// result for audit.
//
func (r *Runner) Run(ctx context.Context, s Spec) (coderca.RCAResult, coderca.RunStatus, error) {
	args, err := BuildArgs(s)
	if err != nil {
		return coderca.RCAResult{}, coderca.RunStatusFailed, err
	}
	binary := s.Binary
	if binary == "" {
		binary = DefaultBinary(s.Agent)
	}

	env, cleanup, err := BuildEnv(s, os.Environ())
	if err != nil {
		return coderca.RCAResult{}, coderca.RunStatusFailed, err
	}
	defer cleanup()
	if len(s.ExtraEnv) > 0 {
		env = append(env, s.ExtraEnv...)
	}

	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, binary, args...)
	cmd.Dir = s.Checkout
	cmd.Env = env
	cmd.WaitDelay = r.waitDelay
	configureSubprocess(cmd)
	// Override CommandContext's lead-pid kill with a whole-group kill (§6.5).
	cmd.Cancel = func() error { return killProcessTree(cmd) }

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	raw := stdout.String()

	if runCtx.Err() == context.DeadlineExceeded {
		return coderca.RCAResult{Raw: raw}, coderca.RunStatusTimeout,
			fmt.Errorf("clirunner: run exceeded %s: %w", r.timeout, runCtx.Err())
	}
	if runErr != nil {
		return coderca.RCAResult{Raw: raw}, coderca.RunStatusFailed,
			fmt.Errorf("clirunner: %s: %w (stderr: %s)", binary, runErr, truncate(stderr.String(), 512))
	}

	out := raw
	if s.Agent == AgentCodex {
		out = reconstructCodexText(raw)
	}
	res, status := coderca.ParseRCAResult(out)
	res.Raw = raw // retain full stdout for audit, regardless of normalization
	return res, status, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
