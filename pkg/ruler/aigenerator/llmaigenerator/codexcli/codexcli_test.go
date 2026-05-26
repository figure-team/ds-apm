package codexcli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// writeFakeCodexBinary writes an executable shell script at dir/codex and
// returns its path. The script behaviour:
//   - If env var FAKE_CODEX_SLEEP != "", sleeps that many seconds.
//   - If env var FAKE_CODEX_PROMPT_FILE != "", writes the last argument (the
//     combined prompt) to that file.
//   - If env var FAKE_CODEX_EXIT != "", exits with that code and writes
//     FAKE_CODEX_STDERR to stderr.
//   - Otherwise echoes FAKE_CODEX_OUTPUT (default: '{"ok":true}') to stdout.
func writeFakeCodexBinary(t *testing.T, dir string) string {
	t.Helper()
	script := `#!/bin/sh
if [ -n "$FAKE_CODEX_SLEEP" ]; then
  sleep "$FAKE_CODEX_SLEEP"
fi
if [ -n "$FAKE_CODEX_PROMPT_FILE" ]; then
  # last argument is the combined prompt
  eval "last=\${$#}"
  printf '%s' "$last" > "$FAKE_CODEX_PROMPT_FILE"
fi
if [ -n "$FAKE_CODEX_EXIT" ]; then
  if [ -n "$FAKE_CODEX_STDERR" ]; then
    printf '%s' "$FAKE_CODEX_STDERR" >&2
  fi
  exit "$FAKE_CODEX_EXIT"
fi
output=${FAKE_CODEX_OUTPUT:-'{"ok":true}'}
printf '%s\n' "$output"
`
	path := filepath.Join(dir, "codex")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

// TestProvider_CompleteRunsBinaryAndReturnsStdout verifies the happy path:
// fake script echoes canned JSON, Complete returns it verbatim (with newline).
func TestProvider_CompleteRunsBinaryAndReturnsStdout(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeCodexBinary(t, dir)

	t.Setenv("FAKE_CODEX_OUTPUT", `{"ok":true}`)

	p := New(WithBinary(bin))
	got, err := p.Complete(context.Background(), "sys", "user")
	require.NoError(t, err)
	require.Equal(t, `{"ok":true}`+"\n", got)
}

// TestProvider_CompleteFailureIncludesStderr verifies that when the script
// exits non-zero the error message contains the stderr text.
func TestProvider_CompleteFailureIncludesStderr(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeCodexBinary(t, dir)

	t.Setenv("FAKE_CODEX_EXIT", "2")
	t.Setenv("FAKE_CODEX_STDERR", "boom")

	p := New(WithBinary(bin))
	_, err := p.Complete(context.Background(), "sys", "user")
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}

// TestProvider_CompletePassesPromptCorrectly verifies that the combined prompt
// (system + "---" + user) is passed as the last argument to the binary.
func TestProvider_CompletePassesPromptCorrectly(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeCodexBinary(t, dir)
	promptFile := filepath.Join(dir, "prompt.txt")

	t.Setenv("FAKE_CODEX_PROMPT_FILE", promptFile)

	const wantSystem = "you are a calculator"
	const wantUser = "what is 2+2?"

	p := New(WithBinary(bin))
	_, err := p.Complete(context.Background(), wantSystem, wantUser)
	require.NoError(t, err)

	raw, err := os.ReadFile(promptFile)
	require.NoError(t, err)
	combined := string(raw)

	require.Contains(t, combined, wantSystem)
	require.Contains(t, combined, "---")
	require.Contains(t, combined, wantUser)
	// system must appear before user
	require.Less(t, strings.Index(combined, wantSystem), strings.Index(combined, wantUser))
}

func TestProvider_EnvContainsOpenAIKeyWhenOAuthTokenSet(t *testing.T) {
	p := New(WithBinary("/bin/true"), WithOAuthToken("sk-xyz"))
	found := false
	for _, e := range p.commandEnv([]string{"PATH=/usr/bin"}) {
		if e == "OPENAI_API_KEY=sk-xyz" {
			found = true
		}
	}
	if !found {
		t.Fatalf("OPENAI_API_KEY not injected")
	}
}

func TestProvider_NoOAuthEnvWhenEmpty(t *testing.T) {
	p := New(WithBinary("/bin/true"))
	for _, e := range p.commandEnv([]string{"PATH=/usr/bin"}) {
		if strings.HasPrefix(e, "OPENAI_API_KEY=") {
			t.Fatalf("unexpected env: %s", e)
		}
	}
}

func TestProvider_DropsInheritedOpenAIKey(t *testing.T) {
	p := New(WithOAuthToken("override"))
	count := 0
	for _, e := range p.commandEnv([]string{"OPENAI_API_KEY=stale", "PATH=/usr/bin"}) {
		if strings.HasPrefix(e, "OPENAI_API_KEY=") {
			count++
			if e != "OPENAI_API_KEY=override" {
				t.Fatalf("wrong value: %s", e)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one OPENAI_API_KEY entry, got %d", count)
	}
}

func TestPrepareEnv_EmptyTokenIsPassThrough(t *testing.T) {
	p := New()
	env, cleanup, err := p.prepareEnv([]string{"PATH=/usr/bin"})
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	cleanup() // must be a safe no-op
	require.Equal(t, []string{"PATH=/usr/bin"}, env)
}

func TestPrepareEnv_RawAPIKeyInjectsOpenAIKey(t *testing.T) {
	p := New(WithOAuthToken("sk-test"))
	env, cleanup, err := p.prepareEnv([]string{"CODEX_HOME=/leftover", "PATH=/usr/bin"})
	require.NoError(t, err)
	defer cleanup()
	require.Contains(t, env, "OPENAI_API_KEY=sk-test")
	for _, e := range env {
		require.False(t, strings.HasPrefix(e, "CODEX_HOME="),
			"raw API key path must drop stale CODEX_HOME entries; got %q", e)
	}
}

func TestPrepareEnv_JSONTokenMaterializesAuthFile(t *testing.T) {
	payload := `{"auth_mode":"chatgpt","tokens":{"access_token":"a","refresh_token":"r"},"OPENAI_API_KEY":null}`
	p := New(WithOAuthToken(payload))
	env, cleanup, err := p.prepareEnv([]string{"OPENAI_API_KEY=stale", "PATH=/usr/bin"})
	require.NoError(t, err)
	defer cleanup()

	var codexHome string
	for _, e := range env {
		if strings.HasPrefix(e, "CODEX_HOME=") {
			codexHome = strings.TrimPrefix(e, "CODEX_HOME=")
		}
		require.False(t, strings.HasPrefix(e, "OPENAI_API_KEY="),
			"JSON path must drop OPENAI_API_KEY so it doesn't shadow auth.json: got %q", e)
	}
	require.NotEmpty(t, codexHome, "CODEX_HOME must be set when JSON token is configured")

	authBytes, readErr := os.ReadFile(filepath.Join(codexHome, "auth.json"))
	require.NoError(t, readErr)
	require.JSONEq(t, payload, string(authBytes))
}

func TestPrepareEnv_JSONTokenCleanupRemovesTempdir(t *testing.T) {
	payload := `{"auth_mode":"chatgpt","tokens":{}}`
	p := New(WithOAuthToken(payload))
	env, cleanup, err := p.prepareEnv([]string{"PATH=/usr/bin"})
	require.NoError(t, err)
	var codexHome string
	for _, e := range env {
		if strings.HasPrefix(e, "CODEX_HOME=") {
			codexHome = strings.TrimPrefix(e, "CODEX_HOME=")
		}
	}
	require.DirExists(t, codexHome)
	cleanup()
	_, statErr := os.Stat(codexHome)
	require.Error(t, statErr, "tempdir should be gone after cleanup")
	require.True(t, os.IsNotExist(statErr))
}

func TestPrepareEnv_InvalidJSONErrors(t *testing.T) {
	p := New(WithOAuthToken(`{this is not json`))
	_, cleanup, err := p.prepareEnv([]string{"PATH=/usr/bin"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse")
	if cleanup != nil {
		cleanup() // must be safe even on error path
	}
}

// TestProvider_CompleteRespectsContextCancel verifies that a context deadline
// causes Complete to return well before the fake sleep finishes.
func TestProvider_CompleteRespectsContextCancel(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeCodexBinary(t, dir)

	t.Setenv("FAKE_CODEX_SLEEP", "2")

	p := New(WithBinary(bin))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := p.Complete(ctx, "sys", "user")
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, 500*time.Millisecond, "Complete should have returned quickly after context cancel")
}
