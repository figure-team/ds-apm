package remediation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// SelectionPromptVersion identifies the runbook-selection prompt template.
const SelectionPromptVersion = "ds-runbook-select-ko-v1"

// selectionScriptPreviewMax bounds how many bytes of each runbook/script the
// prompt carries, so a few large runbooks can't blow the token budget.
const selectionScriptPreviewMax = 2000

// SelectionInput is the full context for one runbook-selection decision.
type SelectionInput struct {
	OrgID            string
	IncidentID       string
	AlertFingerprint string
	Labels           map[string]string
	Annotations      map[string]string
	SOP              ruletypes.SOPDocument
	Runbooks         []ruletypes.Runbook // approved runbooks, registration order
	Evidence         []ruletypes.AIEvidenceRef
}

const selectionSystemMessage = `당신은 SOP 기반 자동대응 Runbook 선택 AI입니다. 입력으로 SOP 본문, 등록된 Runbook 목록, 알람 라벨/어노테이션, evidence가 주어집니다.
현재 알람 상황과 각 Runbook을 비교해 가장 적합한 하나를 고르거나, 적합한 것이 없으면 안전한 대응 스크립트를 직접 제안합니다.
응답은 반드시 단일 JSON 객체로만 합니다. 추가 설명, 마크다운, 코드 펜스 금지.
스키마: { "outcome": "selected|fallback|none", "chosenRunbookId": string, "confidence": "low|medium|high", "rationale": string, "fallbackScript": string, "fallbackSummary": string, "limitations": [string] }
- outcome=selected: 입력 Runbook 중 적합한 것이 있을 때. chosenRunbookId는 반드시 입력에 주어진 id 중 하나여야 하며, rationale에 선택 근거를 적습니다.
- outcome=fallback: 적합한 Runbook이 없을 때만. fallbackScript에 bash 스크립트를, fallbackSummary에 1줄 요약을 적습니다. 파괴적/되돌릴 수 없는 명령(rm -rf, 대량 삭제, DROP 등)과 자격증명을 stdout에 출력하는 행위를 금지합니다. 환경변수 참조는 허용합니다.
- outcome=none: Runbook도 부적합하고 안전한 스크립트도 제안할 수 없을 때. rationale에 이유를 적습니다.
confidence는 근거가 충분할 때만 medium/high로 둡니다. 근거가 부족하면 low로 두고 limitations에 이유를 적습니다.
응답 언어는 한국어. 실제로 실행되는 것은 사람이 검토·승인한 뒤이며, 당신의 제안이 곧바로 실행되지 않습니다.`

// RenderSelectionPrompt builds the deterministic system+user messages.
func RenderSelectionPrompt(in SelectionInput) (system, user string) {
	system = selectionSystemMessage
	var sb strings.Builder

	fmt.Fprintf(&sb, "# SOP\n")
	fmt.Fprintf(&sb, "- id: %s\n", firstNonEmptyStr(in.SOP.SOPID, "(none)"))
	fmt.Fprintf(&sb, "- version: %s\n", in.SOP.Version)
	fmt.Fprintf(&sb, "- title: %s\n", in.SOP.Title)
	fmt.Fprintf(&sb, "- body:\n%s\n", in.SOP.BodyMarkdown)

	fmt.Fprintf(&sb, "\n# Runbooks (등록 순)\n")
	for i, rb := range in.Runbooks {
		fmt.Fprintf(&sb, "## [%d] id=%s\n", i+1, rb.ID)
		fmt.Fprintf(&sb, "- title: %s\n", rb.Title)
		fmt.Fprintf(&sb, "- confidence: %.2f\n", rb.Confidence)
		if d := strings.TrimSpace(rb.Description); d != "" {
			fmt.Fprintf(&sb, "- description: %s\n", truncatePrompt(d, selectionScriptPreviewMax))
		}
		fmt.Fprintf(&sb, "- script:\n%s\n", truncatePrompt(rb.ExecutableScript, selectionScriptPreviewMax))
	}

	fmt.Fprintf(&sb, "\n# Alert\n")
	fmt.Fprintf(&sb, "- incident_id: %s\n", in.IncidentID)
	fmt.Fprintf(&sb, "- alert_fingerprint: %s\n", in.AlertFingerprint)
	fmt.Fprintf(&sb, "- labels:\n")
	writeSortedKV(&sb, in.Labels)
	fmt.Fprintf(&sb, "- annotations:\n")
	writeSortedKV(&sb, in.Annotations)

	fmt.Fprintf(&sb, "\n# Evidence\n")
	if len(in.Evidence) == 0 {
		fmt.Fprintf(&sb, "(none)\n")
	} else {
		for _, ref := range in.Evidence {
			fmt.Fprintf(&sb, "- [%s] %s: %s (confidence=%s)\n", ref.Type, ref.RefID, ref.Observation, ref.Confidence)
		}
	}

	return system, sb.String()
}

func writeSortedKV(sb *strings.Builder, m map[string]string) {
	if len(m) == 0 {
		fmt.Fprintf(sb, "(none)\n")
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(sb, "%s=%s\n", k, m[k])
	}
}

func truncatePrompt(s string, n int) string {
	if len(s) > n {
		return s[:n] + "...(truncated)"
	}
	return s
}

func firstNonEmptyStr(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
