// Package claudecli implements llmaigenerator.Provider by shelling out to the
// local `claude` CLI (Claude Code). The CLI must already be authenticated.
// Intended for local demos and dev only — do NOT run in production servers.
package claudecli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator"
	"github.com/SigNoz/signoz/pkg/ruler/cliaudit"
)

const DefaultBinary = "claude"
const DefaultModel = "claude-sonnet-4-6"

// Provider implements llmaigenerator.Provider by shelling out to the local
// `claude` CLI (Claude Code). The CLI must already be authenticated. The
// caller is responsible for not running this in production server contexts —
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
	extra      []string // extra args appended after -p
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
	p.binary = resolveBinary(p.binary)
	return p
}

// resolveBinary returns the executable path for name.
// It tries exec.LookPath first; if that fails it asks npm where its global
// prefix lives and probes there. This recovers the common case where the server
// process inherits a narrow PATH that omits the user-scoped npm global bin
// directory (typical on Windows where npm installs per-user).
// If name is already an absolute path it is returned as-is.
func resolveBinary(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	// Ask npm for its global prefix and probe <prefix>/<name> (Windows) and
	// <prefix>/bin/<name> (Unix/macOS).
	out, err := exec.Command("npm", "config", "get", "prefix").Output()
	if err != nil {
		return name
	}
	prefix := strings.TrimSpace(string(out))
	for _, rel := range []string{name, filepath.Join("bin", name)} {
		if p, err := exec.LookPath(filepath.Join(prefix, rel)); err == nil {
			return p
		}
	}
	return name
}

// commandEnv returns the env slice to pass to cmd.Env. If an OAuth token is
// configured, CLAUDE_CODE_OAUTH_TOKEN is appended (overriding any inherited
// value).
func (p *Provider) commandEnv(parent []string) []string {
	if p.oauthToken == "" {
		return parent
	}
	out := make([]string, 0, len(parent)+1)
	for _, e := range parent {
		if strings.HasPrefix(e, "CLAUDE_CODE_OAUTH_TOKEN=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, "CLAUDE_CODE_OAUTH_TOKEN="+p.oauthToken)
}

// Complete shells out to: `<binary> -p <user> --append-system-prompt <system> --model <model>`
// stdout is captured as the model output; stderr is included in error messages.
func (p *Provider) Complete(ctx context.Context, system, user string) (string, error) {
	args := []string{
		"-p", user,
		"--append-system-prompt", system,
		"--model", p.model,
	}
	args = append(args, p.extra...)
	cmd := exec.CommandContext(ctx, p.binary, args...)
	cmd.Env = p.commandEnv(os.Environ())
	cmd.WaitDelay = 200 * time.Millisecond // forcibly reap subprocess after ctx cancel
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	rec := cliaudit.Record{
		Via:         "claudecli",
		Binary:      p.binary,
		Model:       p.model,
		DurationMS:  time.Since(start).Milliseconds(),
		OutputBytes: stdout.Len(),
		Outcome:     "ok",
	}
	if runErr != nil {
		rec.Outcome = "failed"
		if ctx.Err() == context.DeadlineExceeded {
			rec.Outcome = "timeout"
		}
		rec.Err = truncate(strings.TrimSpace(stderr.String()), 256)
	}
	cliaudit.Default().Log(rec)

	if runErr != nil {
		return "", fmt.Errorf("claudecli: run %s: %w (stderr: %s)", p.binary, runErr, truncate(stderr.String(), 512))
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
