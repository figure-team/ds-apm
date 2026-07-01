// Package cliaudit provides a process-wide, append-only audit log for every
// time DS-APM actually shells out to an LLM CLI (the `claude` / `codex`
// binaries). It exists to answer one operational question that the general
// server logs cannot answer cheaply: "did this incident notification / Code
// RCA really invoke the LLM CLI, or did it silently fall back to the
// mock/local generator?"
//
// The log is written at the exact exec seam — one JSON line per spawned
// process — so the file is truthful by construction: a line present means the
// CLI ran; an empty file during an incident means the CLI path was never taken
// (mock/local fallback, or the request was gated/suppressed before exec).
//
// Configuration is a single env var, DS_APM_LLM_CLI_LOG. When it is empty the
// logger is a no-op (nil *Logger), so this package adds zero behavior unless an
// operator opts in by pointing it at a file path (e.g. a mounted volume).
package cliaudit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EnvPath is the environment variable naming the audit log file. Empty/unset
// disables auditing.
const EnvPath = "DS_APM_LLM_CLI_LOG"

// Record is one CLI invocation outcome. Fields left zero are omitted from the
// JSON line where it is safe to do so.
type Record struct {
	// Time is stamped at write if empty (RFC3339 with nanoseconds, UTC).
	Time string `json:"time"`
	// Via identifies the exec seam: "claudecli", "codexcli", or "coderca-cli".
	Via string `json:"via"`
	// Binary is the resolved executable path that was spawned.
	Binary string `json:"binary"`
	// Model is the model the CLI was asked to use, when known.
	Model string `json:"model,omitempty"`
	// Outcome is "ok", "timeout", or "failed".
	Outcome string `json:"outcome"`
	// DurationMS is wall-clock time spent in the spawned process.
	DurationMS int64 `json:"duration_ms"`
	// OutputBytes is the number of stdout bytes captured.
	OutputBytes int `json:"output_bytes,omitempty"`
	// Err is a truncated error summary when Outcome != "ok".
	Err string `json:"err,omitempty"`
	// Source records the remediation script origin when Via is a remediation
	// exec seam: "runbook" or "llm-generated". Empty for non-remediation records.
	Source string `json:"source,omitempty"`
	// Fingerprint is the alert fingerprint tied to this invocation, when known.
	Fingerprint string `json:"fingerprint,omitempty"`
	// Transport is the remediation execution channel: "local" or "ssh". Empty for
	// non-remediation records (design §3.4).
	Transport string `json:"transport,omitempty"`
	// Target is the remote host when Transport is "ssh". Empty otherwise.
	Target string `json:"target,omitempty"`
}

// Logger appends one JSON line per CLI invocation to a file. It is safe for
// concurrent use. A nil *Logger is a valid no-op, so callers can hold a
// possibly-nil *Logger and call Log without branching.
type Logger struct {
	mu sync.Mutex
	f  *os.File
}

// Open returns a Logger appending to path. An empty path yields a nil (no-op)
// Logger and no error. A non-empty path that cannot be opened returns a nil
// Logger and the error, so callers can warn and proceed without auditing
// rather than fail startup.
func Open(path string) (*Logger, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		// Best-effort: a failing MkdirAll surfaces through OpenFile below.
		_ = os.MkdirAll(dir, 0o755)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cliaudit: open %q: %w", path, err)
	}
	return &Logger{f: f}, nil
}

// Log appends r as a single JSON line. Best-effort: a nil logger, marshal
// error, or write error is silently ignored — auditing must never break the
// CLI invocation it is observing.
func (l *Logger) Log(r Record) {
	if l == nil || l.f == nil {
		return
	}
	if r.Time == "" {
		r.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.Marshal(r)
	if err != nil {
		return
	}
	b = append(b, '\n')
	l.mu.Lock()
	_, _ = l.f.Write(b)
	l.mu.Unlock()
}

// Close releases the underlying file. Safe on a nil Logger. The process-wide
// Default logger is intentionally never closed (it lives for the process
// lifetime); Close exists mainly for tests and short-lived loggers.
func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.f.Close()
}

var (
	defaultOnce sync.Once
	defaultLog  *Logger
)

// Default returns the process-wide audit logger configured from EnvPath. It is
// initialized exactly once on first call; later changes to the env var are
// ignored. Returns nil (a valid no-op) when EnvPath is empty or the file
// cannot be opened (the failure is noted on stderr).
func Default() *Logger {
	defaultOnce.Do(func() {
		l, err := Open(os.Getenv(EnvPath))
		if err != nil {
			fmt.Fprintf(os.Stderr, "cliaudit: disabled: %v\n", err)
			return
		}
		defaultLog = l
	})
	return defaultLog
}
