package remediation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// remoteKillGraceSeconds는 원격 자체-kill을 클라이언트 워치독보다 살짝 늦게
// 잡는 여유다: 결과 판정은 항상 클라이언트 타임아웃이 먼저, 원격 timeout은
// 프로세스 잔존 방지용 최후 보루.
const remoteKillGraceSeconds = 5

// remoteExecCommand는 stdin 주입 스크립트를 실행할 원격 exec 명령을 만든다.
// 타겟에 coreutils/busybox `timeout`이 있으면 (execTimeout+grace)초 후 타겟
// 스스로 실행 중인 bash를 SIGKILL한다 — 클라이언트의 세션 close는 원격 kill을
// 보장하지 않는다(design §3.5 B3)는 갭을 닫는다. (자식 프로세스 트리 전체
// 종료는 timeout 구현에 따라 다르므로 lead bash 종료만 보장한다.) `timeout`이
// 없는 타겟은 기존 `bash -s` 그대로 폴백(하위호환).
func remoteExecCommand(execTimeout time.Duration) string {
	if execTimeout <= 0 {
		execTimeout = DefaultExecTimeout
	}
	secs := int(execTimeout.Seconds()) + remoteKillGraceSeconds
	return fmt.Sprintf(
		"sh -c 'if command -v timeout >/dev/null 2>&1; then exec timeout -s KILL %d bash -s; else exec bash -s; fi'",
		secs,
	)
}

// SSHTransport runs a script on a remote target over SSH using a frozen host-key
// fingerprint for verification and an in-memory private key (design §3.5).
// InsecureIgnoreHostKey is never used.
type SSHTransport struct {
	target      ruletypes.RemediationTarget
	signer      ssh.Signer
	execTimeout time.Duration
	dialTimeout time.Duration
}

func newSSHTransport(target ruletypes.RemediationTarget, privateKeyPEM string, execTimeout, dialTimeout time.Duration) (*SSHTransport, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("ssh: parse private key: %w", err)
	}
	if execTimeout <= 0 {
		execTimeout = DefaultExecTimeout
	}
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	return &SSHTransport{target: target, signer: signer, execTimeout: execTimeout, dialTimeout: dialTimeout}, nil
}

// pinnedHostKey returns a HostKeyCallback that accepts only the frozen fingerprint.
func (s *SSHTransport) pinnedHostKey() ssh.HostKeyCallback {
	want := strings.TrimSpace(s.target.HostKeyFingerprint)
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if got := ssh.FingerprintSHA256(key); got != want {
			return fmt.Errorf("ssh: host key mismatch: got %s want %s", got, want)
		}
		return nil
	}
}

func (s *SSHTransport) Exec(ctx context.Context, script string) (string, int, bool, error) {
	runCtx, cancel := context.WithTimeout(ctx, s.execTimeout)
	defer cancel()

	cfg := &ssh.ClientConfig{
		User:            s.target.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(s.signer)},
		HostKeyCallback: s.pinnedHostKey(),
		Timeout:         s.dialTimeout,
	}
	addr := net.JoinHostPort(s.target.Host, strconv.Itoa(s.target.Port))

	dialer := &net.Dialer{Timeout: s.dialTimeout}
	conn, err := dialer.DialContext(runCtx, "tcp", addr)
	if err != nil {
		return "", -1, false, fmt.Errorf("ssh: dial %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return "", -1, false, fmt.Errorf("ssh: handshake %s: %w", addr, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", -1, false, fmt.Errorf("ssh: new session: %w", err)
	}
	defer session.Close()

	// Best-effort cancel: close the session/conn when the run context expires
	// (design §3.5 B3 — this is NOT a guaranteed remote kill).
	// timedOut is written by the watchdog goroutine and read by this goroutine
	// after CombinedOutput returns; it MUST be atomic (a plain bool is a data
	// race that `go test -race` flags and that is non-deterministic in prod).
	var timedOut atomic.Bool
	done := make(chan struct{})
	go func() {
		select {
		case <-runCtx.Done():
			if runCtx.Err() == context.DeadlineExceeded {
				timedOut.Store(true)
			}
			_ = session.Close()
			_ = client.Close()
		case <-done:
		}
	}()
	defer close(done)

	// Inject the script via stdin + `bash -s` — avoids re-quoting a multiline
	// #!/bin/bash script (design §3.5 Medium 10). remoteExecCommand が타겟 측
	// timeout으로 감싼다(§3.5 B3 해소) — stdin 프로토콜은 동일.
	session.Stdin = strings.NewReader(script)
	// CombinedOutput runs the command and returns stdout+stderr merged.
	outBytes, runErr := session.CombinedOutput(remoteExecCommand(s.execTimeout))
	out := string(outBytes)

	if timedOut.Load() {
		return out, -1, true, nil
	}
	if runErr != nil {
		var exitErr *ssh.ExitError
		if errors.As(runErr, &exitErr) {
			return out, exitErr.ExitStatus(), false, nil
		}
		return out, -1, false, runErr
	}
	return out, 0, false, nil
}
