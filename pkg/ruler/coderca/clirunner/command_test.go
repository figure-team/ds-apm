package clirunner

import (
	"errors"
	"strings"
	"testing"
)

// hasFlagValue reports whether args contains flag immediately followed by val.
func hasFlagValue(args []string, flag, val string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}

func contains(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func TestBuildArgsClaudeIsReadOnly(t *testing.T) {
	args, err := BuildArgs(Spec{
		Agent:        AgentClaude,
		Model:        "claude-opus-4-8",
		Checkout:     "/work/co/run1",
		SystemPrompt: "you are a code RCA agent",
		Prompt:       "find the root cause",
		MaxBudgetUSD: "0.50",
	})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}

	checks := []struct{ flag, val string }{
		{"-p", "find the root cause"},
		{"--append-system-prompt", "you are a code RCA agent"},
		{"--model", "claude-opus-4-8"},
		{"--add-dir", "/work/co/run1"},
		{"--permission-mode", "default"},
		{"--allowed-tools", "Read,Grep,Glob"},
		{"--disallowed-tools", "Bash,Write,Edit,WebFetch,WebSearch"},
		{"--max-budget-usd", "0.50"},
	}
	for _, c := range checks {
		if !hasFlagValue(args, c.flag, c.val) {
			t.Errorf("claude args missing %s %q; got %v", c.flag, c.val, args)
		}
	}
	// Read-only must NOT be bypassed.
	if contains(args, "--dangerously-skip-permissions") {
		t.Error("claude invocation must never skip permissions")
	}
	// MaxTurns unset (0) → no --max-turns flag.
	if contains(args, "--max-turns") {
		t.Errorf("claude args must omit --max-turns when MaxTurns<=0; got %v", args)
	}
}

func TestBuildArgsClaudeMaxTurns(t *testing.T) {
	args, err := BuildArgs(Spec{
		Agent:        AgentClaude,
		Model:        "claude-sonnet-4-6",
		Checkout:     "/work/co/run1",
		Prompt:       "find the root cause",
		MaxBudgetUSD: "2.00",
		MaxTurns:     40,
	})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if !hasFlagValue(args, "--max-turns", "40") {
		t.Errorf("claude args missing --max-turns 40; got %v", args)
	}
}

func TestBuildArgsCodexIsReadOnly(t *testing.T) {
	args, err := BuildArgs(Spec{
		Agent:        AgentCodex,
		Model:        "gpt-5",
		Checkout:     "/work/co/run2",
		SystemPrompt: "system rules",
		Prompt:       "why are we 5xx",
	})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if !contains(args, "exec") {
		t.Errorf("codex args missing exec subcommand: %v", args)
	}
	if !hasFlagValue(args, "-s", "read-only") {
		t.Errorf("codex args missing OS read-only sandbox: %v", args)
	}
	if !hasFlagValue(args, "-C", "/work/co/run2") {
		t.Errorf("codex args missing -C checkout scope: %v", args)
	}
	if !hasFlagValue(args, "-m", "gpt-5") {
		t.Errorf("codex args missing model: %v", args)
	}
	if !contains(args, "--json") {
		t.Errorf("codex args missing --json: %v", args)
	}
	if len(args) == 0 {
		t.Fatal("codex args empty")
	}
	// The combined prompt is the trailing positional and carries both system
	// and user text (codex has no separate system-prompt flag).
	last := args[len(args)-1]
	if !strings.Contains(last, "system rules") || !strings.Contains(last, "why are we 5xx") {
		t.Errorf("codex combined prompt missing system/user text: %q", last)
	}
}

func TestBuildArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		wantErr error
	}{
		{
			name:    "claude without checkout is rejected",
			spec:    Spec{Agent: AgentClaude, Model: "m", MaxBudgetUSD: "1"},
			wantErr: ErrNoCheckout,
		},
		{
			name:    "claude without budget ceiling is rejected",
			spec:    Spec{Agent: AgentClaude, Model: "m", Checkout: "/c"},
			wantErr: ErrNoBudgetCap,
		},
		{
			name:    "missing model is rejected",
			spec:    Spec{Agent: AgentCodex, Model: "", Checkout: "/c"},
			wantErr: ErrNoModel,
		},
		{
			name:    "unknown agent is rejected",
			spec:    Spec{Agent: "bogus", Model: "m", Checkout: "/c"},
			wantErr: ErrUnknownAgent,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildArgs(tc.spec)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("BuildArgs err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
