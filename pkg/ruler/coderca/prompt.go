package coderca

import (
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// PromptVersion identifies the coderca-owned prompt contract for audit.
const PromptVersion = "coderca.rca.v1"

// BuildPrompt returns the (system, user) prompt pair for a code-RCA run
// (design §7). The system prompt is the coderca-owned, read-only instruction:
// explore the checkout, hypothesize a root cause, propose a fix as a SUGGESTION
// only (HITL — never applied), state confidence + limitations, echo the
// analyzed baseline commit, and emit a single fenced ```json block whose keys
// match ParseRCAResult. The user prompt carries the error context (and any
// evidence). Pure + deterministic.
//
// PART-B STUB: returns empty strings → prompt-content assertions fail (RED).
func BuildPrompt(rc RCAContext, evidence []ruletypes.AIEvidenceRef) (system string, user string) {
	return "", ""
}
