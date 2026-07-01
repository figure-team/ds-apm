package remediation

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Transport executes a script and returns raw combined output, an exit code, and
// whether it timed out. LocalTransport runs bash on this host; SSHTransport runs
// it on a remote target (design §3.4).
type Transport interface {
	Exec(ctx context.Context, script string) (raw string, exitCode int, timedOut bool, err error)
}

// LocalTransport runs `bash -c <script>` under process-group containment on the
// DS-APM host — the pre-remote behaviour, unchanged (design §3.4).
type LocalTransport struct {
	timeout time.Duration
}

func newLocalTransport(timeout time.Duration) *LocalTransport {
	if timeout <= 0 {
		timeout = DefaultExecTimeout
	}
	return &LocalTransport{timeout: timeout}
}

func (l *LocalTransport) Exec(ctx context.Context, script string) (string, int, bool, error) {
	runCtx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "bash", "-c", script)
	cmd.WaitDelay = defaultWaitDelay
	configureSubprocess(cmd)
	cmd.Cancel = func() error { return killProcessTree(cmd) }

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()
	out := buf.String()

	if runCtx.Err() == context.DeadlineExceeded {
		return out, -1, true, nil
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return out, exitErr.ExitCode(), false, nil
		}
		if out == "" {
			out = "실행 실패: " + strings.TrimSpace(runErr.Error())
		}
		return out, -1, false, runErr
	}
	return out, 0, false, nil
}
