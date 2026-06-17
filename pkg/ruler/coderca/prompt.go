package coderca

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// PromptVersion identifies the coderca-owned prompt contract for audit.
// v2 adds a Korean readability style guide so root_cause/proposed_fix render as
// scannable, plain-language guidance for on-call operators.
const PromptVersion = "coderca.rca.v2"

// systemPromptIntro states the read-only / HITL contract that holds for every
// agent. The agent-specific "how to read the code" sentence is injected between
// this and systemPromptTask from an AgentTooling port, so the prompt never
// hardcodes one agent's tool model.
const systemPromptIntro = `You are a code root-cause analysis agent. You have READ-ONLY access to a source
checkout pinned at a specific baseline commit. This is a human-in-the-loop workflow:
your proposed fix is a SUGGESTION only and is NEVER applied automatically.`

// systemPromptTask is the fixed task + output contract. It names the exact JSON
// keys ParseRCAResult consumes, so the agent's response stays machine-parseable.
const systemPromptTask = `Your task:
1. Explore the checkout to locate the code paths matching the reported error signature.
2. Hypothesize the most likely root cause.
3. Propose a fix as a suggestion (a diff sketch or concrete steps). Do not apply it.
4. State your confidence (high, medium, or low) and the limitations of this analysis.
5. Echo the baseline commit you analyzed.

Write all human-readable analysis text — the values of "root_cause", "proposed_fix", and
"limitations" — in Korean (한국어) by default. Keep the JSON keys and the "confidence" value
(one of high, medium, low) in English exactly as specified below.

Readability matters as much as accuracy. The reader is a busy on-call operator who may not
know this codebase, reading on a small panel under pressure. Write so they grasp the problem
in seconds and know exactly what to do next. Follow these rules for the Korean text:
- 결론부터: 첫 문장은 원인(또는 조치)을 한 문장으로 요약한다. 배경 설명은 그 다음에.
- 짧게: 긴 문단 대신 짧은 문장과 불릿("- …")으로 쓴다. 실제 줄바꿈으로 항목을 나눈다(화면이 줄바꿈을 그대로 보여준다).
- 쉽게: 일상적인 표현을 쓴다. 전문 용어나 파일·함수명이 꼭 필요하면 괄호 안에 한 줄로 풀어 설명한다.
- "root_cause": 1줄 요약 → 핵심 근거 불릿 2~3개(어느 파일/함수에서 무엇이 잘못되는지). 그 이상은 줄인다.
- "proposed_fix": 운영자가 따라 할 수 있는 번호 매긴 구체적 단계("1. …", "2. …")로 쓴다. 막연한 방향 제시는 금지. 단, 이는 제안(suggestion)일 뿐 절대 자동 적용되지 않는다.
- "limitations": 확인하지 못한 점을 짧은 불릿 1~2개로만.
- 읽는 사람의 이해나 조치에 도움이 안 되는 내용은 모두 뺀다. 짧을수록 좋다.

Respond with a single fenced ` + "```json" + ` block and nothing after it, with exactly these keys:
{
  "baseline_commit": "<the commit you analyzed>",
  "root_cause": "<your root cause>",
  "proposed_fix": "<suggested fix; never applied>",
  "confidence": "high|medium|low",
  "limitations": "<limitations of this analysis>"
}`

// buildSystemPrompt assembles the full system prompt by sandwiching the agent's
// read-only inspection directive between the fixed intro and task sections.
func buildSystemPrompt(tooling AgentTooling) string {
	return systemPromptIntro + "\n" + tooling.ReadOnlyDirective() + "\n\n" + systemPromptTask
}

// BuildPrompt returns the (system, user) prompt pair for a code-RCA run
// (design §7). The system prompt's read-only inspection directive comes from the
// supplied AgentTooling port (so it matches the agent's actual CLI flags); the
// user prompt is assembled from the error context (labels/annotations sorted for
// determinism) plus any evidence.
func BuildPrompt(rc RCAContext, evidence []ruletypes.AIEvidenceRef, tooling AgentTooling) (system string, user string) {
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

	return buildSystemPrompt(tooling), b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
