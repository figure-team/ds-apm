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
// DS-APM host — argvPrefix(샌드박스 래퍼)가 있으면 그 아래에서 (design §3.4).
type LocalTransport struct {
	timeout    time.Duration
	argvPrefix []string // resolveLocalSandbox가 만든 래퍼; nil이면 bash 직행
}

func newLocalTransport(timeout time.Duration, argvPrefix []string) *LocalTransport {
	if timeout <= 0 {
		timeout = DefaultExecTimeout
	}
	return &LocalTransport{timeout: timeout, argvPrefix: argvPrefix}
}

func (l *LocalTransport) Exec(ctx context.Context, script string) (string, int, bool, error) {
	runCtx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	argv := append(append([]string{}, l.argvPrefix...), "bash", "-c", script)
	cmd := exec.CommandContext(runCtx, argv[0], argv[1:]...)
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
