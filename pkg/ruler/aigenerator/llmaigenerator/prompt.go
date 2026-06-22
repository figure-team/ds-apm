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
스키마 (ruletypes.AIStrategy): { "headline": string, "hypotheses": [{"rank":int,"text":string,"confidence":"low|medium|high","evidenceRefs":[string],"sopStepRefs":[string]}], "firstActions": [{"text":string,"sopStepRef":string,"evidenceRefs":[string],"requiresHumanApproval":bool}], "notificationBody": string, "customerUpdateDraft": string, "vendorRequestDraft": string, "confidence":"low|medium|high", "limitations":[string] }
notificationBody는 운영자가 알림에서 바로 읽을 SOP 본문 기반 상황 요약입니다. SOP 문서 본문을 근거로 현재 상황·핵심 점검 항목·첫 조치를 markdown(제목 ##, 항목 -)으로 간결히 작성합니다. 이는 고객용 공지(customerUpdateDraft)와 별개이며, 장애 원인 단정·자동 조치 단정·확정 복구 시각(ETA) 약속은 금지입니다.
입력에 customerUpdateTemplate가 주어지면 customerUpdateDraft는 그 템플릿의 문구·구조·항목 순서를 그대로 유지하고 {중괄호} 슬롯만 인시던트 정보로 채웁니다(채울 근거가 없는 슬롯은 "확인 중"). 템플릿이 없으면 customerUpdateDraft는 공지문 형식으로 직접 작성합니다: 줄글(문단)이 아니라 첫 줄에 대괄호 제목(예: [결제 서비스 이용 장애 안내]), 빈 줄, 그 아래 각 항목을 "■ 라벨: 내용" 형태로 줄바꿈(\n)으로 구분해 나열하고 필수 항목 5개(■ 발생 현황, ■ 영향 범위, ■ 조치 사항, ■ 향후 안내, ■ 문의처)를 포함합니다. 어느 경우든 한국어 존댓말이며, 장애 원인 단정·배상/보상/법적 책임 언급·확정적 복구 시각(ETA) 약속은 금지입니다.
입력에 vendorRequestTemplate가 주어지면 vendorRequestDraft도 그 템플릿의 슬롯만 채웁니다. 템플릿이 없으면 공급자/벤더에게 보낼 확인 요청 초안을 직접 작성합니다. 자동 조치를 했다고 단정하지 마십시오.
firstActions의 모든 항목은 사람의 승인 후에만 실행되어야 하므로 requiresHumanApproval을 반드시 true로 설정하십시오.
각 hypothesis와 각 firstAction은 evidenceRefs 또는 sopStepRefs(firstAction은 sopStepRef) 중 최소 하나를 반드시 인용해야 합니다. 입력에 주어진 evidence의 refId와 SOP 단계만 인용하고, 없는 근거를 지어내지 마십시오.
SOP 문서와 충분한 근거가 있을 때만 confidence를 medium/high로 두고, 그 경우 evidenceRefs를 최소 하나 채웁니다. 근거가 부족하면 hypotheses와 firstActions를 비우고 confidence를 low로, limitations에 이유를 적습니다.
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

	// Org-approved comms templates (optional). When present the generator must
	// fill these rather than free-writing the customer/vendor draft.
	if t := strings.TrimSpace(req.SOPDocument.CustomerUpdateTemplate); t != "" {
		fmt.Fprintf(&sb, "- customerUpdateTemplate:\n%s\n", t)
	}
	if t := strings.TrimSpace(req.SOPDocument.VendorRequestTemplate); t != "" {
		fmt.Fprintf(&sb, "- vendorRequestTemplate:\n%s\n", t)
	}

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

	// Prior incidents section — past occurrences of the same failure signature,
	// most recent first. Rendered only when present so the prompt stays compact
	// for first-time alerts. The model may reference these to spot recurrence.
	if len(req.PriorIncidents) > 0 {
		fmt.Fprintf(&sb, "\n# Prior Incidents (동일 장애 과거 이력)\n")
		for _, prior := range req.PriorIncidents {
			fmt.Fprintf(&sb, "- [%s] %s (status=%s, confidence=%s): %s\n",
				prior.GeneratedAt, prior.IncidentID, prior.Status, prior.Confidence, prior.Headline)
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
