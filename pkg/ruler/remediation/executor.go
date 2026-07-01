// Package remediation executes pre-approved Runbook bash scripts under a hard
// timeout + process-group containment (design §5). It reuses clirunner's
// subprocess-containment approach but runs bash directly — no LLM, no checkout.
package remediation

import (
	"context"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/cliaudit"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
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

// RunMeta carries audit-tagging metadata for a single script run.
type RunMeta struct {
	Via         string // cliaudit seam; "" defaults to "remediation-exec"
	Source      string // "runbook" | "llm-generated"
	Fingerprint string
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

// RemoteTarget bundles a frozen target with its execute-time unsealed private
// key. A nil *RemoteTarget selects LocalTransport (design §3.4).
type RemoteTarget struct {
	Target        ruletypes.RemediationTarget
	PrivateKeyPEM string
}

// Run executes script via the appropriate transport under the executor timeout,
// logging a cliaudit.Record tagged with transport/target. target == nil → local.
func (e *Executor) Run(ctx context.Context, script string, target *RemoteTarget, meta RunMeta) ExecResult {
	var (
		tr        Transport
		transport = "local"
		targetTag string
		err       error
	)
	if target == nil {
		tr = newLocalTransport(e.timeout)
	} else {
		transport = "ssh"
		targetTag = target.Target.Host
		tr, err = newSSHTransport(target.Target, target.PrivateKeyPEM, e.timeout, 5*time.Second)
		if err != nil {
			// Key parse failure = fail-closed for this run (never falls back to local).
			e.logRecord(meta, transport, targetTag, 0, "failed", err)
			return ExecResult{ExitCode: -1, Output: "원격 실행 준비 실패: " + strings.TrimSpace(err.Error())}
		}
	}

	start := time.Now()
	raw, exitCode, timedOut, execErr := tr.Exec(ctx, script)
	out := truncate(raw, maxOutputCapture)

	res := ExecResult{Output: out, ExitCode: exitCode, TimedOut: timedOut}
	outcome := "ok"
	switch {
	case timedOut:
		outcome = "timeout"
		res.ExitCode = -1
	case execErr != nil:
		outcome = "failed"
		if res.Output == "" {
			res.Output = "실행 실패: " + strings.TrimSpace(execErr.Error())
		}
		res.ExitCode = -1
	case exitCode != 0:
		// non-zero exit is a script result, not an infra failure — record ok.
	}
	e.logRecordDur(meta, transport, targetTag, len(out), outcome, execErr, time.Since(start))
	return res
}

func (e *Executor) logRecord(meta RunMeta, transport, target string, outBytes int, outcome string, err error) {
	e.logRecordDur(meta, transport, target, outBytes, outcome, err, 0)
}

func (e *Executor) logRecordDur(meta RunMeta, transport, target string, outBytes int, outcome string, err error, dur time.Duration) {
	via := meta.Via
	if via == "" {
		via = "remediation-exec"
	}
	rec := cliaudit.Record{
		Via: via, Binary: "bash", Source: meta.Source, Fingerprint: meta.Fingerprint,
		DurationMS: dur.Milliseconds(), OutputBytes: outBytes, Outcome: outcome,
		Transport: transport, Target: target,
	}
	if err != nil {
		rec.Err = truncate(strings.TrimSpace(err.Error()), 256)
	}
	cliaudit.Default().Log(rec)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
