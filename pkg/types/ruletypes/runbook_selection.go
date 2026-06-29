package ruletypes

import (
	"errors"
	"fmt"
	"strings"
)

const (
	RunbookSelectionContractVersion = "ds.runbook_selection.v1"

	RunbookSelectionOutcomeSelected = "selected"
	RunbookSelectionOutcomeFallback = "fallback"
	RunbookSelectionOutcomeNone     = "none"
)

// RunbookSelectionDecision is the LLM's runtime decision about which approved
// Runbook fits the current incident, or — when none fit — a directly proposed
// fallback bash script. It is NOT persisted as-is: an actionable decision is
// reduced to a RemediationExecution; the full decision is recorded in cliaudit.
type RunbookSelectionDecision struct {
	ContractVersion string   `json:"contractVersion"`
	Outcome         string   `json:"outcome"`         // selected | fallback | none
	ChosenRunbookID string   `json:"chosenRunbookId"` // Outcome=selected only
	Confidence      string   `json:"confidence"`      // low | medium | high
	Rationale       string   `json:"rationale"`
	FallbackScript  string   `json:"fallbackScript"`  // Outcome=fallback only
	FallbackSummary string   `json:"fallbackSummary"`
	Limitations     []string `json:"limitations,omitempty"`
}

var allowedRunbookSelectionOutcomes = map[string]struct{}{
	RunbookSelectionOutcomeSelected: {},
	RunbookSelectionOutcomeFallback: {},
	RunbookSelectionOutcomeNone:     {},
}

// IsActionable reports whether the decision yields a RemediationExecution:
// a selected outcome (with a chosen id) or a fallback (with a script).
func (d RunbookSelectionDecision) IsActionable() bool {
	switch d.Outcome {
	case RunbookSelectionOutcomeSelected:
		return strings.TrimSpace(d.ChosenRunbookID) != ""
	case RunbookSelectionOutcomeFallback:
		return strings.TrimSpace(d.FallbackScript) != ""
	default:
		return false
	}
}

// ValidateRunbookSelectionDecision checks the decision shape and returns a
// possibly-normalized copy. A `selected` outcome whose ChosenRunbookID is not
// in approvedRunbookIDs is demoted to `none` (CF-11 dangling-id lesson: never
// act on an unknown id). Hard violations (bad outcome/confidence, oversized or
// NUL-containing fallback script) return an error and the caller must treat the
// decision as non-actionable (fail-open).
func ValidateRunbookSelectionDecision(d RunbookSelectionDecision, approvedRunbookIDs map[string]struct{}) (RunbookSelectionDecision, error) {
	var errs []error

	pilotRequireAllowed(&errs, "outcome", d.Outcome, allowedRunbookSelectionOutcomes)
	pilotRequireAllowed(&errs, "confidence", d.Confidence, allowedAIConfidenceValues)
	if len(errs) > 0 {
		return d, errors.Join(errs...)
	}

	switch d.Outcome {
	case RunbookSelectionOutcomeSelected:
		id := strings.TrimSpace(d.ChosenRunbookID)
		if id == "" {
			return d, fmt.Errorf("chosenRunbookId: required when outcome=selected")
		}
		if _, ok := approvedRunbookIDs[id]; !ok {
			// Demote: act on nothing rather than an unknown id.
			d.Outcome = RunbookSelectionOutcomeNone
			d.ChosenRunbookID = ""
			d.FallbackScript = ""
			d.Rationale = ""
			d.FallbackSummary = ""
		}
	case RunbookSelectionOutcomeFallback:
		s := d.FallbackScript
		if strings.TrimSpace(s) == "" {
			return d, fmt.Errorf("fallbackScript: required when outcome=fallback")
		}
		if len(s) > RunbookMaxScriptLen {
			return d, fmt.Errorf("fallbackScript: exceeds %d-byte limit (got %d)", RunbookMaxScriptLen, len(s))
		}
		if strings.ContainsRune(s, 0) {
			return d, fmt.Errorf("fallbackScript: must not contain NUL byte")
		}
	}

	pilotAppendSecretLikeStringErrors(&errs, "rationale", d.Rationale)
	pilotAppendSecretLikeStringErrors(&errs, "fallbackSummary", d.FallbackSummary)
	return d, errors.Join(errs...)
}
