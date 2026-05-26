// Package llmrunbookdrafter implements ruletypes.RunbookDrafter on top of an
// existing llmaigenerator.Provider. Construction is intentionally cheap —
// callers can build a drafter per AIConfig and discard.
package llmrunbookdrafter

import (
	"context"
	"fmt"

	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// LLM drafts runbooks via a llmaigenerator.Provider. Immutable after New.
type LLM struct {
	provider llmaigenerator.Provider
	model    string
}

// New constructs an LLM drafter. model is recorded as Runbook.AIDraftedBy.
func New(provider llmaigenerator.Provider, model string) *LLM {
	return &LLM{provider: provider, model: model}
}

func (l *LLM) Draft(ctx context.Context, req ruletypes.RunbookDraftRequest) (ruletypes.Runbook, error) {
	if l.provider == nil {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: provider must not be nil")
	}
	system, user := renderPrompt(req)
	raw, err := l.provider.Complete(ctx, system, user)
	if err != nil {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: provider complete: %w", err)
	}
	return parseResponse(raw, req, l.model)
}

var _ ruletypes.RunbookDrafter = (*LLM)(nil)
