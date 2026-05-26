// Package localaigenerator wraps the deterministic local strategy builder so
// it satisfies the AIStrategyGenerator interface. This is the placeholder
// for the eventual LLM-backed generator: when the LLM client is added, it
// replaces this implementation behind the same interface.
package localaigenerator

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Generator is a zero-field struct that delegates to the deterministic free
// function GenerateLocalAIStrategy. The context is discarded because the
// underlying call is synchronous and does not perform any I/O.
type Generator struct{}

// New returns a new Generator.
func New() *Generator { return &Generator{} }

// Generate builds an AIStrategy from req using the deterministic local logic.
func (g *Generator) Generate(_ context.Context, req ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	return ruletypes.GenerateLocalAIStrategy(req)
}
