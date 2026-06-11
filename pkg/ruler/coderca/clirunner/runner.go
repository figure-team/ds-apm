package clirunner

import (
	"context"
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
// A3 STUB: returns failed without executing → status/result/err assertions fail
// (RED).
func (r *Runner) Run(ctx context.Context, s Spec) (coderca.RCAResult, coderca.RunStatus, error) {
	return coderca.RCAResult{}, coderca.RunStatusFailed, nil
}
