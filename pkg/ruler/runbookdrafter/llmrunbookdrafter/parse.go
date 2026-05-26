package llmrunbookdrafter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// rawResponse mirrors the JSON contract documented in the spec's §4.3.
// "rationale" is read but not persisted in v0.1.
type rawResponse struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	ExecutableScript string   `json:"executableScript"`
	Confidence       *float64 `json:"confidence,omitempty"`
	Rationale        string   `json:"rationale,omitempty"`
}

// parseResponse decodes raw LLM output and assembles a Runbook with the
// server-controlled fields (id, status, timestamps, source examples, etc.)
// already populated.
func parseResponse(raw string, req ruletypes.RunbookDraftRequest, model string) (ruletypes.Runbook, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: parse: empty response")
	}

	var r rawResponse
	if err := json.Unmarshal([]byte(trimmed), &r); err != nil {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: parse: %w", err)
	}

	if strings.TrimSpace(r.Title) == "" {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: parse: title is required")
	}
	if strings.TrimSpace(r.Description) == "" {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: parse: description is required")
	}
	if strings.TrimSpace(r.ExecutableScript) == "" {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: parse: executableScript is required")
	}

	confidence := 0.5
	if r.Confidence != nil {
		confidence = *r.Confidence
		if confidence < 0 {
			confidence = 0
		}
		if confidence > 1 {
			confidence = 1
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	rb := ruletypes.Runbook{
		ID:                  uuid.NewString(),
		Title:               r.Title,
		Description:         r.Description,
		ExecutableScript:    r.ExecutableScript,
		Status:              ruletypes.RunbookStatusDraft,
		Confidence:          confidence,
		AIDraftedBy:         model,
		SourceErrorExamples: append([]string(nil), req.ErrorExamples...),
		CreatedAt:           now,
		UpdatedAt:           now,
		UpdatedBy:           "ai",
	}
	if err := ruletypes.ValidateRunbook(rb); err != nil {
		return ruletypes.Runbook{}, fmt.Errorf("llmrunbookdrafter: parse: produced invalid runbook: %w", err)
	}
	return rb, nil
}
