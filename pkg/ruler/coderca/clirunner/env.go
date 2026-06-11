package clirunner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildEnv returns the child-process env carrying the agent's OWN model-API
// auth, reusing the claudecli/codexcli credential-prep approach (design §4/§9):
//
//   - claude: a non-empty token sets CLAUDE_CODE_OAUTH_TOKEN (replacing any
//     inherited value).
//   - codex: a token starting with "{" is treated as ~/.codex/auth.json content
//     (ChatGPT subscription); it is materialized to a fresh CODEX_HOME tempdir
//     and CODEX_HOME points there. Otherwise the token is a raw OPENAI_API_KEY.
//
// BuildEnv carries NO git credential and NO RCA secret — only the agent's model
// auth. The returned cleanup is always non-nil and safe to defer; it removes any
// materialized tempdir.
//
// A2 STUB: returns parent unchanged → auth-injection assertions fail (RED).
func BuildEnv(s Spec, parent []string) (env []string, cleanup func(), err error) {
	return parent, func() {}, nil
}

// appendEnv returns parent with key=value appended, after dropping any inherited
// entries for key and for every name in dropAlso. This makes the configured
// value win over a stale parent entry and prevents a dangling var when switching
// auth modes (e.g. a previous CODEX_HOME must not survive an OPENAI_API_KEY run).
func appendEnv(parent []string, key, value string, dropAlso ...string) []string {
	skip := map[string]struct{}{key: {}}
	for _, d := range dropAlso {
		skip[d] = struct{}{}
	}
	out := make([]string, 0, len(parent)+1)
	for _, e := range parent {
		drop := false
		for k := range skip {
			if strings.HasPrefix(e, k+"=") {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, e)
		}
	}
	return append(out, key+"="+value)
}

// materializeCodexHome writes a codex auth.json paste to a fresh tempdir and
// returns the dir + a cleanup. The paste must be a valid JSON object before we
// touch the filesystem.
func materializeCodexHome(paste string) (dir string, cleanup func(), err error) {
	var probe map[string]any
	if jsonErr := json.Unmarshal([]byte(paste), &probe); jsonErr != nil {
		return "", func() {}, fmt.Errorf("clirunner: codex auth looked like JSON but failed to parse: %w", jsonErr)
	}
	dir, err = os.MkdirTemp("", "coderca-codex-home-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("clirunner: create CODEX_HOME tempdir: %w", err)
	}
	if writeErr := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(paste), 0o600); writeErr != nil {
		_ = os.RemoveAll(dir)
		return "", func() {}, fmt.Errorf("clirunner: write auth.json: %w", writeErr)
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}
