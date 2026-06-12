package llmaigenerator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// partialResponse holds only the LLM-controlled fields extracted from the raw
// JSON response. Caller-controlled metadata (contract version, audit, IDs) is
// never taken from LLM output.
type partialResponse struct {
	Headline            string                    `json:"headline"`
	Hypotheses          []ruletypes.AIHypothesis  `json:"hypotheses"`
	FirstActions        []ruletypes.AIFirstAction `json:"firstActions"`
	CustomerUpdateDraft string                    `json:"customerUpdateDraft"`
	VendorRequestDraft  string                    `json:"vendorRequestDraft"`
	Confidence          string                    `json:"confidence"`
	Limitations         []string                  `json:"limitations"`
	Status              string                    `json:"status"`
}

// Parse extracts an AIStrategy from the raw LLM text output.
// It locates the first '{' and last '}' in raw, unmarshals the substring, and
// populates all caller-controlled fields from req and model.
func Parse(raw string, req ruletypes.AIStrategyRequest, model string) (ruletypes.AIStrategy, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return ruletypes.AIStrategy{}, fmt.Errorf("llmaigenerator: no JSON object found in LLM response")
	}

	jsonStr := raw[start : end+1]
	var partial partialResponse
	if err := json.Unmarshal([]byte(jsonStr), &partial); err != nil {
		return ruletypes.AIStrategy{}, fmt.Errorf("llmaigenerator: unmarshal LLM response: %w", err)
	}

	// Resolve status — default to ready if absent or unrecognised.
	status := partial.Status
	if _, ok := allowedStatuses[status]; !ok || status == "" {
		status = ruletypes.AIStrategyStatusReady
	}

	// Resolve confidence — default to low if absent or unrecognised.
	confidence := partial.Confidence
	if _, ok := allowedConfidences[confidence]; !ok || confidence == "" {
		confidence = ruletypes.AIConfidenceLow
	}

	strategy := ruletypes.AIStrategy{
		ContractVersion:     ruletypes.AIStrategyContractVersion,
		StrategyID:          deterministicLLMStrategyID(req),
		IncidentID:          req.IncidentID,
		AlertFingerprint:    req.AlertFingerprint,
		SOPID:               req.SOPDocument.SOPID,
		SOPVersion:          req.SOPDocument.Version,
		Language:            "ko-KR",
		Status:              status,
		Confidence:          confidence,
		Headline:            partial.Headline,
		Hypotheses:          partial.Hypotheses,
		FirstActions:        partial.FirstActions,
		CustomerUpdateDraft: partial.CustomerUpdateDraft,
		VendorRequestDraft:  partial.VendorRequestDraft,
		EvidenceRefs:        req.EvidenceRefs,
		Limitations:         partial.Limitations,
		Audit: ruletypes.AIStrategyAudit{
			PromptVersion:    PromptVersion,
			Model:            model,
			GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
			RedactionApplied: true, // TODO v0.2: implement actual redaction pass
		},
	}

	// SOP grounding enforcement: when an SOP document was injected into the
	// prompt, a ready draft must cite it via at least one SOP step reference.
	// A ready draft grounded on evidence refs alone is not SOP-grounded and is
	// downgraded so consumers are not misled into trusting a recommendation the
	// model never tied back to the bound SOP (hallucination suppression).
	if strategy.Status == ruletypes.AIStrategyStatusReady &&
		strings.TrimSpace(req.SOPDocument.SOPID) != "" &&
		!isSOPGrounded(strategy) {
		strategy.Status = ruletypes.AIStrategyStatusLowConfidence
		strategy.Confidence = ruletypes.AIConfidenceLow
		strategy.Limitations = append(strategy.Limitations, groundingMissingLimitation)
	}

	if err := ruletypes.ValidateAIStrategy(strategy); err != nil {
		return ruletypes.AIStrategy{}, err
	}

	return strategy, nil
}

// groundingMissingLimitation explains a draft that was downgraded because it
// did not cite the bound SOP.
const groundingMissingLimitation = "AI draft was not grounded in the bound SOP (no SOP step citation); confidence downgraded."

// isSOPGrounded reports whether the strategy cites the SOP at least once via a
// hypothesis sopStepRefs entry or a first-action sopStepRef.
func isSOPGrounded(strategy ruletypes.AIStrategy) bool {
	for _, h := range strategy.Hypotheses {
		for _, ref := range h.SOPStepRefs {
			if strings.TrimSpace(ref) != "" {
				return true
			}
		}
	}
	for _, a := range strategy.FirstActions {
		if strings.TrimSpace(a.SOPStepRef) != "" {
			return true
		}
	}
	return false
}

var allowedStatuses = map[string]struct{}{
	ruletypes.AIStrategyStatusReady:               {},
	ruletypes.AIStrategyStatusUnavailable:         {},
	ruletypes.AIStrategyStatusTimeout:             {},
	ruletypes.AIStrategyStatusBlockedByPolicy:     {},
	ruletypes.AIStrategyStatusQuotaExhausted:      {},
	ruletypes.AIStrategyStatusSOPMissing:          {},
	ruletypes.AIStrategyStatusEvidenceUnavailable: {},
	ruletypes.AIStrategyStatusLowConfidence:       {},
}

var allowedConfidences = map[string]struct{}{
	ruletypes.AIConfidenceHigh:   {},
	ruletypes.AIConfidenceMedium: {},
	ruletypes.AIConfidenceLow:    {},
}

// deterministicLLMStrategyID computes a collision-resistant StrategyID for LLM-generated
// strategies. Uses SHA-256 hash over IncidentID, AlertFingerprint, SOPID, and Version
// to ensure uniqueness across concurrent incidents with identical IncidentIDs (which can
// occur due to incident correlation). Matches the pattern used in ruletypes.deterministicAIStrategyID.
func deterministicLLMStrategyID(req ruletypes.AIStrategyRequest) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(req.IncidentID),
		strings.TrimSpace(req.AlertFingerprint),
		strings.TrimSpace(req.SOPDocument.SOPID),
		strings.TrimSpace(req.SOPDocument.Version),
	}, "|")))

	return "llm-" + hex.EncodeToString(sum[:])[:16]
}
