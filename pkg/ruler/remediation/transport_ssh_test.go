package remediation

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// startTestSSHD boots a loopback sshd that runs any exec request through `bash`
// (fed via stdin/stdout/stderr on the channel) and returns
// (addr, hostKeyFingerprint, clientPrivateKeyPEM, stop).
//
// It uses only golang.org/x/crypto/ssh: a generated ed25519 host key, a
// generated ed25519 client key accepted by a PublicKeyCallback, and a session
// handler that decodes the "exec" request payload (a length-prefixed command
// string) and runs it via `bash -s` with the channel wired as stdin + combined
// stdout/stderr, then returns the exit status.
func startTestSSHD(t *testing.T) (addr, hostKeyFP, clientKeyPEM string, stop func()) {
	t.Helper()

	// --- host key (ed25519) ---
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatalf("host signer: %v", err)
	}

	// --- client key (ed25519) ---
	_, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen client key: %v", err)
	}
	clientSigner, err := ssh.NewSignerFromKey(clientPriv)
	if err != nil {
		t.Fatalf("client signer: %v", err)
	}
	clientPubMarshaled := clientSigner.PublicKey().Marshal()

	pemBlock, err := ssh.MarshalPrivateKey(clientPriv, "")
	if err != nil {
		t.Fatalf("marshal client private key: %v", err)
	}
	clientKeyPEM = string(pem.EncodeToMemory(pemBlock))
	hostKeyFP = ssh.FingerprintSHA256(hostSigner.PublicKey())

	// --- server config ---
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientPubMarshaled) {
				return &ssh.Permissions{}, nil
			}
			return nil, errUnauthorizedTestKey
		},
	}
	cfg.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	var (
		mu     sync.Mutex
		conns  []net.Conn
		closed bool
	)

	go func() {
		for {
			nConn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			mu.Lock()
			if closed {
				mu.Unlock()
				_ = nConn.Close()
				return
			}
			conns = append(conns, nConn)
			mu.Unlock()
			go handleTestConn(nConn, cfg)
		}
	}()

	stop = func() {
		mu.Lock()
		closed = true
		_ = ln.Close()
		cs := conns
		conns = nil
		mu.Unlock()
		for _, c := range cs {
			_ = c.Close()
		}
	}

	return ln.Addr().String(), hostKeyFP, clientKeyPEM, stop
}

var errUnauthorizedTestKey = &testErr{"unauthorized client key"}

type testErr struct{ s string }

func (e *testErr) Error() string { return e.s }

func handleTestConn(nConn net.Conn, cfg *ssh.ServerConfig) {
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, cfg)
	if err != nil {
		_ = nConn.Close()
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			_ = newChan.Reject(ssh.UnknownChannelType, "only session channels")
			continue
		}
		ch, chReqs, err := newChan.Accept()
		if err != nil {
			continue
		}
		go handleTestSession(ch, chReqs)
	}
}

func handleTestSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	for req := range reqs {
		switch req.Type {
		case "exec":
			// Payload is a single length-prefixed string: the command.
			var payload struct{ Command string }
			if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
				_ = ch.Close()
				return
			}
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			// Run the command with the channel as stdin AND combined stdout/stderr.
			cmd := exec.Command("bash", "-c", payload.Command)
			cmd.Stdin = ch
			cmd.Stdout = ch
			cmd.Stderr = ch
			code := uint32(0)
			if err := cmd.Run(); err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					code = uint32(ee.ExitCode())
				} else {
					code = 1
				}
			}
			_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{code}))
			_ = ch.Close()
			return
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

func mustPort(t *testing.T, addr string) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port %q: %v", addr, err)
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("atoi port %q: %v", portStr, err)
	}
	return p
}

func hostOf(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func TestSSHTransport_RunsRemoteScriptCapturesOutputAndExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test sshd runs the exec payload via `bash` server-side")
	}
	addr, fp, keyPEM, stop := startTestSSHD(t)
	defer stop()
	host, portStr, _ := net.SplitHostPort(addr)
	_ = portStr
	tg := ruletypes.RemediationTarget{
		Host: host, Port: mustPort(t, addr), User: "tester", HostKeyFingerprint: fp,
	}
	tr, err := newSSHTransport(tg, keyPEM, 5*time.Second, 3*time.Second)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	raw, code, timedOut, err := tr.Exec(context.Background(), "#!/bin/bash\necho hello; exit 7")
	if err != nil || timedOut {
		t.Fatalf("exec err=%v timedOut=%v", err, timedOut)
	}
	if code != 7 || !contains(raw, "hello") {
		t.Fatalf("want exit 7 + hello, got code=%d raw=%q", code, raw)
	}
}

func TestSSHTransport_RejectsHostKeyMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test sshd runs the exec payload via `bash` server-side")
	}
	addr, _, keyPEM, stop := startTestSSHD(t)
	defer stop()
	host := hostOf(addr)
	tg := ruletypes.RemediationTarget{
		Host: host, Port: mustPort(t, addr), User: "tester",
		HostKeyFingerprint: "SHA256:deadbeefWRONG", // 프리즈된 지문 불일치
	}
	tr, err := newSSHTransport(tg, keyPEM, 5*time.Second, 3*time.Second)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, _, _, err := tr.Exec(context.Background(), "echo x"); err == nil {
		t.Fatal("must reject connection on host key fingerprint mismatch")
	}
}
