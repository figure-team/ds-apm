// Package llmaigenerator implements ruletypes.AIStrategyGenerator by calling
// an LLM provider. The package is provider-agnostic: concrete implementations
// of Provider (e.g. Claude API, Codex CLI) live in sub-packages and are wired
// in by the factory (pkg/ruler/aigenerator).
package llmaigenerator

import "context"

// Provider talks to the underlying LLM (HTTP API or local CLI).
// Implementations must respect ctx cancellation/timeout.
type Provider interface {
	// Complete returns the raw text body produced by the LLM in response to
	// the given system + user messages. Implementations should NOT parse
	// JSON — that's the Generator's job.
	Complete(ctx context.Context, system, user string) (string, error)
}
