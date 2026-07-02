package remediation

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// FetchHostKeyFingerprint dials host:port, records the server host key during the
// SSH handshake, and returns its SHA256 fingerprint + key type. It never
// authenticates — the host key arrives before auth, so an auth failure is
// expected and ignored (the fingerprint is already captured by then). The
// record-only callback here is EXCLUSIVE to this probe (design §3.4); the
// execution path keeps pinnedHostKey. InsecureIgnoreHostKey is never used.
func FetchHostKeyFingerprint(ctx context.Context, host string, port int, dialTimeout time.Duration) (string, string, error) {
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	var fingerprint, keyType string
	cfg := &ssh.ClientConfig{
		User: "ds-apm-fingerprint-probe",
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			fingerprint = ssh.FingerprintSHA256(key)
			keyType = key.Type()
			return nil
		},
		Timeout: dialTimeout,
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", "", fmt.Errorf("ssh: dial %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err == nil {
		_ = ssh.NewClient(sshConn, chans, reqs).Close()
	} else {
		_ = conn.Close()
	}
	// auth 실패(err != nil)여도 호스트키는 이미 콜백으로 획득됐다.
	if fingerprint == "" {
		return "", "", fmt.Errorf("ssh: no host key from %s: %w", addr, err)
	}
	return fingerprint, keyType, nil
}

// TestConnection runs `echo ok` on target over SSH with fixed short timeouts
// (dial 5s / exec 10s — the org ExecTimeout is intentionally not used here,
// design §3.4). It reuses the production newSSHTransport + Exec so the pinned
// host-key verification is identical to the execution path. The signozruler
// handler needs this exported wrapper because newSSHTransport is unexported.
func TestConnection(ctx context.Context, target ruletypes.RemediationTarget, privateKeyPEM string) (string, int, error) {
	tr, err := newSSHTransport(target, privateKeyPEM, 10*time.Second, 5*time.Second)
	if err != nil {
		return "", -1, err
	}
	out, code, timedOut, err := tr.Exec(ctx, "echo ok")
	if err != nil {
		return out, -1, err
	}
	if timedOut {
		return out, -1, fmt.Errorf("ssh: connection test timed out")
	}
	return out, code, nil
}
