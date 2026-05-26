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
스키마 (ruletypes.AIStrategy): { "headline": string, "hypotheses": [{"rank":int,"text":string,"confidence":"low|medium|high","evidenceRefs":[string]}], "firstActions": [{"text":string,"sopStepRef":string,"requiresHumanApproval":bool}], "confidence":"low|medium|high", "limitations":[string] }
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
