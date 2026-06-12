package coderca

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// PromptVersion identifies the coderca-owned prompt contract for audit.
const PromptVersion = "coderca.rca.v1"

// systemPrompt is the coderca-owned, read-only / HITL instruction. It is fixed
// (versioned by PromptVersion) and names the exact output keys ParseRCAResult
// consumes, so the agent's response stays machine-parseable.
const systemPrompt = `You are a code root-cause analysis agent. You have READ-ONLY access to a source
checkout pinned at a specific baseline commit. Do NOT modify, create, or delete
any files, and do NOT run shell commands. This is a human-in-the-loop workflow:
your proposed fix is a SUGGESTION only and is NEVER applied automatically.

Your task:
1. Explore the checkout to locate the code paths matching the reported error signature.
2. Hypothesize the most likely root cause.
3. Propose a fix as a suggestion (a diff sketch or concrete steps). Do not apply it.
4. State your confidence (high, medium, or low) and the limitations of this analysis.
5. Echo the baseline commit you analyzed.

Respond with a single fenced ` + "```json" + ` block and nothing after it, with exactly these keys:
{
  "baseline_commit": "<the commit you analyzed>",
  "root_cause": "<your root cause>",
  "proposed_fix": "<suggested fix; never applied>",
  "confidence": "high|medium|low",
  "limitations": "<limitations of this analysis>"
}`

// BuildPrompt returns the (system, user) prompt pair for a code-RCA run
// (design §7). The system prompt is fixed; the user prompt is assembled from the
// error context (labels/annotations sorted for determinism) plus any evidence.
func BuildPrompt(rc RCAContext, evidence []ruletypes.AIEvidenceRef) (system string, user string) {
	var b strings.Builder
	b.WriteString("# Error context\n")
	fmt.Fprintf(&b, "- Service: %s\n", rc.Service)
	fmt.Fprintf(&b, "- Severity: %s\n", rc.Severity)
	fmt.Fprintf(&b, "- Environment: %s\n", rc.Environment)
	fmt.Fprintf(&b, "- Alert fingerprint: %s\n", rc.Fingerprint)
	fmt.Fprintf(&b, "- Error signature: %s\n", rc.ErrorSignature)
	fmt.Fprintf(&b, "- Baseline commit (analyze this): %s\n", rc.BaselineCommit)

	if len(rc.Labels) > 0 {
		b.WriteString("\n## Labels\n")
		for _, k := range sortedKeys(rc.Labels) {
			fmt.Fprintf(&b, "- %s = %s\n", k, rc.Labels[k])
		}
	}
	if len(rc.Annotations) > 0 {
		b.WriteString("\n## Annotations\n")
		for _, k := range sortedKeys(rc.Annotations) {
			fmt.Fprintf(&b, "- %s = %s\n", k, rc.Annotations[k])
		}
	}
	if len(evidence) > 0 {
		b.WriteString("\n## Supporting evidence\n")
		for _, e := range evidence {
			fmt.Fprintf(&b, "- [%s/%s] %s\n", e.Type, e.RefID, e.Observation)
		}
	}

	return systemPrompt, b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
