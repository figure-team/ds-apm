package ruletypes

import "context"

// AIStrategyGenerator builds an AIStrategy from an AIStrategyRequest. The
// interface is intentionally narrow so it can be implemented by either the
// deterministic local generator (current default), a JSON-driven mock used
// for demos, or — eventually — an LLM-backed caller. Implementations must
// honor the request context (callers typically pass a short timeout).
type AIStrategyGenerator interface {
	Generate(ctx context.Context, req AIStrategyRequest) (AIStrategy, error)
}
