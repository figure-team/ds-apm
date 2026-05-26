// Package codexcli implements llmaigenerator.Provider by shelling out to the
// local `codex` CLI (OpenAI Codex companion). The CLI must already be
// authenticated. Intended for local demos and dev only — do NOT run in
// production servers.
package codexcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator"
)

const (
	DefaultBinary = "codex"
	DefaultModel  = "gpt-5"
)

// Provider implements llmaigenerator.Provider by shelling out to the local
// `codex` CLI (OpenAI Codex companion). The CLI must already be authenticated.
// The caller is responsible for not running this in production server contexts —
// it's intended for local demos and dev.
//
// Concurrency: Provider is immutable after New. All fields are written only
// through functional options inside New(...) and read-only thereafter, so
// Complete is safe to call concurrently from multiple goroutines. Do NOT add
// setter methods or mutate fields post-construction — callers rely on this
// for race-free reuse across alert dispatches.
type Provider struct {
	binary     string
	model      string
	extra      []string // extra args appended after the prompt
	oauthToken string
}

type Option func(*Provider)

func WithBinary(path string) Option    { return func(p *Provider) { p.binary = path } }
func WithModel(m string) Option        { return func(p *Provider) { p.model = m } }
func WithExtraArgs(a ...string) Option { return func(p *Provider) { p.extra = append(p.extra, a...) } }
func WithOAuthToken(t string) Option   { return func(p *Provider) { p.oauthToken = t } }

func New(opts ...Option) *Provider {
	p := &Provider{binary: DefaultBinary, model: DefaultModel}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// prepareEnv builds the env slice for the child codex process based on the
// configured oauth token.
//
//   - Empty token → return parent unchanged. No-op cleanup.
//   - Token starts with "{" → treat as ~/.codex/auth.json content (ChatGPT
//     subscription auth). Materialize to a fresh tempdir + auth.json and
//     point CODEX_HOME at it; cleanup removes the tempdir.
//   - Otherwise → treat as a raw OPENAI_API_KEY. Inject into env and drop any
//     inherited value. No-op cleanup.
//
// The cleanup function is always non-nil and safe to defer.
func (p *Provider) prepareEnv(parent []string) (env []string, cleanup func(), err error) {
	cleanup = func() {}
	if p.oauthToken == "" {
		return parent, cleanup, nil
	}

	trimmed := strings.TrimSpace(p.oauthToken)
	if strings.HasPrefix(trimmed, "{") {
		// JSON paste — must be valid object before we touch the filesystem.
		var probe map[string]any
		if jsonErr := json.Unmarshal([]byte(trimmed), &probe); jsonErr != nil {
			return nil, cleanup, fmt.Errorf("codexcli: oauthToken looked like JSON but failed to parse: %w", jsonErr)
		}
		dir, mkErr := os.MkdirTemp("", "codex-home-*")
		if mkErr != nil {
			return nil, cleanup, fmt.Errorf("codexcli: create CODEX_HOME tempdir: %w", mkErr)
		}
		authPath := filepath.Join(dir, "auth.json")
		if writeErr := os.WriteFile(authPath, []byte(trimmed), 0o600); writeErr != nil {
			_ = os.RemoveAll(dir)
			return nil, cleanup, fmt.Errorf("codexcli: write auth.json: %w", writeErr)
		}
		cleanup = func() { _ = os.RemoveAll(dir) }
		return appendEnv(parent, "CODEX_HOME", dir, "OPENAI_API_KEY"), cleanup, nil
	}

	// Raw API key path.
	return appendEnv(parent, "OPENAI_API_KEY", p.oauthToken, "CODEX_HOME"), cleanup, nil
}

// appendEnv returns parent with the supplied key=value pair appended, after
// dropping any inherited entries for the set keyword AND for any of dropAlso.
// This ensures the configured token wins over a stale parent env AND that
// switching between auth modes does not leave the other env var dangling
// (e.g., a previous CODEX_HOME shouldn't survive when we want OPENAI_API_KEY
// only).
func appendEnv(parent []string, key, value string, dropAlso ...string) []string {
	out := make([]string, 0, len(parent)+1)
	skip := map[string]struct{}{key: {}}
	for _, d := range dropAlso {
		skip[d] = struct{}{}
	}
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

// commandEnv preserves the old single-purpose helper for backward-compatible
// tests: returns env with only OPENAI_API_KEY injection (no JSON detection,
// no filesystem side effects). Callers that need full functionality must use
// prepareEnv.
func (p *Provider) commandEnv(parent []string) []string {
	if p.oauthToken == "" {
		return parent
	}
	return appendEnv(parent, "OPENAI_API_KEY", p.oauthToken, "CODEX_HOME")
}

// Complete shells out to: `<binary> exec --model <model> <combined-prompt>`
// where combined-prompt = "<system>\n\n---\n\n<user>" (codex CLI has no
// separate system prompt flag).
func (p *Provider) Complete(ctx context.Context, system, user string) (string, error) {
	prompt := user
	if system != "" {
		prompt = system + "\n\n---\n\n" + user
	}
	args := []string{"exec", "--model", p.model}
	args = append(args, p.extra...)
	args = append(args, prompt)

	env, cleanup, err := p.prepareEnv(os.Environ())
	if err != nil {
		return "", err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, p.binary, args...)
	cmd.Env = env
	cmd.WaitDelay = 200 * time.Millisecond // forcibly reap subprocess after ctx cancel
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("codexcli: run %s: %w (stderr: %s)", p.binary, err, truncate(stderr.String(), 512))
	}
	return stdout.String(), nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// compile-time assertion: Provider must satisfy llmaigenerator.Provider.
var _ llmaigenerator.Provider = (*Provider)(nil)
