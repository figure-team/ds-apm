package signozruler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/aiconfigstoretest"
	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// stubGen is a minimal AIStrategyGenerator that always returns the canned
// AIStrategy supplied at construction time. It proves that the handler
// delegates generation to the injected generator rather than calling the
// free function directly.
type stubGen struct{ out ruletypes.AIStrategy }

func (s stubGen) Generate(_ context.Context, _ ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	return s.out, nil
}

// newAIConfigTestHandler returns a handler wired with a fresh in-memory
// aiconfigstoretest.Fake and a PlaintextCipher. The store is also returned so
// callers can seed it directly.
func newAIConfigTestHandler(t *testing.T) (*aiconfigstoretest.Fake, *handler) {
	t.Helper()
	store := aiconfigstoretest.New()
	cipher := secretbox.PlaintextCipher()
	h := &handler{
		sopStore:       newMemSOPStore(),
		aiHistoryStore: newMemAIHistoryStore(),
		aiGenerator:    nil,
		aiConfigStore:  store,
		aiCipher:       cipher,
	}
	return store, h
}

// identityEncrypt and identityDecrypt are the PlaintextCipher functions for
// direct store seeding in tests.
func identityEncrypt(s string) (string, error) { return s, nil }
func identityDecrypt(s string) (string, error) { return s, nil }

func TestPreviewAIStrategy_UsesInjectedGenerator(t *testing.T) {
	canned := ruletypes.AIStrategy{
		ContractVersion:  ruletypes.AIStrategyContractVersion,
		StrategyID:       "strat-mock",
		IncidentID:       "INC-1",
		AlertFingerprint: "fp-1",
		Status:           ruletypes.AIStrategyStatusReady,
		Language:         "ko-KR",
		Confidence:       "medium",
		Audit: ruletypes.AIStrategyAudit{
			PromptVersion:    "mock",
			Model:            "mock",
			GeneratedAt:      "2026-05-20T09:00:00Z",
			RedactionApplied: true,
		},
	}

	h := &handler{
		sopStore:       newMemSOPStore(),
		aiHistoryStore: newMemAIHistoryStore(),
		aiGenerator:    stubGen{out: canned},
	}

	body, err := json.Marshal(ruletypes.AIStrategyRequest{
		IncidentID:       "INC-1",
		AlertFingerprint: "fp-1",
		Labels:           map[string]string{"alertname": "X", "service.name": "Y"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/ai/strategy/preview", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	rr := httptest.NewRecorder()

	h.PreviewAIStrategy(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	var got struct {
		Data ruletypes.AIStrategy `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, "strat-mock", got.Data.StrategyID)
}

func TestUpdateAIConfig_PreservesOAuthTokenOnSentinel(t *testing.T) {
	store, h := newAIConfigTestHandler(t)

	ctx := withSOPTestClaims(httptest.NewRequest("GET", "/", nil)).Context()
	orgID := "00000000-0000-0000-0000-000000000001"

	// Seed with a real token.
	if err := store.Upsert(ctx, ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           orgID,
		Provider:        "llm",
		LLMProvider:     "claude",
		Transport:       "cli",
		OAuthToken:      "preserved-token",
		UpdatedAt:       "2026-05-21T00:00:00Z",
	}, identityEncrypt); err != nil {
		t.Fatalf("seed: %v", err)
	}

	body := strings.NewReader(`{
		"contractVersion":"ds.ai_config.v1",
		"provider":"llm","llmProvider":"claude","transport":"cli",
		"oauthToken":"<unchanged>"
	}`)
	req := httptest.NewRequest("PUT", "/api/v2/ds/ai/config", body).WithContext(ctx)
	rw := httptest.NewRecorder()
	h.UpdateAIConfig(rw, req)
	if rw.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, body=%s", rw.Code, rw.Body.String())
	}

	got, err := store.Get(ctx, orgID, identityDecrypt)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.OAuthToken != "preserved-token" {
		t.Fatalf("OAuthToken: got %q, want %q", got.OAuthToken, "preserved-token")
	}
}

// stubRebuilder is a minimal aiGeneratorRebuilder for use in tests that need to
// intercept the AIConfig passed to GeneratorFor.
type stubRebuilder struct {
	gen                  ruletypes.AIStrategyGenerator
	lastConfigOAuthToken string
}

func (s *stubRebuilder) Invalidate(_ string) {}

func (s *stubRebuilder) GeneratorFor(cfg ruletypes.AIConfig) (ruletypes.AIStrategyGenerator, error) {
	s.lastConfigOAuthToken = cfg.OAuthToken
	return s.gen, nil
}

// stubGenerator is a pointer-receiver AIStrategyGenerator that returns a
// canned strategy. Unlike the value-receiver stubGen, it is addressable and
// can capture state via GeneratorFor.
type stubGenerator struct {
	strategy ruletypes.AIStrategy
}

func (s *stubGenerator) Generate(_ context.Context, _ ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	return s.strategy, nil
}

func TestTestAIConfig_SubstitutesOAuthTokenSentinel(t *testing.T) {
	store, h := newAIConfigTestHandler(t)
	// Stub the rebuilder so the test exercises only the sentinel-substitution
	// path, not real CLI execution.
	gen := &stubGenerator{strategy: ruletypes.AIStrategy{Headline: "ok", Audit: ruletypes.AIStrategyAudit{Model: "claude-test"}}}
	rb := &stubRebuilder{gen: gen}
	h.aiRebuilder = rb

	ctx := withSOPTestClaims(httptest.NewRequest("POST", "/api/v2/ds/ai/config/test", nil)).Context()
	if err := store.Upsert(ctx, ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           "00000000-0000-0000-0000-000000000001",
		Provider:        "llm",
		LLMProvider:     "claude",
		Transport:       "cli",
		OAuthToken:      "real-token",
		UpdatedAt:       "2026-05-21T00:00:00Z",
	}, identityEncrypt); err != nil {
		t.Fatalf("seed: %v", err)
	}

	body := strings.NewReader(`{
		"contractVersion":"ds.ai_config.v1",
		"provider":"llm","llmProvider":"claude","transport":"cli",
		"oauthToken":"<unchanged>"
	}`)
	req := httptest.NewRequest("POST", "/api/v2/ds/ai/config/test", body)
	req = withSOPTestClaims(req)
	rw := httptest.NewRecorder()
	h.TestAIConfig(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rw.Code, rw.Body.String())
	}

	// The stub rebuilder captures the AIConfig that was actually used.
	if rb.lastConfigOAuthToken != "real-token" {
		t.Fatalf("OAuthToken substitution failed: got %q, want %q",
			rb.lastConfigOAuthToken, "real-token")
	}
}

func TestGetAIConfig_ScrubsOAuthToken(t *testing.T) {
	store, h := newAIConfigTestHandler(t)

	ctx := withSOPTestClaims(httptest.NewRequest("GET", "/", nil)).Context()
	orgID := "00000000-0000-0000-0000-000000000001"

	_ = store.Upsert(ctx, ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           orgID,
		Provider:        "llm",
		LLMProvider:     "claude",
		Transport:       "cli",
		OAuthToken:      "real-token-do-not-leak",
		UpdatedAt:       "2026-05-21T00:00:00Z",
	}, identityEncrypt)

	req := httptest.NewRequest("GET", "/api/v2/ds/ai/config", nil).WithContext(ctx)
	rw := httptest.NewRecorder()
	h.GetAIConfig(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rw.Code, rw.Body.String())
	}
	if strings.Contains(rw.Body.String(), "real-token-do-not-leak") {
		t.Fatalf("OAuthToken leaked in response: %s", rw.Body.String())
	}
	// Unmarshal and check the field value directly — JSON encodes '<' as <.
	var envelope struct {
		Data ruletypes.AIConfig `json:"data"`
	}
	if err := json.Unmarshal(rw.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if envelope.Data.OAuthToken != APIKeyPlaceholder {
		t.Fatalf("expected OAuthToken to be %q sentinel, got %q", APIKeyPlaceholder, envelope.Data.OAuthToken)
	}
}
