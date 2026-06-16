//go:build !windows

package claudecli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// writeFakeClaudeBinary writes an executable shell script at dir/claude and
// returns its path. The script behaviour:
//   - If env var FAKE_CLAUDE_EXIT != "", exits with that code and writes
//     FAKE_CLAUDE_STDERR to stderr.
//   - If env var FAKE_CLAUDE_ARGS_FILE != "", dumps "$@" (one arg per line)
//     to that file.
//   - If env var FAKE_CLAUDE_SLEEP != "", sleeps that many seconds.
//   - Otherwise echoes FAKE_CLAUDE_OUTPUT (default: {"ok":true}) to stdout.
func writeFakeClaudeBinary(t *testing.T, dir string) string {
	t.Helper()
	script := `#!/bin/sh
if [ -n "$FAKE_CLAUDE_SLEEP" ]; then
  sleep "$FAKE_CLAUDE_SLEEP"
fi
if [ -n "$FAKE_CLAUDE_ARGS_FILE" ]; then
  for arg in "$@"; do
    printf '%s\n' "$arg" >> "$FAKE_CLAUDE_ARGS_FILE"
  done
fi
if [ -n "$FAKE_CLAUDE_EXIT" ]; then
  if [ -n "$FAKE_CLAUDE_STDERR" ]; then
    printf '%s' "$FAKE_CLAUDE_STDERR" >&2
  fi
  exit "$FAKE_CLAUDE_EXIT"
fi
output=${FAKE_CLAUDE_OUTPUT:-'{"ok":true}'}
printf '%s\n' "$output"
`
	path := filepath.Join(dir, "claude")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestProvider_CompleteRunsBinaryAndReturnsStdout(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeClaudeBinary(t, dir)

	t.Setenv("FAKE_CLAUDE_OUTPUT", `{"ok":true}`)

	p := New(WithBinary(bin))
	got, err := p.Complete(context.Background(), "sys", "user")
	require.NoError(t, err)
	require.Equal(t, `{"ok":true}`+"\n", got)
}

func TestProvider_CompleteFailureIncludesStderr(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeClaudeBinary(t, dir)

	t.Setenv("FAKE_CLAUDE_EXIT", "2")
	t.Setenv("FAKE_CLAUDE_STDERR", "boom")

	p := New(WithBinary(bin))
	_, err := p.Complete(context.Background(), "sys", "user")
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}

func TestProvider_CompletePassesArgsCorrectly(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeClaudeBinary(t, dir)
	argsFile := filepath.Join(dir, "args.txt")

	t.Setenv("FAKE_CLAUDE_ARGS_FILE", argsFile)

	const wantModel = "claude-test-v1"
	const wantUser = "what is 2+2?"
	const wantSystem = "you are a calculator"

	p := New(WithBinary(bin), WithModel(wantModel))
	_, err := p.Complete(context.Background(), wantSystem, wantUser)
	require.NoError(t, err)

	raw, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	args := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")

	argMap := make(map[string]string)
	for i := 0; i < len(args)-1; i++ {
		if strings.HasPrefix(args[i], "-") {
			argMap[args[i]] = args[i+1]
		}
	}

	require.Equal(t, wantUser, argMap["-p"])
	require.Equal(t, wantSystem, argMap["--append-system-prompt"])
	require.Equal(t, wantModel, argMap["--model"])
}

func TestProvider_CompleteRespectsContextCancel(t *testing.T) {
	dir := t.TempDir()
	bin := writeFakeClaudeBinary(t, dir)

	t.Setenv("FAKE_CLAUDE_SLEEP", "2")

	p := New(WithBinary(bin))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := p.Complete(ctx, "sys", "user")
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, 500*time.Millisecond, "Complete should have returned quickly after context cancel")
}
