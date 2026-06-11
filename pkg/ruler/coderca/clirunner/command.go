// Package clirunner drives a CLI coding agent (claude / codex) against a
// per-run source checkout to produce a root-cause analysis (design §9). It
// reuses the env/credential-prep approach of the existing claudecli/codexcli
// adapters but adds the things code-RCA needs: cmd.Dir scoped to the checkout,
// read-only tool enforcement, minutes-scale timeouts, and process-group /
// parent-death subprocess containment (§6.5).
//
// The agent receives NO secrets. Git read-credentials are delivered to git via
// GIT_ASKPASS out of band; the agent's only writable surface is the disposable
// checkout, which is removed after the run.
package clirunner

import (
	"errors"
	"fmt"
	"strings"
)

// Agent selects which CLI coding agent to drive.
type Agent string

const (
	AgentClaude Agent = "claude"
	AgentCodex  Agent = "codex"
)

// claude has no OS-level read-only sandbox, so read-only is enforced at the
// application level: allow only read/search tools, forbid every write / exec /
// network tool, and cap spend with --max-budget-usd (design §9).
const (
	claudeAllowedTools    = "Read,Grep,Glob"
	claudeDisallowedTools = "Bash,Write,Edit,WebFetch,WebSearch"
)

// Spec is a single CLI invocation. Secrets are NEVER part of a Spec.
type Spec struct {
	Agent        Agent
	Binary       string // resolved binary; DefaultBinary(agent) when empty
	Model        string
	Checkout     string // absolute path to the per-run checkout (cmd.Dir + scope)
	SystemPrompt string
	Prompt       string
	MaxBudgetUSD string // claude hard $ ceiling, e.g. "0.50" — REQUIRED for claude
}

var (
	ErrNoCheckout   = errors.New("clirunner: checkout dir is required (read-only scope)")
	ErrNoModel      = errors.New("clirunner: model is required")
	ErrNoBudgetCap  = errors.New("clirunner: claude requires a --max-budget-usd ceiling")
	ErrUnknownAgent = errors.New("clirunner: unknown agent")
)

// DefaultBinary returns the conventional binary name for an agent.
func DefaultBinary(a Agent) string {
	switch a {
	case AgentClaude:
		return "claude"
	case AgentCodex:
		return "codex"
	default:
		return ""
	}
}

// BuildArgs returns the exact argv (excluding the binary) for the configured
// agent, built with read-only enforcement flags (design §9). This build IS the
// security contract: it must never produce a write-capable or unscoped
// invocation. Pure + table-tested.
//
func BuildArgs(s Spec) ([]string, error) {
	if strings.TrimSpace(s.Checkout) == "" {
		return nil, ErrNoCheckout
	}
	if strings.TrimSpace(s.Model) == "" {
		return nil, ErrNoModel
	}
	switch s.Agent {
	case AgentClaude:
		if strings.TrimSpace(s.MaxBudgetUSD) == "" {
			return nil, ErrNoBudgetCap
		}
		return []string{
			"-p", s.Prompt,
			"--append-system-prompt", s.SystemPrompt,
			"--model", s.Model,
			"--add-dir", s.Checkout,
			"--permission-mode", "default",
			"--allowed-tools", claudeAllowedTools,
			"--disallowed-tools", claudeDisallowedTools,
			"--max-budget-usd", s.MaxBudgetUSD,
		}, nil
	case AgentCodex:
		return []string{
			"exec",
			"-s", "read-only",
			"-C", s.Checkout,
			"-m", s.Model,
			"--json",
			codexCombinedPrompt(s.SystemPrompt, s.Prompt),
		}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownAgent, s.Agent)
	}
}

// codexCombinedPrompt folds the system prompt into the user prompt, since codex
// exec has no separate system-prompt flag (mirrors codexcli).
func codexCombinedPrompt(system, user string) string {
	if strings.TrimSpace(system) == "" {
		return user
	}
	return system + "\n\n---\n\n" + user
}
