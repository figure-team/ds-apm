package aigenerator

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// TestNew_LLMEndpointOverrideRoutesToConfiguredServer pins that Config.LLMEndpoint
// is honored: an llm/api generator must send its request to the configured
// endpoint (e.g. a wiremock mock) instead of the hardcoded provider default.
// This is the plumbing the T1 integration test relies on.
func TestNew_LLMEndpointOverrideRoutesToConfiguredServer(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Header().Set("content-type", "application/json")
		// Anthropic Messages API shape: the draft JSON lives in content[].text.
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"{\"headline\":\"PG timeout 우선 확인\",\"hypotheses\":[{\"rank\":1,\"text\":\"외부 PG 지연\",\"confidence\":\"medium\",\"evidenceRefs\":[\"metric:err\"],\"sopStepRefs\":[\"SOP-1#1\"]}],\"firstActions\":[{\"text\":\"PG 로그 확인\",\"sopStepRef\":\"SOP-1#1\",\"requiresHumanApproval\":true}],\"confidence\":\"medium\",\"status\":\"ready\"}"}]}`)
	}))
	defer srv.Close()

	gen, err := New(Config{
		Provider:          "llm",
		LLMProvider:       "claude",
		LLMTransport:      "api",
		LLMAPIKey:         "test-key",
		LLMEndpoint:       srv.URL,
		LLMTimeoutSeconds: 5,
	})
	require.NoError(t, err)

	_, err = gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "INC-1",
		AlertFingerprint: "fp-1",
		SOPDocument:      ruletypes.SOPDocument{SOPID: "SOP-1", Version: "v1"},
		EvidenceRefs: []ruletypes.AIEvidenceRef{
			{RefID: "metric:err", Type: "metric", Observation: "elevated", Confidence: "medium"},
		},
	})
	require.NoError(t, err)
	require.True(t, hit, "generate must reach the configured LLM endpoint")
}
