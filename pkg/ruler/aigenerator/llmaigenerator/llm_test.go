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

// sequenceProvider returns responses[i] on the i-th Complete call (clamping to
// the last entry) and records how many times it was called.
type sequenceProvider struct {
	responses []string
	calls     int
}

func (s *sequenceProvider) Complete(_ context.Context, _, _ string) (string, error) {
	i := s.calls
	s.calls++
	if i >= len(s.responses) {
		i = len(s.responses) - 1
	}
	return s.responses[i], nil
}

func TestGenerator_RetriesOnParseFailure(t *testing.T) {
	// Two unparseable completions, then a valid one: Generate must keep trying
	// and succeed on the third attempt.
	stub := &sequenceProvider{responses: []string{"not json", "{still not valid}", llmHappyJSON}}
	gen := New(stub, "model-x", 5*time.Second)
	strategy, err := gen.Generate(context.Background(), llmTestReq)
	require.NoError(t, err)
	require.Equal(t, "LLM 테스트 헤드라인", strategy.Headline)
	require.Equal(t, 3, stub.calls, "should retry until a valid completion")
}

func TestGenerator_FailsAfterMaxParseAttempts(t *testing.T) {
	// Every completion is unparseable: Generate gives up after maxParseAttempts.
	stub := &sequenceProvider{responses: []string{"never valid"}}
	gen := New(stub, "model-x", 5*time.Second)
	_, err := gen.Generate(context.Background(), llmTestReq)
	require.Error(t, err)
	require.Equal(t, maxParseAttempts, stub.calls, "should stop after maxParseAttempts")
}

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
