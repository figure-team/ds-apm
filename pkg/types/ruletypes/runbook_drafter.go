package ruletypes

import "context"

// RunbookDraftRequest is the input for a single drafter call. ErrorExamples
// is bounded to 3 entries upstream so we don't pass enormous prompts to the LLM.
type RunbookDraftRequest struct {
	OrgID         string // owning org; lets a store-aware drafter resolve per-org AI creds
	SOP           SOPDocument
	ErrorExamples []string
	Source        string // "manual-paste" (v0.1) | "ai-strategy-history" (v0.1.5) | ...
}

// RunbookDrafter produces a Runbook draft from observed error examples and
// the parent SOP's context. Implementations:
//   - mockrunbookdrafter.Mock  — fixed response for tests
//   - llmrunbookdrafter.LLM    — wraps llmaigenerator.Provider
type RunbookDrafter interface {
	Draft(ctx context.Context, req RunbookDraftRequest) (Runbook, error)
}
