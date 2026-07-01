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
	// #!/bin/bash script (design §3.5 Medium 10).
	session.Stdin = strings.NewReader(script)
	// CombinedOutput runs the command and returns stdout+stderr merged.
	outBytes, runErr := session.CombinedOutput("bash -s")
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
