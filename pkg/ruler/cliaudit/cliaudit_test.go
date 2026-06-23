package cliaudit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNilLoggerIsNoop(t *testing.T) {
	var l *Logger
	// Must not panic.
	l.Log(Record{Via: "claudecli", Outcome: "ok"})
}

func TestOpenEmptyPathReturnsNilNoop(t *testing.T) {
	l, err := Open("")
	if err != nil {
		t.Fatalf("Open(\"\") err = %v, want nil", err)
	}
	if l != nil {
		t.Fatalf("Open(\"\") = %v, want nil (no-op)", l)
	}
	l.Log(Record{Via: "codexcli", Outcome: "ok"}) // must not panic
}

func TestLogAppendsJSONLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "llm-cli.log")
	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	l.Log(Record{Via: "claudecli", Binary: "claude", Model: "m1", Outcome: "ok", DurationMS: 12, OutputBytes: 100})
	l.Log(Record{Via: "coderca-cli", Binary: "claude", Outcome: "timeout", DurationMS: 300000, Err: "exceeded 5m0s"})
	// Close so the OS releases the handle (Windows cannot delete an open file
	// during t.TempDir cleanup).
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), data)
	}

	var first Record
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal line 0: %v", err)
	}
	if first.Via != "claudecli" || first.Outcome != "ok" || first.Model != "m1" {
		t.Fatalf("line 0 = %+v", first)
	}
	if first.Time == "" {
		t.Fatalf("line 0 Time should be auto-stamped")
	}

	var second Record
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if second.Outcome != "timeout" || second.Err == "" {
		t.Fatalf("line 1 = %+v", second)
	}
}
