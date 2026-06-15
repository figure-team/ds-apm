package clirunner

import "github.com/SigNoz/signoz/pkg/ruler/coderca"

// This file supplies the coderca.AgentTooling implementations for each agent.
// They live next to BuildArgs on purpose: the read-only directive an agent is
// TOLD in the prompt must match the read-only enforcement BuildArgs actually
// passes on the command line. Change one, change the other here.
//
//   - Claude (BuildArgs: --allowed-tools Read,Grep,Glob, --disallowed-tools
//     Bash,Write,Edit,...): reads through dedicated tools, never a shell.
//   - Codex (BuildArgs: -s read-only): the OS sandbox prevents all writes, so
//     read-only shell commands are how Codex inspects code. Forbidding the shell
//     (the Claude rule) would leave Codex with no way to read the checkout.

// claudeTooling instructs Claude to inspect via its read-only file tools only.
type claudeTooling struct{}

func (claudeTooling) ReadOnlyDirective() string {
	return "Inspect the code only through your read-only file tools (Read, Grep, Glob). " +
		"Do NOT modify, create, or delete any files, and do NOT run shell commands."
}

// codexTooling instructs Codex to inspect via read-only shell commands, which is
// how `codex exec` reads files; the -s read-only sandbox already blocks writes.
type codexTooling struct{}

func (codexTooling) ReadOnlyDirective() string {
	return "Inspect the code by running read-only shell commands (such as cat, ls, grep, sed -n). " +
		"Do NOT modify, create, or delete any files, do NOT access the network, and NEVER run a " +
		"command that writes, deletes, moves, or otherwise mutates state."
}

// ToolingFor returns the prompt tooling profile for an agent, kept in lockstep
// with the read-only flags BuildArgs emits. Unknown agents fall back to the
// most restrictive (Claude-style, no shell) profile.
func ToolingFor(a Agent) coderca.AgentTooling {
	switch a {
	case AgentCodex:
		return codexTooling{}
	default:
		return claudeTooling{}
	}
}

// compile-time assertions that both profiles satisfy the domain port.
var (
	_ coderca.AgentTooling = claudeTooling{}
	_ coderca.AgentTooling = codexTooling{}
)
