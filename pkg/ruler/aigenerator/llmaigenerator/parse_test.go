package llmaigenerator

import (
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// parseReq is a minimal AIStrategyRequest that satisfies ValidateAIStrategy for
// a ready strategy: it supplies sopId, sopVersion, and at least one evidence ref.
var parseReq = ruletypes.AIStrategyRequest{
	IncidentID:       "INC-PARSE-001",
	AlertFingerprint: "fp-parse",
	SOPDocument: ruletypes.SOPDocument{
		SOPID:   "SOP-PARSE-001",
		Version: "2026-05-20.1",
	},
	EvidenceRefs: []ruletypes.AIEvidenceRef{
		{
			RefID:       "metric:err:svc",
			Type:        "metric",
			Observation: "error rate elevated",
			Confidence:  "medium",
		},
	},
}

// happyJSON is a complete ready-strategy JSON the LLM might return.
const happyJSON = `{
  "headline": "결제 API 오류율 급증 — PG timeout 가능성 우선 확인",
  "hypotheses": [
    {
      "rank": 1,
      "text": "외부 PG 응답 지연으로 5xx 증가",
      "confidence": "medium",
      "evidenceRefs": ["metric:err:svc"]
    }
  ],
  "firstActions": [
    {
      "text": "결제 성공률 dashboard와 PG timeout log를 확인",
      "sopStepRef": "SOP-PARSE-001#1",
      "requiresHumanApproval": true
    }
  ],
  "confidence": "medium",
  "status": "ready"
}`

func TestParse_HappyJSON(t *testing.T) {
	strategy, err := Parse(happyJSON, parseReq, "test-model-v1")
	require.NoError(t, err)
	require.Equal(t, "결제 API 오류율 급증 — PG timeout 가능성 우선 확인", strategy.Headline)
	require.Equal(t, ruletypes.AIStrategyStatusReady, strategy.Status)
	require.Equal(t, ruletypes.AIConfidenceMedium, strategy.Confidence)
	require.Len(t, strategy.Hypotheses, 1)
	require.Equal(t, 1, strategy.Hypotheses[0].Rank)
	require.Len(t, strategy.FirstActions, 1)
	require.True(t, strategy.FirstActions[0].RequiresHumanApproval)
	require.Equal(t, ruletypes.AIStrategyContractVersion, strategy.ContractVersion)
	require.NotEmpty(t, strategy.StrategyID, "StrategyID must be set")
	require.True(t, strings.HasPrefix(strategy.StrategyID, "llm-"), "StrategyID must start with llm- prefix")
	require.Equal(t, "INC-PARSE-001", strategy.IncidentID)
	require.Equal(t, "SOP-PARSE-001", strategy.SOPID)
	require.Equal(t, "2026-05-20.1", strategy.SOPVersion)
	require.Equal(t, "ko-KR", strategy.Language)
	require.NoError(t, ruletypes.ValidateAIStrategy(strategy))
}

func TestParse_StripsSurroundingProse(t *testing.T) {
	raw := `Sure, here's the response: {"headline":"x","confidence":"medium","status":"ready","hypotheses":[{"rank":1,"text":"가설","confidence":"medium","evidenceRefs":["metric:err:svc"]}],"firstActions":[{"text":"확인","sopStepRef":"SOP-PARSE-001#1","requiresHumanApproval":true}]} hope that helps`
	strategy, err := Parse(raw, parseReq, "model-x")
	require.NoError(t, err)
	require.Equal(t, "x", strategy.Headline)
	require.Equal(t, ruletypes.AIStrategyStatusReady, strategy.Status)
}

func TestParse_RejectsBrokenJSON(t *testing.T) {
	_, err := Parse("{not json", parseReq, "model-x")
	require.Error(t, err)
}

func TestParse_NoJSON(t *testing.T) {
	_, err := Parse("plain text", parseReq, "model-x")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "JSON") || strings.Contains(err.Error(), "json"),
		"error should mention JSON, got: %s", err.Error())
}

// TestParse_ExtractsCustomerAndVendorDrafts pins the output-structure
// requirement that the LLM-controlled customer/vendor communication drafts
// survive parsing. These fields exist on AIStrategy and are surfaced via
// AIStrategyIncidentAnnotations, but the LLM parse path previously dropped
// them.
func TestParse_ExtractsCustomerAndVendorDrafts(t *testing.T) {
	const raw = `{
	  "headline": "결제 API 오류율 급증 — PG timeout 가능성 우선 확인",
	  "hypotheses": [
	    {"rank":1,"text":"외부 PG 응답 지연으로 5xx 증가","confidence":"medium","evidenceRefs":["metric:err:svc"]}
	  ],
	  "firstActions": [
	    {"text":"결제 성공률 dashboard와 PG timeout log를 확인","sopStepRef":"SOP-PARSE-001#1","requiresHumanApproval":true}
	  ],
	  "customerUpdateDraft": "현재 결제 지연 알림을 확인하여 SOP 기준 초동 분석 중입니다. 다음 업데이트는 15분 내 공유하겠습니다.",
	  "vendorRequestDraft": "PG사 측 응답 지연 또는 장애 공지 여부를 확인 부탁드립니다.",
	  "confidence": "medium",
	  "status": "ready"
	}`

	strategy, err := Parse(raw, parseReq, "test-model-v1")
	require.NoError(t, err)
	require.Equal(t,
		"현재 결제 지연 알림을 확인하여 SOP 기준 초동 분석 중입니다. 다음 업데이트는 15분 내 공유하겠습니다.",
		strategy.CustomerUpdateDraft)
	require.Equal(t,
		"PG사 측 응답 지연 또는 장애 공지 여부를 확인 부탁드립니다.",
		strategy.VendorRequestDraft)
}

// TestDraft_GroundedCitation pins the SOP-grounding requirement: when an SOP
// document is injected into the request, a ready draft must cite the SOP via at
// least one sopStepRef/sopStepRefs. A draft grounded on evidence refs alone is
// not SOP-grounded and is downgraded so consumers are not misled into trusting
// an ungrounded (potentially hallucinated) recommendation.
func TestDraft_GroundedCitation(t *testing.T) {
	// Positive: the LLM cites the SOP step — the citation survives parsing and
	// the strategy stays ready.
	groundedJSON := `{
	  "headline": "결제 API 오류율 급증 — PG timeout 우선 확인",
	  "hypotheses": [
	    {"rank":1,"text":"외부 PG 응답 지연으로 5xx 증가","confidence":"medium","evidenceRefs":["metric:err:svc"],"sopStepRefs":["SOP-PARSE-001#1"]}
	  ],
	  "firstActions": [
	    {"text":"PG timeout log 확인","sopStepRef":"SOP-PARSE-001#1","requiresHumanApproval":true}
	  ],
	  "confidence": "medium",
	  "status": "ready"
	}`
	grounded, err := Parse(groundedJSON, parseReq, "test-model")
	require.NoError(t, err)
	require.Equal(t, ruletypes.AIStrategyStatusReady, grounded.Status)
	require.Equal(t, "SOP-PARSE-001#1", grounded.FirstActions[0].SOPStepRef,
		"SOP citation must survive parsing")

	// Negative: SOP is injected (parseReq.SOPDocument.SOPID is set) but the LLM
	// grounds everything on evidence refs only — zero SOP citations. The draft
	// is not SOP-grounded, so it must be downgraded and flagged.
	ungroundedJSON := `{
	  "headline": "결제 API 오류율 급증",
	  "hypotheses": [
	    {"rank":1,"text":"외부 PG 응답 지연으로 5xx 증가","confidence":"medium","evidenceRefs":["metric:err:svc"]}
	  ],
	  "firstActions": [
	    {"text":"PG timeout log 확인","evidenceRefs":["metric:err:svc"],"requiresHumanApproval":true}
	  ],
	  "confidence": "medium",
	  "status": "ready"
	}`
	ungrounded, err := Parse(ungroundedJSON, parseReq, "test-model")
	require.NoError(t, err)
	require.Equal(t, ruletypes.AIStrategyStatusLowConfidence, ungrounded.Status,
		"ungrounded ready draft must be downgraded to low_confidence")
	require.Equal(t, ruletypes.AIConfidenceLow, ungrounded.Confidence)
	require.NotEmpty(t, ungrounded.Limitations,
		"downgrade must explain the missing SOP grounding")
}

func TestParse_AuditFieldsPopulated(t *testing.T) {
	strategy, err := Parse(happyJSON, parseReq, "gpt-4o-mini")
	require.NoError(t, err)
	require.Equal(t, "gpt-4o-mini", strategy.Audit.Model)
	require.Equal(t, PromptVersion, strategy.Audit.PromptVersion)
	require.NotEmpty(t, strategy.Audit.GeneratedAt)
}
