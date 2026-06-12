package llmaigenerator

import (
	"context"
	"fmt"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

const (
	// PromptVersion identifies the prompt template used by this package.
	// v2 adds customer/vendor communication drafts and explicit SOP-step /
	// evidence grounding refs to the requested output schema.
	PromptVersion = "ds-ir-ko-llm-v2"

	// DefaultTimeout is applied when New is called with timeout <= 0.
	DefaultTimeout = 15 * time.Second
)

// Generator calls a Provider to obtain LLM output and parses it into an
// AIStrategy. It satisfies ruletypes.AIStrategyGenerator.
type Generator struct {
	provider Provider
	model    string        // recorded in AIStrategyAudit.Model
	timeout  time.Duration // applied as context.WithTimeout per Generate call
}

// New returns a new Generator. If timeout <= 0, DefaultTimeout is used.
func New(provider Provider, model string, timeout time.Duration) *Generator {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Generator{provider: provider, model: model, timeout: timeout}
}

// Generate builds an AIStrategy for req by rendering the prompt, calling the
// Provider, and parsing the raw response. A per-call timeout derived from
// g.timeout is layered on top of the caller-supplied ctx.
func (g *Generator) Generate(ctx context.Context, req ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	if g.provider == nil {
		return ruletypes.AIStrategy{}, fmt.Errorf("llmaigenerator: provider must not be nil")
	}

	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	system, user := Render(req)

	raw, err := g.provider.Complete(ctx, system, user)
	if err != nil {
		return ruletypes.AIStrategy{}, fmt.Errorf("llmaigenerator: provider complete: %w", err)
	}

	return Parse(raw, req, g.model)
}

// compile-time assertion: Generator must satisfy AIStrategyGenerator.
var _ ruletypes.AIStrategyGenerator = (*Generator)(nil)
