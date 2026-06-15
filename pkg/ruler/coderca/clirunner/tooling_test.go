package clirunner

import (
	"strings"
	"testing"
)

func TestToolingForCodexAllowsReadOnlyShell(t *testing.T) {
	d := strings.ToLower(ToolingFor(AgentCodex).ReadOnlyDirective())
	if !strings.Contains(d, "shell") {
		t.Errorf("codex directive must permit read-only shell inspection; got %q", d)
	}
	if strings.Contains(d, "do not run shell") {
		t.Errorf("codex directive must NOT forbid the shell it reads with; got %q", d)
	}
	// Read-only safety must still be stated.
	for _, must := range []string{"do not modify", "never run a command that writes"} {
		if !strings.Contains(d, must) {
			t.Errorf("codex directive missing read-only guard %q; got %q", must, d)
		}
	}
}

func TestToolingForClaudeForbidsShell(t *testing.T) {
	d := strings.ToLower(ToolingFor(AgentClaude).ReadOnlyDirective())
	if !strings.Contains(d, "do not run shell commands") {
		t.Errorf("claude directive must forbid shell (it reads via Read/Grep/Glob); got %q", d)
	}
	if !strings.Contains(d, "read, grep, glob") {
		t.Errorf("claude directive should name its read-only tools; got %q", d)
	}
}

func TestToolingForUnknownAgentFallsBackToClaude(t *testing.T) {
	if ToolingFor(Agent("unknown")).ReadOnlyDirective() != ToolingFor(AgentClaude).ReadOnlyDirective() {
		t.Error("unknown agent must fall back to the most restrictive (claude) profile")
	}
}
