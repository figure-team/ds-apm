package llmaigenerator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// systemMessage is the fixed Korean system prompt sent to the LLM.
const systemMessage = `당신은 SigNoz 알람의 1차 분석 AI 입니다. 입력으로 SOP 문서 본문, 알람 라벨/어노테이션, evidence 리스트가 주어집니다.
응답은 반드시 단일 JSON 객체로만 합니다. 추가 설명, 마크다운, 코드 펜스 금지.
스키마 (ruletypes.AIStrategy): { "headline": string, "hypotheses": [{"rank":int,"text":string,"confidence":"low|medium|high","evidenceRefs":[string]}], "firstActions": [{"text":string,"sopStepRef":string,"requiresHumanApproval":bool}], "confidence":"low|medium|high", "limitations":[string] }
응답 언어는 한국어. 청구되지 않은 필드는 비웁니다.`

// Render builds the system and user messages for the LLM from req.
// It is deterministic: label and annotation keys are sorted alphabetically
// so golden tests remain stable.
func Render(req ruletypes.AIStrategyRequest) (system, user string) {
	system = systemMessage
	user = renderUser(req)
	return system, user
}

func renderUser(req ruletypes.AIStrategyRequest) string {
	var sb strings.Builder

	// SOP section
	sopID := req.SOPDocument.SOPID
	if strings.TrimSpace(sopID) == "" {
		sopID = "(none)"
	}
	fmt.Fprintf(&sb, "# SOP\n")
	fmt.Fprintf(&sb, "- id: %s\n", sopID)
	fmt.Fprintf(&sb, "- version: %s\n", req.SOPDocument.Version)
	fmt.Fprintf(&sb, "- title: %s\n", req.SOPDocument.Title)
	fmt.Fprintf(&sb, "- body:\n%s\n", req.SOPDocument.BodyMarkdown)

	// Alert section
	fmt.Fprintf(&sb, "\n# Alert\n")
	fmt.Fprintf(&sb, "- incident_id: %s\n", req.IncidentID)
	fmt.Fprintf(&sb, "- alert_fingerprint: %s\n", req.AlertFingerprint)

	fmt.Fprintf(&sb, "- labels:\n")
	if len(req.Labels) == 0 {
		fmt.Fprintf(&sb, "(none)\n")
	} else {
		keys := sortedKeys(req.Labels)
		for _, k := range keys {
			fmt.Fprintf(&sb, "%s=%s\n", k, req.Labels[k])
		}
	}

	fmt.Fprintf(&sb, "- annotations:\n")
	if len(req.Annotations) == 0 {
		fmt.Fprintf(&sb, "(none)\n")
	} else {
		keys := sortedKeys(req.Annotations)
		for _, k := range keys {
			fmt.Fprintf(&sb, "%s=%s\n", k, req.Annotations[k])
		}
	}

	// Evidence section
	fmt.Fprintf(&sb, "\n# Evidence\n")
	if len(req.EvidenceRefs) == 0 {
		fmt.Fprintf(&sb, "(none)\n")
	} else {
		for _, ref := range req.EvidenceRefs {
			fmt.Fprintf(&sb, "- [%s] %s: %s (confidence=%s)\n", ref.Type, ref.RefID, ref.Observation, ref.Confidence)
		}
	}

	return sb.String()
}

// sortedKeys returns the keys of m sorted alphabetically.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
