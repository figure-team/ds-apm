package llmaigenerator

import (
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// goldenPaymentRequest is a fixed AIStrategyRequest used for golden-output
// tests. All fields are stable so the rendered prompt never drifts.
var goldenPaymentRequest = ruletypes.AIStrategyRequest{
	IncidentID:       "INC-PAY-001",
	AlertFingerprint: "fp-pay-abc",
	Labels: map[string]string{
		"alertname":    "PaymentAPI5xxHigh",
		"service.name": "paymentservice",
		"severity":     "critical",
	},
	Annotations: map[string]string{
		"summary": "Payment API 5xx rate is high",
	},
	SOPDocument: ruletypes.SOPDocument{
		SOPID:        "SOP-PAY-001",
		Version:      "2026-05-20.1",
		Title:        "Payment API 5xx 대응 절차",
		BodyMarkdown: "## 1단계\n- PG timeout 로그 확인\n- 결제 성공률 dashboard 확인",
	},
	EvidenceRefs: []ruletypes.AIEvidenceRef{
		{
			RefID:       "metric:error_rate:payment-api",
			Type:        "metric",
			Observation: "5xx rate 15% for 5 minutes",
			Confidence:  "high",
		},
	},
}

const expectedSystem = `당신은 SigNoz 알람의 1차 분석 AI 입니다. 입력으로 SOP 문서 본문, 알람 라벨/어노테이션, evidence 리스트가 주어집니다.
응답은 반드시 단일 JSON 객체로만 합니다. 추가 설명, 마크다운, 코드 펜스 금지.
스키마 (ruletypes.AIStrategy): { "headline": string, "hypotheses": [{"rank":int,"text":string,"confidence":"low|medium|high","evidenceRefs":[string],"sopStepRefs":[string]}], "firstActions": [{"text":string,"sopStepRef":string,"evidenceRefs":[string],"requiresHumanApproval":bool}], "notificationBody": string, "customerUpdateDraft": string, "vendorRequestDraft": string, "confidence":"low|medium|high", "limitations":[string] }
notificationBody는 운영자가 알림에서 바로 읽을 SOP 본문 기반 내용입니다. 첫 줄에 현재 알람 상황을 1줄로 요약하고("**현황:** ..." 형식), 빈 줄 뒤에는 SOP 문서 본문의 섹션 헤딩(##)과 그 순서를 그대로 보존하여 각 섹션 내용을 알람 맥락에 맞게 작성합니다. SOP에 없는 섹션을 새로 만들거나 SOP 헤딩 이름을 바꾸지 말고, SOP 본문의 '(작성)'·'(예)' 같은 자리표시자는 알람 근거로 구체화합니다. 이는 고객용 공지(customerUpdateDraft)와 별개이며, 장애 원인 단정·자동 조치 단정·확정 복구 시각(ETA) 약속은 금지입니다.
입력에 customerUpdateTemplate가 주어지면 customerUpdateDraft는 그 템플릿의 문구·구조·항목 순서를 그대로 유지하고 {중괄호} 슬롯만 인시던트 정보로 채웁니다(채울 근거가 없는 슬롯은 "확인 중"). 템플릿이 없으면 customerUpdateDraft는 공지문 형식으로 직접 작성합니다: 줄글(문단)이 아니라 첫 줄에 대괄호 제목(예: [결제 서비스 이용 장애 안내]), 빈 줄, 그 아래 각 항목을 "■ 라벨: 내용" 형태로 줄바꿈(\n)으로 구분해 나열하고 필수 항목 5개(■ 발생 현황, ■ 영향 범위, ■ 조치 사항, ■ 향후 안내, ■ 문의처)를 포함합니다. 어느 경우든 한국어 존댓말이며, 장애 원인 단정·배상/보상/법적 책임 언급·확정적 복구 시각(ETA) 약속은 금지입니다.
입력에 vendorRequestTemplate가 주어지면 vendorRequestDraft도 그 템플릿의 슬롯만 채웁니다. 템플릿이 없으면 공급자/벤더에게 보낼 확인 요청 초안을 직접 작성합니다. 자동 조치를 했다고 단정하지 마십시오.
firstActions의 모든 항목은 사람의 승인 후에만 실행되어야 하므로 requiresHumanApproval을 반드시 true로 설정하십시오.
각 hypothesis와 각 firstAction은 evidenceRefs 또는 sopStepRefs(firstAction은 sopStepRef) 중 최소 하나를 반드시 인용해야 합니다. 입력에 주어진 evidence의 refId와 SOP 단계만 인용하고, 없는 근거를 지어내지 마십시오.
SOP 문서와 충분한 근거가 있을 때만 confidence를 medium/high로 두고, 그 경우 evidenceRefs를 최소 하나 채웁니다. 근거가 부족하면 hypotheses와 firstActions를 비우고 confidence를 low로, limitations에 이유를 적습니다.
응답 언어는 한국어. 청구되지 않은 필드는 비웁니다.`

const expectedUser = `# SOP
- id: SOP-PAY-001
- version: 2026-05-20.1
- title: Payment API 5xx 대응 절차
- body:
## 1단계
- PG timeout 로그 확인
- 결제 성공률 dashboard 확인

# Alert
- incident_id: INC-PAY-001
- alert_fingerprint: fp-pay-abc
- labels:
alertname=PaymentAPI5xxHigh
service.name=paymentservice
severity=critical
- annotations:
summary=Payment API 5xx rate is high

# Evidence
- [metric] metric:error_rate:payment-api: 5xx rate 15% for 5 minutes (confidence=high)
`

func TestRender_GoldenPayment(t *testing.T) {
	sys, user := Render(goldenPaymentRequest)
	require.Equal(t, expectedSystem, sys)
	require.Equal(t, expectedUser, user)
}

// TestRender_SchemaRequestsCommunicationDrafts pins that the system prompt
// instructs the model to emit the customer/vendor communication drafts. Without
// this the model never produces them and the structured-output enhancement is
// inert.
func TestRender_SchemaRequestsCommunicationDrafts(t *testing.T) {
	sys, _ := Render(goldenPaymentRequest)
	require.Contains(t, sys, "customerUpdateDraft",
		"system prompt must request the customer update draft")
	require.Contains(t, sys, "vendorRequestDraft",
		"system prompt must request the vendor request draft")
}

// TestRender_NotificationBodyIsHybrid pins the hybrid notificationBody format:
// a one-line situation summary on top, then the SOP body's own section headings
// preserved (rather than a fixed 현재상황/점검/첫조치 rewrite).
func TestRender_NotificationBodyIsHybrid(t *testing.T) {
	sys, _ := Render(goldenPaymentRequest)
	require.Contains(t, sys, "1줄로 요약",
		"prompt must instruct a one-line situation summary on top")
	require.Contains(t, sys, "섹션 헤딩",
		"prompt must instruct preserving the SOP body's section headings")
}

// TestRender_IncludesCommsTemplatesWhenPresent pins CF-2 comms grounding: when
// the bound SOP carries org-approved comms templates, the user prompt surfaces
// them so the model fills their slots instead of free-writing the drafts. Absent
// templates must not emit the lines.
func TestRender_IncludesCommsTemplatesWhenPresent(t *testing.T) {
	req := goldenPaymentRequest
	req.SOPDocument.CustomerUpdateTemplate = "[결제 안내]\n■ 발생 현황: {상황}"
	req.SOPDocument.VendorRequestTemplate = "PG사 확인 요청: {증상}"
	_, withTpl := Render(req)
	require.Contains(t, withTpl, "customerUpdateTemplate:")
	require.Contains(t, withTpl, "[결제 안내]")
	require.Contains(t, withTpl, "vendorRequestTemplate:")

	_, noTpl := Render(goldenPaymentRequest)
	require.NotContains(t, noTpl, "customerUpdateTemplate:")
	require.NotContains(t, noTpl, "vendorRequestTemplate:")
}

// TestRender_IncludesPriorIncidents pins task #3 consumption: when the request
// carries past occurrences of the same failure, the user prompt surfaces them
// so the model can reference recurrence history.
func TestRender_IncludesPriorIncidents(t *testing.T) {
	req := goldenPaymentRequest
	req.PriorIncidents = []ruletypes.AIPriorIncident{
		{
			IncidentID:  "INC-PAY-OLD-2",
			GeneratedAt: "2026-05-18T08:00:00Z",
			Status:      "ready",
			Confidence:  "high",
			Headline:    "PG timeout으로 결제 5xx 급증 (지난주 재발)",
		},
		{
			IncidentID:  "INC-PAY-OLD-1",
			GeneratedAt: "2026-05-10T08:00:00Z",
			Status:      "ready",
			Confidence:  "medium",
			Headline:    "결제 승인 지연",
		},
	}

	_, user := Render(req)
	require.Contains(t, user, "INC-PAY-OLD-2")
	require.Contains(t, user, "PG timeout으로 결제 5xx 급증 (지난주 재발)")
	require.Contains(t, user, "INC-PAY-OLD-1")
	// Ordering preserved (caller supplies most-recent first).
	require.Less(t, strings.Index(user, "INC-PAY-OLD-2"), strings.Index(user, "INC-PAY-OLD-1"),
		"prior incidents must render in the order supplied")
}

func TestRender_EmptyEvidence(t *testing.T) {
	req := goldenPaymentRequest
	req.EvidenceRefs = nil
	_, user := Render(req)
	lines := strings.Split(user, "\n")
	foundHeader := false
	for i, line := range lines {
		if line == "# Evidence" {
			foundHeader = true
			// Next non-empty line must be (none)
			for _, next := range lines[i+1:] {
				require.Equal(t, "(none)", next, "expected (none) under # Evidence")
				break
			}
			break
		}
	}
	require.True(t, foundHeader, "expected # Evidence header in user message")
}

func TestRender_LabelsAreSorted(t *testing.T) {
	req := ruletypes.AIStrategyRequest{
		IncidentID: "INC-1",
		Labels: map[string]string{
			"b": "2",
			"a": "1",
		},
	}
	_, user := Render(req)
	idxA := strings.Index(user, "a=1")
	idxB := strings.Index(user, "b=2")
	require.True(t, idxA >= 0, "label a=1 not found in user message")
	require.True(t, idxB >= 0, "label b=2 not found in user message")
	require.Less(t, idxA, idxB, "label a=1 must appear before b=2")
}
