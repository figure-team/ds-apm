package llmrunbookdrafter

import (
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// RunbookPromptVersion identifies the prompt template used by this package.
// Bump when the system prompt or the rendered user-prompt shape changes
// in a way that would affect model output.
const RunbookPromptVersion = "ds-runbook-ko-llm-v1"

const systemPrompt = `You are an SRE assistant. Given an SOP context and recent error examples, draft a runbook (with executable bash script) that an operator can run to mitigate or fix the issue. The script must be bash, idempotent, and safe to run multiple times. Output strict JSON only — no markdown fence, no preamble, no trailing text. Use exactly these fields: "title", "description", "executableScript", "confidence" (number 0.0-1.0), "rationale".`

func renderPrompt(req ruletypes.RunbookDraftRequest) (system, user string) {
	system = systemPrompt

	var b strings.Builder
	fmt.Fprintf(&b, "[SOP Context]\nTitle: %s\n", req.SOP.Title)
	if strings.TrimSpace(req.SOP.SOPID) != "" {
		fmt.Fprintf(&b, "SOP ID: %s\n", req.SOP.SOPID)
	}
	if strings.TrimSpace(req.SOP.BodyMarkdown) != "" {
		summary := req.SOP.BodyMarkdown
		if len(summary) > 2000 {
			summary = summary[:2000] + "...(truncated)"
		}
		fmt.Fprintf(&b, "Body (excerpt):\n%s\n", summary)
	}
	b.WriteString("\n[Recent error examples]\n")
	for i, ex := range req.ErrorExamples {
		fmt.Fprintf(&b, "%d. %s\n", i+1, ex)
	}
	b.WriteString("\n[Task]\nDraft a runbook for handling this error pattern. The script must be bash, idempotent, and safe to run multiple times.\n")

	user = b.String()
	return
}
