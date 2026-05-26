package llmaigenerator

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// stubProvider is a minimal in-test Provider implementation.
type stubProvider struct {
	response string
	err      error
	delay    time.Duration
}

func (s *stubProvider) Complete(ctx context.Context, _, _ string) (string, error) {
	if s.delay > 0 {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(s.delay):
		}
	}
	return s.response, s.err
}

// llmTestReq is a minimal ready-strategy request for Generator tests.
var llmTestReq = ruletypes.AIStrategyRequest{
	IncidentID:       "INC-LLM-001",
	AlertFingerprint: "fp-llm",
	SOPDocument: ruletypes.SOPDocument{
		SOPID:   "SOP-LLM-001",
		Version: "2026-05-20.1",
	},
	EvidenceRefs: []ruletypes.AIEvidenceRef{
		{
			RefID:       "metric:err:llm-svc",
			Type:        "metric",
			Observation: "error rate elevated",
			Confidence:  "medium",
		},
	},
}

const llmHappyJSON = `{
  "headline": "LLM 테스트 헤드라인",
  "hypotheses": [
    {
      "rank": 1,
      "text": "LLM 가설 텍스트",
      "confidence": "medium",
      "evidenceRefs": ["metric:err:llm-svc"]
    }
  ],
  "firstActions": [
    {
      "text": "LLM 첫 조치",
      "sopStepRef": "SOP-LLM-001#1",
      "requiresHumanApproval": true
    }
  ],
  "confidence": "medium",
  "status": "ready"
}`

// TestGenerator_ImplementsInterface verifies the compile-time assertion holds.
func TestGenerator_ImplementsInterface(t *testing.T) {
	var _ ruletypes.AIStrategyGenerator = (*Generator)(nil)
}

func TestGenerator_NilProvider_Errors(t *testing.T) {
	gen := New(nil, "m", time.Second)
	_, err := gen.Generate(context.Background(), llmTestReq)
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider must not be nil")
}

func TestGenerator_HappyPath(t *testing.T) {
	stub := &stubProvider{response: llmHappyJSON}
	gen := New(stub, "claude-sonnet-4", 5*time.Second)
	strategy, err := gen.Generate(context.Background(), llmTestReq)
	require.NoError(t, err)
	require.Equal(t, "LLM 테스트 헤드라인", strategy.Headline)
	require.Equal(t, "claude-sonnet-4", strategy.Audit.Model)
	require.Equal(t, PromptVersion, strategy.Audit.PromptVersion)
	require.Equal(t, ruletypes.AIStrategyStatusReady, strategy.Status)
}

func TestGenerator_ProviderError_Wraps(t *testing.T) {
	originalErr := errors.New("upstream connection refused")
	stub := &stubProvider{err: originalErr}
	gen := New(stub, "model-x", 5*time.Second)
	_, err := gen.Generate(context.Background(), llmTestReq)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "upstream connection refused"),
		"error should wrap original provider error, got: %s", err.Error())
}

func TestGenerator_RespectsTimeout(t *testing.T) {
	stub := &stubProvider{delay: 100 * time.Millisecond, response: llmHappyJSON}
	gen := New(stub, "model-x", 10*time.Millisecond)
	_, err := gen.Generate(context.Background(), llmTestReq)
	require.Error(t, err)
	require.True(t,
		strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "deadline"),
		"expected deadline exceeded error, got: %s", err.Error())
}
