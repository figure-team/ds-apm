package coderca

// AgentTooling is a domain port describing how a specific analysis agent
// inspects a source checkout READ-ONLY. Different CLI agents read code through
// different mechanisms (Claude exposes Read/Grep/Glob tools; Codex inspects via
// read-only shell commands), so the one instruction that must vary per agent —
// "how may you read the code, and what must you never do" — is abstracted here.
//
// The prompt builder (BuildPrompt) depends on this port and never references a
// concrete agent. Each agent adapter (see pkg/.../clirunner) supplies a concrete
// implementation, co-located with the CLI flags it emits, so the prompt's
// read-only directive and the adapter's actual sandbox/tool flags stay in
// lockstep. Mismatching the two is exactly what made Codex unable to read the
// checkout (the prompt forbade the shell it reads with).
type AgentTooling interface {
	// ReadOnlyDirective returns the system-prompt sentence(s) instructing the
	// agent how it may inspect the checkout and what it must never do
	// (modify/create/delete files, network access, etc.).
	ReadOnlyDirective() string
}
