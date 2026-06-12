package clirunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hasEnv(env []string, key, val string) bool {
	want := key + "=" + val
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}

func envValue(env []string, key string) (string, bool) {
	p := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, p) {
			return e[len(p):], true
		}
	}
	return "", false
}

func countEnv(env []string, key string) int {
	p := key + "="
	n := 0
	for _, e := range env {
		if strings.HasPrefix(e, p) {
			n++
		}
	}
	return n
}

func TestBuildEnvClaudeTokenReplacesStale(t *testing.T) {
	parent := []string{"PATH=/usr/bin", "CLAUDE_CODE_OAUTH_TOKEN=stale"}
	env, cleanup, err := BuildEnv(Spec{Agent: AgentClaude, AuthToken: "fresh-token"}, parent)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if !hasEnv(env, "CLAUDE_CODE_OAUTH_TOKEN", "fresh-token") {
		t.Errorf("CLAUDE_CODE_OAUTH_TOKEN not set to fresh token: %v", env)
	}
	if n := countEnv(env, "CLAUDE_CODE_OAUTH_TOKEN"); n != 1 {
		t.Errorf("CLAUDE_CODE_OAUTH_TOKEN appears %d times, want 1 (stale not replaced)", n)
	}
}

func TestBuildEnvClaudeNoTokenLeavesParent(t *testing.T) {
	parent := []string{"PATH=/usr/bin"}
	env, cleanup, err := BuildEnv(Spec{Agent: AgentClaude}, parent)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != len(parent) {
		t.Errorf("env changed without a token: %v", env)
	}
}

func TestBuildEnvCodexRawKey(t *testing.T) {
	parent := []string{"PATH=/usr/bin", "CODEX_HOME=/old", "OPENAI_API_KEY=stale"}
	env, cleanup, err := BuildEnv(Spec{Agent: AgentCodex, AuthToken: "sk-123"}, parent)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if !hasEnv(env, "OPENAI_API_KEY", "sk-123") {
		t.Errorf("OPENAI_API_KEY not set: %v", env)
	}
	if _, ok := envValue(env, "CODEX_HOME"); ok {
		t.Errorf("stale CODEX_HOME not dropped for raw-key mode: %v", env)
	}
}

func TestBuildEnvCodexJSONPasteMaterializes(t *testing.T) {
	paste := `{"OPENAI_API_KEY":"sk-abc","tokens":{"access":"x"}}`
	env, cleanup, err := BuildEnv(Spec{Agent: AgentCodex, AuthToken: paste}, []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}
	home, ok := envValue(env, "CODEX_HOME")
	if !ok {
		cleanup()
		t.Fatal("CODEX_HOME not set for JSON paste")
	}
	b, readErr := os.ReadFile(filepath.Join(home, "auth.json"))
	if readErr != nil {
		cleanup()
		t.Fatalf("auth.json not written: %v", readErr)
	}
	if string(b) != paste {
		t.Errorf("auth.json content = %q, want %q", string(b), paste)
	}
	cleanup()
	if _, statErr := os.Stat(home); !os.IsNotExist(statErr) {
		t.Errorf("CODEX_HOME %q not cleaned up", home)
	}
}

func TestBuildEnvCodexInvalidJSONFails(t *testing.T) {
	_, cleanup, err := BuildEnv(Spec{Agent: AgentCodex, AuthToken: "{not valid"}, nil)
	defer cleanup()
	if err == nil {
		t.Error("expected error for malformed JSON paste, got nil")
	}
}
