package remediation

import (
	"context"
	"runtime"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// TestFetchHostKeyFingerprint verifies the probe returns the server's SHA256 host
// key fingerprint even though it never authenticates. The in-process test sshd
// (startTestSSHD) exposes its host key fingerprint directly, so we compare against
// that. The probe only needs the SSH handshake (host key exchange happens before
// auth), so this runs on Windows too — no bash exec is involved.
func TestFetchHostKeyFingerprint(t *testing.T) {
	addr, hostKeyFP, _, stop := startTestSSHD(t)
	defer stop()
	host := hostOf(addr)
	port := mustPort(t, addr)

	fp, keyType, err := FetchHostKeyFingerprint(context.Background(), host, port, 3*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp != hostKeyFP {
		t.Fatalf("fingerprint: got %s want %s", fp, hostKeyFP)
	}
	if !contains(fp, "SHA256:") {
		t.Fatalf("fingerprint should be SHA256-prefixed, got %q", fp)
	}
	if keyType != ssh.KeyAlgoED25519 {
		t.Fatalf("keyType: got %q want %q", keyType, ssh.KeyAlgoED25519)
	}
}

func TestFetchHostKeyFingerprint_DialError(t *testing.T) {
	_, _, err := FetchHostKeyFingerprint(context.Background(), "127.0.0.1", 1, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected dial error")
	}
}

// TestTestConnection_OK checks the happy path: correct client key + frozen
// fingerprint → exit 0 with "ok" in the output. The harness runs the remote
// command via `bash` server-side, so this is skipped on Windows (same guard the
// transport tests use).
func TestTestConnection_OK(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test sshd runs the exec payload via `bash` server-side")
	}
	addr, fp, keyPEM, stop := startTestSSHD(t)
	defer stop()
	tg := ruletypes.RemediationTarget{
		Host: hostOf(addr), Port: mustPort(t, addr), User: "tester", HostKeyFingerprint: fp,
	}
	out, code, err := TestConnection(context.Background(), tg, keyPEM)
	if err != nil || code != 0 || !contains(out, "ok") {
		t.Fatalf("got out=%q code=%d err=%v", out, code, err)
	}
}

// TestTestConnection_BadFingerprint checks the pinned host-key path rejects a
// mismatched fingerprint. Rejection happens during the SSH handshake (before any
// remote exec), so no server-side bash is needed and this runs on all platforms.
func TestTestConnection_BadFingerprint(t *testing.T) {
	addr, _, keyPEM, stop := startTestSSHD(t)
	defer stop()
	tg := ruletypes.RemediationTarget{
		Host: hostOf(addr), Port: mustPort(t, addr), User: "tester",
		HostKeyFingerprint: "SHA256:wrongwrongwrong",
	}
	_, _, err := TestConnection(context.Background(), tg, keyPEM)
	if err == nil || !contains(err.Error(), "host key mismatch") {
		t.Fatalf("expected host key mismatch, got %v", err)
	}
}

// TestTestConnection_BadKey checks an unparseable private key fails fast at
// transport construction (before any dial).
func TestTestConnection_BadKey(t *testing.T) {
	tg := ruletypes.RemediationTarget{
		Host: "127.0.0.1", Port: 22, User: "tester", HostKeyFingerprint: "SHA256:whatever",
	}
	_, code, err := TestConnection(context.Background(), tg, "not-a-valid-pem")
	if err == nil || code != -1 {
		t.Fatalf("expected parse error with code -1, got code=%d err=%v", code, err)
	}
}
