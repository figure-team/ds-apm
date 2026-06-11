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
스키마 (ruletypes.AIStrategy): { "headline": string, "hypotheses": [{"rank":int,"text":string,"confidence":"low|medium|high","evidenceRefs":[string],"sopStepRefs":[string]}], "firstActions": [{"text":string,"sopStepRef":string,"evidenceRefs":[string],"requiresHumanApproval":bool}], "customerUpdateDraft": string, "vendorRequestDraft": string, "confidence":"low|medium|high", "limitations":[string] }
customerUpdateDraft는 고객에게 보낼 상황 공유 초안, vendorRequestDraft는 공급자/벤더에게 보낼 확인 요청 초안입니다. 자동 조치를 했다고 단정하지 마십시오.
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
